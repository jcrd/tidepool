// This project is licensed under the MIT License (see LICENSE).

package petri

import (
    "math/rand"

    "petri/gene"
)

type Context struct {
    env *Env
    rand *rand.Rand
    vm *VM
    cellsBuf []int32
    dedupBuf []*Cell
}

func newContext(e *Env) *Context {
    ctx := &Context{
        env: e,
        rand: rand.New(rand.NewSource(e.Seed)),
        cellsBuf: make([]int32, e.Width * e.Height),
        dedupBuf: make([]*Cell, 0),
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
