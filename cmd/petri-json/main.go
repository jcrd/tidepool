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
)

func main() {
    w := flag.Int("width", 256, "Environment width")
    h := flag.Int("height", 256, "Environment height")
    g := flag.Int("genome", 1024, "Genome size")
    s := flag.Int64("seed", -1, "Environment seed")
    t := flag.Duration("tick", time.Millisecond, "Clock tick frequency")

    flag.Parse()

    env := petri.NewEnv(int32(*w), int32(*h), int32(*g), *s)
    dts := make(chan *petri.Delta)

    go env.Run(runtime.NumCPU(), *t, dts)

    for dt := range dts {
        json, err := json.Marshal(dt)
        if err != nil {
            fmt.Fprintln(os.Stderr, err)
            os.Exit(1)
        }
        fmt.Println(string(json))
    }
}
