// This project is licensed under the MIT License (see LICENSE).

package petri

import (
    "petri/gene"
)

const (
    VM_NOOP int = iota
    VM_BREAK
    VM_CONTINUE
)

type VM struct {
    ctx *Context

    genomeIdx int32
    genomeMaxIdx int32

    loopStack []int32
    loopStackIdx int32
    loopDepth int32

    pointer int32
    register gene.Gene
    direction int
    buffer gene.Genome

    cellN []int
}

func newVM(ctx *Context) *VM {
    env := ctx.env
    gs := env.GenomeSize

    vm := &VM{
        ctx: ctx,
        genomeMaxIdx: gs - 1,
        buffer: make(gene.Genome, gs),
        loopStack: make([]int32, gs),
        cellN: make([]int, env.Width * env.Height),
    }
    vm.reset()

    return vm
}

func (vm *VM) reset() {
    vm.genomeIdx = genomeStartIdx
    vm.loopStackIdx = 0
    vm.loopDepth = 0
    vm.pointer = 0
    vm.register = gene.ZERO
    vm.direction = 0

    for i := range vm.buffer {
        vm.buffer[i] = gene.STOP
    }

    for i := range vm.cellN {
        vm.cellN[i] = 0
    }
}

func (vm *VM) incGenomeIdx() {
    if vm.genomeIdx == vm.genomeMaxIdx {
        vm.genomeIdx = genomeStartIdx
    } else {
        vm.genomeIdx++
    }
}

func (vm *VM) addCell(dt *Delta, c *Cell) {
    dt.Cells = append(dt.Cells, c)
    vm.cellN[c.idx]++
}

func (vm *VM) dedupCells(dt *Delta) {
    cs := make([]*Cell, 0)

    for _, c := range dt.Cells {
        vm.cellN[c.idx]--
        if vm.cellN[c.idx] == 0 {
            cs = append(cs, c)
        }
    }

    dt.Cells = cs
}

func (vm *VM) execGene(c *Cell, g gene.Gene, dt *Delta) int {
    ctx := vm.ctx
    env := ctx.env

    dt.Stats.GeneExecN[g]++

    switch g {
    case gene.ZERO:
        vm.pointer = 0
        vm.register = gene.ZERO
        vm.direction = 0
    case gene.FWD:
        if vm.pointer == vm.genomeMaxIdx {
            vm.pointer = 0
        } else {
            vm.pointer++
        }
    case gene.BACK:
        if vm.pointer == 0 {
            vm.pointer = vm.genomeMaxIdx
        } else {
            vm.pointer--
        }
    case gene.INC:
        if vm.register == gene.STOP {
            vm.register = gene.ZERO
        } else {
            vm.register++
        }
    case gene.DEC:
        if vm.register == gene.ZERO {
            vm.register = gene.STOP
        } else {
            vm.register--
        }
    case gene.READG:
        vm.register = c.Genome[vm.pointer]
    case gene.WRITEG:
        c.Genome[vm.pointer] = vm.register
    case gene.READB:
        vm.register = vm.buffer[vm.pointer]
    case gene.WRITEB:
        vm.buffer[vm.pointer] = vm.register
    case gene.LOOP:
        if vm.register == gene.ZERO {
            vm.loopDepth = 1
        } else if vm.loopStackIdx > vm.genomeMaxIdx {
            return VM_BREAK
        } else {
            vm.loopStack[vm.loopStackIdx] = vm.genomeIdx
            vm.loopStackIdx++
        }
    case gene.REP:
        if vm.loopStackIdx > 0 {
            vm.loopStackIdx--
            if vm.register != gene.ZERO {
                vm.genomeIdx = vm.loopStack[vm.loopStackIdx]
                return VM_CONTINUE
            }
        }
    case gene.TURN:
        vm.direction = int(vm.register % 4)
    case gene.XCHG:
        reg := vm.register
        vm.incGenomeIdx()
        vm.register = c.Genome[vm.genomeIdx]
        c.Genome[vm.genomeIdx] = reg
    case gene.KILL:
        config := env.GetConfig()
        n := env.getNeighbor(c, vm.direction)
        if n.accessible(ctx, vm.register, gene.KILL) {
            n.resetMetadata(ctx)
            n.resetGenome()

            vm.addCell(dt, n)

            if n.Generation >= config.ViableCellGeneration {
                dt.Stats.ViableCellKilled++
            }
        } else if n.Generation >= config.ViableCellGeneration {
            c.Energy -= c.Energy / config.FailedKillPenalty
        }
    case gene.SHARE:
        config := env.GetConfig()
        n := env.getNeighbor(c, vm.direction)
        if n.accessible(ctx, vm.register, gene.SHARE) {
            e := c.Energy + n.Energy
            n.Energy = e / 2
            c.Energy = e - n.Energy

            if n.ID == 0 {
                n.resetID(ctx)
            }

            vm.addCell(dt, n)

            if n.Generation >= config.ViableCellGeneration {
                dt.Stats.ViableCellShared++
            }
        }
    case gene.STOP:
        return VM_BREAK
    }

    return VM_NOOP
}

func (vm *VM) exec(c *Cell) *Delta {
    ctx := vm.ctx
    env := ctx.env

    dt := newDelta()
    vm.addCell(dt, c)

    defer vm.reset()
    defer vm.dedupCells(dt)

    for c.Energy > 0 {
        g := c.Genome[vm.genomeIdx]

        if env.GetRNG().Mutate(ctx) {
            mut := ctx.getRandomGene()
            if ctx.getRandomBool() {
                g = mut
            } else {
                vm.register = mut
            }
        }

        c.Energy--

        if vm.loopDepth > 0 {
            switch g {
            case gene.LOOP:
                vm.loopDepth++
            case gene.REP:
                vm.loopDepth--
                continue
            }
        } else {
            r := vm.execGene(c, g, dt)
            if r == VM_BREAK {
                break
            } else if r == VM_CONTINUE {
                continue
            }
        }

        vm.incGenomeIdx()
    }

    if vm.buffer[0] != gene.STOP {
        n := env.getNeighbor(c, vm.direction)
        if n.Energy > 0 && n.accessible(ctx, vm.register, gene.STOP) {
            n.ID = env.getNextCellID()
            n.Parent = c.ID
            n.Origin = c.Origin
            n.Generation = c.Generation + 1

            for i, g := range vm.buffer {
                n.Genome[i] = g
            }

            vm.addCell(dt, n)
        }
    }

    return dt
}
