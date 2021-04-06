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
    cellsBuf []int32
}

func newContext(e *Env) *Context {
    ctx := &Context{
        env: e,
        rand: rand.New(rand.NewSource(e.Seed)),
        cellsBuf: make([]int32, e.Width * e.Height),
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
