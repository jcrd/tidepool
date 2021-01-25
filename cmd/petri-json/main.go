// This project is licensed under the MIT License (see LICENSE).

package main

import (
    "encoding/json"
    "fmt"
    "os"

    "petri"
    "petri/cmd"

    "github.com/eiannone/keyboard"
)

func printDelta(dt *petri.Delta) {
    json, err := json.Marshal(dt)
    if err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
    fmt.Println(string(json))
}

func main() {
    env, dts := cmd.ParseAndRun()

    keyEvents, err := keyboard.GetKeys(1)
    if err != nil {
        fmt.Fprintln(os.Stderr, "keyboard: failed to initialize")
        os.Exit(1)
    }
    defer keyboard.Close()

    for {
        select {
        case ev := <-keyEvents:
            if ev.Err != nil {
                fmt.Fprintf(os.Stderr, "keyboard: %s\n", ev.Err)
                break
            }
            switch ev.Key {
            case keyboard.KeyEsc:
                fallthrough
            case keyboard.KeyCtrlC:
                env.Stop()
            }
        case dt, ok := <-dts:
            if !ok {
                return
            }
            printDelta(dt)
        }
    }
}
