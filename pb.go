package petri

import (
    "petri/pb"
)

func (c *Cell) ProtobufMessage() *pb.Cell {
    m := &pb.Cell{
        Idx: c.Idx,
        ID: c.ID,
        Origin: c.Origin,
        Parent: c.Parent,
        Generation: c.Generation,
        Energy: c.Energy,
        X: c.X,
        Y: c.Y,
    }
    for _, g := range c.Genome {
        m.Genome = append(m.Genome, int32(g))
    }
    return m
}

func (dt *Delta) ProtobufMessage() *pb.Delta {
    m := &pb.Delta{
        Stats: dt.Stats,
    }
    for _, c := range dt.Cells {
        m.Cells = append(m.Cells, c.ProtobufMessage())
    }
    return m
}
