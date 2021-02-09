// This project is licensed under the MIT License (see LICENSE).

package petri

import (
    "petri/gene"
)

const genomeStartIdx = 1

type Cell struct {
    Idx int32
    ID int64
    Origin int64
    Parent int64
    Generation int64
    Energy int64
    X int32
    Y int32
    Genome gene.Genome
}

type Delta struct {
    Cells []*Cell
    Stats Stats
}

func newCell(idx, x, y, g int32) *Cell {
    c := &Cell{
        Idx: idx,
        X: x,
        Y: y,
        Genome: make(gene.Genome, g),
    }
    c.resetGenome()

    return c
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
    n := newCell(c.Idx, c.X, c.Y, int32(len(c.Genome)))

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

func (c *Cell) live() bool {
    return c.Energy > 0
}

func (c *Cell) viable(config Config) bool {
    return c.Generation >= config.ViableCellGeneration
}

func (c *Cell) exec(ctx *Context) *Delta {
    return ctx.vm.exec(c)
}

func (c *Cell) resetMetadata(ctx *Context) {
    c.resetID(ctx)
    c.Parent = 0
    c.Generation = 0
}

func (c *Cell) resetID(ctx *Context) {
    if c.live() {
        c.ID = ctx.env.getNextCellID()
    } else {
        c.ID = 0
    }
    c.Origin = c.ID
}

func (c *Cell) seed(ctx *Context) *Delta {
    c.Energy += ctx.env.GetRNG().Energy(ctx)
    c.resetMetadata(ctx)
    c.randomizeGenome(ctx)

    dt := &Delta{
        Cells: make([]*Cell, 1),
        Stats: make(Stats),
    }
    dt.Cells[0] = c

    return dt
}

func (c *Cell) accessible(ctx *Context, g gene.Gene, x gene.Gene) bool {
    return ctx.env.GetRNG().CellAccessible(ctx, c, g, x)
}
