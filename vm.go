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

type CellMap map[int32]*Cell

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

    cellMap CellMap
}

func (cm CellMap) getCell(e *Env, idx int32) *Cell {
    c, ok := cm[idx]
    if !ok {
        c = e.GetCellByIdx(idx)
    }
    return c
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

func newVM(ctx *Context) *VM {
    env := ctx.env
    gs := env.GenomeSize

    vm := &VM{
        ctx: ctx,
        genomeMaxIdx: gs - 1,
        buffer: make(gene.Genome, gs),
        loopStack: make([]int32, gs),
        cellMap: make(CellMap),
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

    vm.cellMap.Reset()
}

func (vm *VM) incGenomeIdx() {
    if vm.genomeIdx == vm.genomeMaxIdx {
        vm.genomeIdx = genomeStartIdx
    } else {
        vm.genomeIdx++
    }
}

func (vm *VM) execGene(c *Cell, g gene.Gene, stats Stats) int {
    ctx := vm.ctx
    env := ctx.env

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
        idx := env.getNeighborIdx(c, vm.direction)
        n := vm.cellMap.getCell(env, idx)
        if n.accessible(ctx, vm.register, gene.KILL) {
            n.resetMetadata(ctx)
            n.resetGenome()

            vm.cellMap.AddCell(n)

            if n.Energy > 0 {
                stats.inc("LiveCellsKilled", 1)
            }
            if n.Generation >= config.ViableCellGeneration {
                stats.inc("ViableCellsKilled", 1)
            }
            stats.inc("CellsKilled", 1)
        } else if n.Generation >= config.ViableCellGeneration {
            c.Energy -= c.Energy / config.FailedKillPenalty
        }
    case gene.SHARE:
        config := env.GetConfig()
        idx := env.getNeighborIdx(c, vm.direction)
        n := vm.cellMap.getCell(env, idx)
        if n.accessible(ctx, vm.register, gene.SHARE) {
            e := c.Energy + n.Energy
            n.Energy = e / 2
            c.Energy = e - n.Energy

            if n.ID == 0 {
                n.resetID(ctx)
            }

            vm.cellMap.AddCell(n)

            if n.Generation >= config.ViableCellGeneration {
                stats.inc("ViableCellsShared", 1)
            }
            stats.inc("CellsShared", 1)
        }
    case gene.STOP:
        return VM_BREAK
    }

    return VM_NOOP
}

func (vm *VM) exec(c *Cell) *Delta {
    ctx := vm.ctx
    env := ctx.env

    defer vm.reset()
    vm.cellMap.AddCell(c)

    stats := make(Stats)

    for c.Energy > 0 {
        g := c.Genome[vm.genomeIdx]

        if env.GetRNG().Mutate(ctx) {
            mut := ctx.getRandomGene()
            if ctx.getRandomBool() {
                g = mut
            } else {
                vm.register = mut
            }
            stats.inc("Mutations", 1)
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
            r := vm.execGene(c, g, stats)
            if r == VM_BREAK {
                break
            } else if r == VM_CONTINUE {
                continue
            }
        }

        vm.incGenomeIdx()
    }

    if vm.buffer[0] != gene.STOP {
        idx := env.getNeighborIdx(c, vm.direction)
        n := vm.cellMap.getCell(env, idx)

        stats.inc("ReproductionAttempts", 1)

        if n.Energy > 0 && n.accessible(ctx, vm.register, gene.STOP) {
            n.ID = env.getNextCellID()
            n.Parent = c.ID
            n.Origin = c.Origin
            n.Generation = c.Generation + 1

            for i, g := range vm.buffer {
                n.Genome[i] = g
            }

            vm.cellMap.AddCell(n)

            stats.inc("Reproductions", 1)
            stats.update("MaxGeneration", n.Generation)
        }
    }

    if c.Energy == 0 {
        stats.inc("NaturalDeaths", 1)
        if c.Generation >= env.GetConfig().ViableCellGeneration {
            stats.inc("ViableCellNaturalDeaths", 1)
        }
    }

    return &Delta{
        Cells: vm.cellMap.Cells(),
        Stats: stats,
    }
}
