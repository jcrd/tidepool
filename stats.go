package petri

import (
    "petri/gene"
)

type Stats struct {
    Ticks int64
    GeneExecN [gene.N]int
    ByName map[string]int64
}

func NewStats() *Stats {
    return &Stats{
        ByName: make(map[string]int64),
    }
}

func (s *Stats) inc(name string, i int64) {
    if v, ok := s.ByName[name]; ok {
        i = i + v
    }
    s.ByName[name] = i
}

func (s *Stats) update(name string, i int64) {
    if max, ok := s.ByName[name]; !ok || i > max {
        s.ByName[name] = i
    }
}

func (s *Stats) Add(a *Stats) {
    if a.Ticks > s.Ticks {
        s.Ticks = a.Ticks
    }
    for i := range a.GeneExecN {
        s.GeneExecN[i] += a.GeneExecN[i]
    }
    for n, i := range a.ByName {
        switch n {
        case "MaxGeneration":
            s.update(n, i)
        default:
            s.inc(n, i)
        }
    }
}
