package main

import (
    "fmt"
    "github.com/cojac/gapless"
    "os"
    "path/filepath"
)

func main() {
    // Config file is mandatory. Ensure one is passed in.
    if len(os.Args) < 2 {
        fmt.Printf("Usage: %s <config-path>\n", filepath.Base(os.Args[0]))
        os.Exit(1)
    }

    // Tell gapless about our settings file.
    gapless.Settings.LoadFromFile(filepath.Clean(os.Args[1]))

    // Start the connections.
    gapless.Run()
}
