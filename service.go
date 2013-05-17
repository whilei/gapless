// Package gocov1 is a code coverage analysis tool for Go.
package gapless

import (
    "encoding/hex"
    "encoding/json"
    "errors"
    "fmt"
    godis "github.com/cojac/godis/redis"
    "log"
    "os"
    "path/filepath"
    "time"
)

var Settings = NewSettingsObj()
var stdout = log.New(os.Stdout, "[Gapless I] ", log.Ldate|log.Ltime)
var stderr = log.New(os.Stderr, "[Gapless E] ", log.Ldate|log.Ltime|log.Lshortfile)

type GapObj struct {
    token      []byte
    identifier uint32
    expiry     time.Duration
    jData      []byte
}

// Main run function. This will listen to our redis connection indefinitely.
func Run() {
    // Prep our certificate file paths.
    apnsCert := Settings.String("apns_cert_path")
    if !filepath.IsAbs(apnsCert) {
        apnsCert = filepath.Dir(Settings.ConfFile) + "/" + apnsCert
    }
    apnsKey := Settings.String("apns_key_path")
    if !filepath.IsAbs(apnsKey) {
        apnsKey = filepath.Dir(Settings.ConfFile) + "/" + apnsKey
    }

    // Initialize the pool of APNS connections.
    err := ConnPool.InitPool(Settings.Int("pool_size", 2), Settings.String("apns_server"), apnsCert, apnsKey)
    if err != nil {
        stderr.Fatalln("Connection pool failed to initialize:", err)
    }

    // Clean up our connection pool when exiting.
    defer ConnPool.ShutdownConns()

    // Init our redis connection.
    redis := godis.New(Settings.String("redis_netaddress"), Settings.Int("redis_db", 0), Settings.String("redis_password"))
    _, err = redis.Ping()
    if err != nil {
        stderr.Fatalln("Redis failed to initialize:", err)
    }

    // Again, clean up our connection when exiting.
    defer redis.Quit()

    // The redis queue key to be used.
    queueKey := Settings.String("redis_queue_key", "")
    if queueKey == "" {
        stderr.Fatalln("The 'redis_queue_key' must be defined in your settings.")
    }

    // Energizer loop.
    for {
        // List to our redis list, one item at a time.
        item, err := redis.Blpop([]string{queueKey}, 0)
        if err != nil {
            stderr.Fatalln("Redis BLPOP failed:", err)
        }

        // Grab the string out.
        raw := item.StringMap()[queueKey]

        // We grab a connection from the pool.
        // This call will block until a connection is available again.
        // If your still getting back logged, increase your pool size.
        conn := ConnPool.GetConn()

        // Process the string in a goroutine.
        go func(v string, apns *ApnsConn) {
            // Ensure to return the connection back to the pool when done here.
            defer ConnPool.ReleaseConn(apns)

            jsonIn := make(map[string]interface{})
            err := json.Unmarshal([]byte(v), &jsonIn)
            if err != nil {
                // If an error occurs while reading the json, ignore this item and continue on.
                stderr.Printf("Json unmarshal error (%s): %s", v, err)
                return
            }

            gapOut, err := parseApnsJson(jsonIn)
            if err != nil {
                // If an error occurs while reading the json, ignore this item and continue on.
                stderr.Printf("Parsing apns structure error (%q): %s", jsonIn, err)
                return
            }

            // Send the payload out.
            err = apns.SendPayload(gapOut.token, gapOut.jData, gapOut.expiry, gapOut.identifier)

            // If we get an error, we will retry.
            if err != nil {
                // Have we retried yet?
                result, present := jsonIn["_gapless_RETRYING"]
                if !present {
                    // Have not retried... add _gapless_RETRYING key and send it back.
                    jsonIn["_gapless_RETRYING"] = 1
                    retryPayload, _ := json.Marshal(jsonIn)

                    stdout.Printf("SendPayload Error (ID %d): %s. Retrying count (1).\n", gapOut.identifier, err)

                    _, err = redis.Lpush(queueKey, retryPayload)
                    if err != nil {
                        stderr.Printf("Redis LPUSH failed (%v): %s", retryPayload, err)
                        return
                    }
                } else if uint32(result.(float64)) < 3 {
                    // Have not retried... add _gapless_RETRYING key and send it back.
                    jsonIn["_gapless_RETRYING"] = uint32(result.(float64)) + 1
                    retryPayload, _ := json.Marshal(jsonIn)

                    stdout.Printf("SendPayload Error (ID %d): %s. Retrying count (%d).\n", gapOut.identifier, err, jsonIn["_gapless_RETRYING"])
                    _, err = redis.Lpush(queueKey, retryPayload)
                    if err != nil {
                        stderr.Printf("Redis LPUSH failed (%v): %s", retryPayload, err)
                        return
                    }
                } else {
                    stdout.Printf("Final SendPayload Error (ID %d): %s | %s\n", gapOut.identifier, err, jsonIn)
                }
            } else if Settings.Bool("log_successes", false) {
                stdout.Printf("Sent: %s\n", gapOut.jData)
            }

        }(raw, conn)
    }
}

func parseApnsJson(in map[string]interface{}) (*GapObj, error) {
    gap := new(GapObj)
    var err error

    // Token
    result, present := in["token"]
    if !present {
        return gap, errors.New(fmt.Sprintf("Json Data Error (%v): Token was missing.\n", in))
    }
    gap.token, err = hex.DecodeString(result.(string))
    if err != nil {
        return gap, err
    }

    // Identifier
    result, present = in["identifier"]
    if !present {
        result = 0
    }
    gap.identifier = uint32(result.(float64))

    // Notification - Expiry
    result, present = in["expiry"]
    if !present {
        result = 7200
    }
    gap.expiry = time.Duration(uint32(result.(float64))) * time.Second

    // Notification - Data
    result, present = in["data"]
    if !present {
        return gap, errors.New(fmt.Sprintf("Missing data structure: %v", in))
    }
    data := result.(map[string]interface{})

    // Wrap it back up.
    gap.jData, err = json.Marshal(data)
    if err != nil {
        return gap, err
    }

    return gap, nil
}
