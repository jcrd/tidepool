// This project is licensed under the MIT License (see LICENSE).

package tidepool

import (
    "encoding/json"
    "testing"
)

func BenchmarkJSONMarshalCells(b *testing.B) {
    b.ReportAllocs()

    var (
        w int32 = 64
        h int32 = 64
        gs int32 = 1024

        stats = Stats{
            "LiveCellsKilled": 10000,
            "LiveCellsShared": 10000,
            "CellsKilled": 10000,
            "CellsShared": 10000,
            "ReproductionAttempts": 10000,
            "Reproductions": 1000,
            "Mutations": 10,
        }
    )

    env := NewEnv(w, h, gs, 0, -1)
    ret := make(chan *Delta)
    defer close(ret)

    go func() {
        env.WithCells <- func(cs []*Cell) {
            ret <- &Delta{
                Cells: cs,
                Stats: stats,
            }
        }
    }()

    dt := <-ret
    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        json.Marshal(dt)
    }
}
