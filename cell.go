// This project is licensed under the MIT License (see LICENSE).

package petri

import (
    "petri/gene"
)

const genomeStartIdx = 1

type Genome []gene.Gene

type Cell struct {
    idx int32
    ID int64
    Origin int64
    Parent int64
    Generation int64
    Energy int
    X int32
    Y int32
    Genome Genome
}

type Stats struct {
    GeneExecN [gene.N]int
    ViableCellKilled int
    ViableCellShared int
}

type Delta struct {
    Cells []*Cell
    Stats Stats
}

func newCell(idx, x, y, g int32) *Cell {
    c := &Cell{
        idx: idx,
        X: x,
        Y: y,
        Genome: make(Genome, g),
    }
    c.resetGenome()

    return c
}

func newDelta() *Delta {
    return &Delta{
        Cells: make([]*Cell, 0),
        Stats: Stats{},
    }
}

func (c *Cell) randomizeGenome(ctx *Context) {
    for i := range c.Genome {
        c.Genome[i] = ctx.getRandomGene()
    }
}

func (c *Cell) resetGenome() {
    for i := range c.Genome {
        c.Genome[i] = gene.STOP
    }
}

func (c *Cell) clone() *Cell {
    n := newCell(c.idx, c.X, c.Y, int32(len(c.Genome)))

    n.ID = c.ID
    n.Origin = c.Origin
    n.Parent = c.Parent
    n.Generation = c.Generation
    n.Energy = c.Energy

    for i, v := range c.Genome {
        n.Genome[i] = v
    }

    return n
}

func (c *Cell) logo() gene.Gene {
    return c.Genome[0]
}

func (c *Cell) exec(ctx *Context) *Delta {
    return ctx.vm.exec(c)
}

func (c *Cell) resetMetadata(ctx *Context) {
    c.ID = ctx.env.getNextCellID()
    c.Origin = c.ID
    c.Parent = 0
    c.Generation = 0
}

func (c *Cell) seed(ctx *Context) *Delta {
    c.resetMetadata(ctx)
    c.randomizeGenome(ctx)
    c.Energy += ctx.env.GetRNG().Energy(ctx)

    dt := newDelta()
    dt.Cells = append(dt.Cells, c)

    return dt
}

func (c *Cell) accessible(ctx *Context, g gene.Gene, x gene.Gene) bool {
    return ctx.env.GetRNG().CellAccessible(ctx, c, g, x)
}
