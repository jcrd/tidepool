// This project is licensed under the MIT License (see LICENSE).

package tidepool

import (
    "math/rand"

    "tidepool/tidepool/gene"
)

type Context struct {
    env *Env
    rand *rand.Rand
    vm *VM
}

func newContext(e *Env) *Context {
    ctx := &Context{
        env: e,
        rand: rand.New(rand.NewSource(e.Seed)),
    }
    ctx.vm = newVM(ctx)

    return ctx
}

func (ctx *Context) getRandomGene() gene.Gene {
    return gene.Gene(ctx.rand.Intn(int(gene.N)))
}

func (ctx *Context) getRandomBool() bool {
    return ctx.rand.Intn(2) == 1
}

func (ctx *Context) seed(nh Neighborhood) *Delta {
    c := nh[0]
    c.Energy += ctx.env.GetRNG().Energy(ctx)
    c.resetMetadata(ctx)
    c.randomizeGenome(ctx)

    dt := &Delta{
        Cells: make([]*Cell, 1),
        Neighborhood: nh,
        Stats: make(Stats),
    }
    dt.Cells[0] = c

    return dt
}
