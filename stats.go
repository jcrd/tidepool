package petri

import (
    "petri/gene"
)

type Stats struct {
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

func (s *Stats) Add(a *Stats) {
    for i := range a.GeneExecN {
        s.GeneExecN[i] += a.GeneExecN[i]
    }
    for n, i := range a.ByName {
        s.inc(n, i)
    }
}
