// This project is licensed under the MIT License (see LICENSE).

package petri

import (
    "encoding/json"
    "testing"

    "google.golang.org/protobuf/proto"
)

func BenchmarkMarshalDelta(b *testing.B) {
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

    var dt *Delta
    env.WithCells(func(cs []*Cell) {
        dt = &Delta{
            Cells: cs,
            Stats: stats,
        }
    })

    b.ResetTimer()

    b.Run("json", func(b *testing.B) {
        b.ReportAllocs()
        for i := 0; i < b.N; i++ {
            json.Marshal(dt)
        }
    })

    b.Run("protobuf", func(b *testing.B) {
        b.ReportAllocs()
        for i := 0; i < b.N; i++ {
            proto.Marshal(dt.ProtobufMessage())
        }
    })
}
