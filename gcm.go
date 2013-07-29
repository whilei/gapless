// Package gapless is a highly available Apple push notification service.
package gapless

import (
    "encoding/json"
    "github.com/alexjlockwood/gcm"
    "log"
    "os"
)

var gcmout = log.New(os.Stdout, "[Gapless-GCM I] ", log.Ldate|log.Ltime)
var gcmerr = log.New(os.Stderr, "[Gapless-GCM E] ", log.Ldate|log.Ltime|log.Lshortfile)

// Main run function. This will listen to our redis connection indefinitely.
// If redis were to crash, this application will panic and exit.
func RunGcm() {
    // Init our redis connections.
    redisClient := newRedisConn()

    // Again, clean up our connection when exiting.
    defer redisClient.Quit()

    // The redis queue key to be used.
    queueKey := Settings.String("redis_gcm_queue_key", "")
    if queueKey == "" {
        gcmerr.Fatalf("The 'redis_gcm_queue_key' must be defined in your settings.")
    }

    logSuccesses := Settings.Bool("log_successes", false)
    apiKey := Settings.String("gcm_api_key", "")

    // Energizer loop.
    for {
        // List to our redis list, one item at a time.
        item, err := redisClient.BLPop(0, "", queueKey)
        if err != nil {
            gcmerr.Fatalf("Redis BLPop failed: %s. (%v)", err, item)
        }

        // Grab the string out.
        raw := item[1]

        // Process the string in a goroutine.
        go func(input string) {
            jsonIn := make(map[string]interface{})
            err := json.Unmarshal([]byte(input), &jsonIn)
            if err != nil {
                // If an error occurs while reading the json, ignore this item and continue on.
                gcmerr.Printf("Json unmarshal error (%s): %s.", input, err)
                return
            }

            // Id
            id, present := jsonIn["id"]
            if !present {
                gcmerr.Printf("Json Data Error (%v): Id was missing.", input)
                return
            }

            // Data
            data, present := jsonIn["data"]
            if !present {
                gcmerr.Printf("Json Data Error (%v): Data was missing.", input)
                return
            }

            // Create the message to be sent.
            regIds := []string{id.(string)}
            msg := gcm.NewMessage(data.(map[string]string), regIds...)

            // Create a Sender to send the message.
            sender := &gcm.Sender{apiKey, nil}

            // Send the message and receive the response after at most two retries.
            response, err := sender.Send(msg, 2)
            if err != nil {
                gcmout.Printf("Failed to send message: %s", err)
                return
            }

            if logSuccesses {
                gcmout.Printf("Sent: %s |%q.", input, response)
            }

        }(raw)
    }
}
