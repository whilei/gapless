// Package gapless is a highly available Apple push notification service.
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

// Global settings which are loaded from a json file passed via cmd line.
var Settings = NewSettingsObj()

var stdout = log.New(os.Stdout, "[Gapless I] ", log.Ldate|log.Ltime)
var stderr = log.New(os.Stderr, "[Gapless E] ", log.Ldate|log.Ltime|log.Lshortfile)

type gapObj struct {
    token      []byte
    identifier uint32
    expiry     time.Duration
    jData      []byte
}

// Main run function. This will listen to our redis connection indefinitely.
// If redis were to crash, this application will panic and exit.
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
    err := connPool.InitPool(Settings.Int("pool_size", 2), Settings.String("apns_server"), apnsCert, apnsKey)
    if err != nil {
        stderr.Fatalf("Connection pool failed to initialize: %s.", err)
    }

    // Clean up our connection pool when exiting.
    defer connPool.ShutdownConns()

    // Init our redis connection.
    redis := godis.New(Settings.String("redis_netaddress"), Settings.Int("redis_db", 0), Settings.String("redis_password"))
    _, err = redis.Ping()
    if err != nil {
        stderr.Fatalf("Redis failed to initialize: %s.", err)
    }

    // Again, clean up our connection when exiting.
    defer redis.Quit()

    // The redis queue key to be used.
    queueKey := Settings.String("redis_queue_key", "")
    if queueKey == "" {
        stderr.Fatalf("The 'redis_queue_key' must be defined in your settings.")
    }

    logSuccesses := Settings.Bool("log_successes", false)

    // Energizer loop.
    for {
        // List to our redis list, one item at a time.
        item, err := redis.Blpop([]string{queueKey}, 0)
        if err != nil {
            stderr.Fatalf("Redis BLPOP failed: %s.", err)
        }

        // Grab the string out.
        raw := item.StringMap()[queueKey]

        // We grab a connection from the pool.
        // This call will block until a connection is available again.
        // If your still getting back logged, increase your pool size.
        conn := connPool.GetConn()

        // Process the string in a goroutine.
        go func(input string, apns *apnsConn) {
            // Ensure to return the connection back to the pool when done here.
            defer connPool.ReleaseConn(apns)

            jsonIn := make(map[string]interface{})
            err := json.Unmarshal([]byte(input), &jsonIn)
            if err != nil {
                // If an error occurs while reading the json, ignore this item and continue on.
                stderr.Printf("Json unmarshal error (%s): %s.", input, err)
                return
            }

            gapOut, err := parseApnsJson(jsonIn)
            if err != nil {
                // If an error occurs while reading the json, ignore this item and continue on.
                stderr.Printf("Parsing apns structure error (%q): %s.", jsonIn, err)
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

                    stdout.Printf("SendPayload Error (ID %d): %s. Retrying count (1).", gapOut.identifier, err)

                    _, err = redis.Lpush(queueKey, retryPayload)
                    if err != nil {
                        stderr.Printf("Redis LPUSH failed (%v): %s.", retryPayload, err)
                        return
                    }
                } else if uint32(result.(float64)) < 3 {
                    // Have not retried... add _gapless_RETRYING key and send it back.
                    jsonIn["_gapless_RETRYING"] = uint32(result.(float64)) + 1
                    retryPayload, _ := json.Marshal(jsonIn)

                    stdout.Printf("SendPayload Error (ID %d): %s. Retrying count (%d).", gapOut.identifier, err, jsonIn["_gapless_RETRYING"])
                    _, err = redis.Lpush(queueKey, retryPayload)
                    if err != nil {
                        stderr.Printf("Redis LPUSH failed (%v): %s.", retryPayload, err)
                        return
                    }
                } else {
                    stdout.Printf("Final SendPayload Error (ID %d): %s | %s.", gapOut.identifier, err, jsonIn)
                }
            } else if logSuccesses {
                stdout.Printf("Sent: %s.", input)
            }

        }(raw, conn)
    }
}

func parseApnsJson(in map[string]interface{}) (*gapObj, error) {
    gap := new(gapObj)
    var err error

    // Token
    result, present := in["token"]
    if !present {
        return gap, errors.New(fmt.Sprintf("Json Data Error (%v): Token was missing.", in))
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
        return gap, errors.New(fmt.Sprintf("Missing data structure: %v.", in))
    }
    data := result.(map[string]interface{})

    // Wrap it back up.
    gap.jData, err = json.Marshal(data)
    if err != nil {
        return gap, err
    }

    return gap, nil
}
