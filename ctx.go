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
}

func newContext(e *Env) *Context {
    ctx := &Context{
        env: e,
        rand: rand.New(rand.NewSource(e.Seed)),
    }
    ctx.vm = newVM(ctx)

    return ctx
}

func (ctx *Context) getRandomCell() *Cell {
    x := ctx.rand.Int31n(ctx.env.Width)
    y := ctx.rand.Int31n(ctx.env.Height)
    return ctx.env.GetCell(x, y)
}

func (ctx *Context) getRandomGene() gene.Gene {
    return gene.Gene(ctx.rand.Intn(int(gene.N)))
}

func (ctx *Context) getRandomBool() bool {
    return ctx.rand.Intn(2) == 1
}
