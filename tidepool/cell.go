// This project is licensed under the MIT License (see LICENSE).

package tidepool

import (
    "tidepool/tidepool/gene"
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

// The center cell is at index 0, followed by the 8 neighbors.
type Neighborhood [9]*Cell

type Delta struct {
    Cells []*Cell
    Neighborhood Neighborhood
    Stats Stats
}

type CellMap map[int32]*Cell
type Refs map[int32]int

func (r Refs) inc(c *Cell) {
    if v, ok := r[c.Idx]; ok {
        r[c.Idx] = v + 1
    } else {
        r[c.Idx] = 1
    }
}

func (r Refs) dec(c *Cell) {
    if v, ok := r[c.Idx]; ok {
        v--
        if v == 0 {
            delete(r, c.Idx)
        } else {
            r[c.Idx] = v
        }
    }
}

func (cm CellMap) getNeighbor(nh Neighborhood, dir int) *Cell {
    // The executing cell is at index 0.
    n := nh[dir + 1]
    if c, ok := cm[n.Idx]; ok {
        return c
    }
    return n
}

func (cm CellMap) AddCell(c *Cell) {
    cm[c.Idx] = c
}

func (cm CellMap) Reset() {
    for i := range cm {
        delete(cm, i)
    }
}

func (cm CellMap) Cells() []*Cell {
    cs := make([]*Cell, len(cm))
    i := 0
    for _, c := range cm {
        cs[i] = c
        i++
    }
    return cs
}

func (nh Neighborhood) seed(ctx *Context) *Delta {
    c := nh[0]
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

func (c *Cell) overwrite(w *Cell) {
    w.ID = c.ID
    w.Origin = c.Origin
    w.Parent = c.Parent
    w.Generation = c.Generation
    w.Energy = c.Energy

    for i, v := range c.Genome {
        w.Genome[i] = v
    }
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

func (c *Cell) accessible(ctx *Context, g gene.Gene, x gene.Gene) bool {
    return ctx.env.GetRNG().CellAccessible(ctx, c, g, x)
}
