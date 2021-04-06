// This project is licensed under the MIT License (see LICENSE).

package main

import (
    "encoding/json"
    "fmt"
    "os"
    "os/signal"

    "tidepool/cmd"
)

func main() {
    env, dts := cmd.ParseAndRun()

    sig := make(chan os.Signal, 1)
    signal.Notify(sig, os.Interrupt)
    defer signal.Stop(sig)

    for {
        select {
        case <-sig:
            env.Stop()
        case dt, ok := <-dts:
            if !ok {
                return
            }
            json, err := json.Marshal(dt)
            if err != nil {
                fmt.Fprintln(os.Stderr, err)
                os.Exit(1)
            }
            fmt.Println(string(json))
        }
    }
}
