// This project is licensed under the MIT License (see LICENSE).

package main

import (
    "encoding/json"
    "flag"
    "fmt"
    "os"
    "runtime"
    "time"

    "petri"

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
    w := flag.Int("width", 256, "Environment width")
    h := flag.Int("height", 256, "Environment height")
    g := flag.Int("genome", 1024, "Genome size")
    p := flag.Float64("pop", 0.01, "Initial population percent")
    s := flag.Int64("seed", -1, "Environment seed")
    t := flag.Duration("tick", time.Millisecond, "Clock tick frequency")

    flag.Parse()

    pop := int32(*p * float64(*w * *h))
    env := petri.NewEnv(int32(*w), int32(*h), int32(*g), pop, *s)

    dts := make(chan *petri.Delta)

    go env.Run(runtime.NumCPU(), *t, dts)

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
