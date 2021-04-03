package cmd

import (
    "flag"
    "runtime"
    "time"

    tp "tidepool"
)

func ParseAndRun() (*tp.Env, <-chan *tp.Delta) {
    w := flag.Int("width", 256, "Environment width")
    h := flag.Int("height", 256, "Environment height")
    g := flag.Int("genome", 1024, "Genome size")
    p := flag.Float64("pop", 0.01, "Initial population percent")
    s := flag.Int64("seed", -1, "Environment seed")
    t := flag.Duration("tick", time.Millisecond, "Clock tick frequency")

    flag.Parse()

    pop := int32(*p * float64(*w * *h))
    env := tp.NewEnv(int32(*w), int32(*h), int32(*g), pop, *s)

    dts := make(chan *tp.Delta)

    go env.Run(runtime.NumCPU(), *t, dts)

    return env, dts
}
