// This project is licensed under the MIT License (see LICENSE).

package petri

import (
    "petri/gene"
)

type RNG interface {
    Mutate(*Context) bool
    Energy(*Context) int
    CellAccessible(*Context, *Cell, gene.Gene, gene.Gene) bool
}

type DefaultRNG struct {
    MutationRate float64
    InflowRateBase int
    InflowRateModifier int
    bitsPerGene [gene.N]int
}

var defaultRNG = DefaultRNG{
    MutationRate: 0.00000115,
    InflowRateBase: 600,
    InflowRateModifier: 1000,
    bitsPerGene: [gene.N]int{0, 1, 1, 2, 1, 2, 2, 3, 1, 2, 2, 3, 2, 3, 3, 4},
}

func (r DefaultRNG) Mutate(ctx *Context) bool {
    return ctx.rand.Float64() < r.MutationRate
}

func (r DefaultRNG) Energy(ctx *Context) int {
    return r.InflowRateBase + (ctx.rand.Int() % r.InflowRateModifier)
}

func (r DefaultRNG) CellAccessible(ctx *Context, c *Cell,
    logo gene.Gene, mode gene.Gene) bool {
    if c.Energy == 0 || c.Generation == 0 {
        return true
    }

    i := int(ctx.getRandomGene())
    b := r.bitsPerGene[c.logo() ^ logo]

    switch mode {
    case gene.KILL:
        fallthrough
    case gene.STOP:
        return i <= b
    case gene.SHARE:
        return i >= b
    }

    return false
}
