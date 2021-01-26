package petri

import (
    "petri/gene"
)

type Stats struct {
    GeneExecN [gene.N]int
    CellKilled int
    CellShared int
    ViableCellKilled int
    ViableCellShared int
}

func (s *Stats) Add(a *Stats) {
    for i := range a.GeneExecN {
        s.GeneExecN[i] += a.GeneExecN[i]
    }
    s.CellKilled += a.CellKilled
    s.CellShared += a.CellShared
    s.ViableCellKilled += a.ViableCellKilled
    s.ViableCellShared += a.ViableCellShared
}
