// This project is licensed under the MIT License (see LICENSE).

package petri

import (
    "context"
    "sync"
    "sync/atomic"
    "time"
)

type Env struct {
    Width int32
    Height int32
    GenomeSize int32
    Seed int64

    initPop int32

    config atomic.Value
    rng atomic.Value

    mutex *sync.RWMutex
    cells []*Cell
    liveCells map[int32]bool
    execCells map[int32]bool

    nextCellID chan int64

    Stop context.CancelFunc
}

type Config struct {
    InflowFrequency int64
    ViableCellGeneration int64
    FailedKillPenalty int64
    SeedLiveCells bool
}

const (
    dirLeft int = iota
    dirRight
    dirUp
    dirDown
)

const (
    cellDead int = iota
    cellLive
    cellAny
)

var defaultConfig = Config{
    InflowFrequency: 10,
    ViableCellGeneration: 2,
    FailedKillPenalty: 3,
    SeedLiveCells: true,
}

func NewEnv(width, height, genomeSize, pop int32, seed int64) *Env {
    e := &Env{
        Width: width,
        Height: height,
        GenomeSize: genomeSize,
        Seed: seed,
        initPop: pop,
        mutex: &sync.RWMutex{},
        cells: make([]*Cell, width * height),
        liveCells: make(map[int32]bool),
        execCells: make(map[int32]bool),
        nextCellID: make(chan int64),
    }

    if seed < 1 {
        e.Seed = time.Now().UnixNano()
    }

    for i := range e.cells {
        idx := int32(i)
        x := idx % width
        y := idx / width
        e.cells[i] = newCell(idx, x, y, genomeSize)
    }

    e.SetConfig(defaultConfig)
    e.SetRNG(defaultRNG)

    return e
}

func (e *Env) GetConfig() Config {
    return e.config.Load().(Config)
}

func (e *Env) SetConfig(c Config) {
    e.config.Store(c)
}

func (e *Env) GetRNG() RNG {
    return e.rng.Load().(RNG)
}

func (e *Env) SetRNG(r RNG) {
    e.rng.Store(r)
}

func (e *Env) getNextCellID() int64 {
    return <-e.nextCellID
}

func (e *Env) applyDelta(dt *Delta) {
    e.mutex.Lock()

    for _, c := range dt.Cells {
        if c.live() {
            e.liveCells[c.Idx] = true
        } else {
            delete(e.liveCells, c.Idx)
        }
        e.cells[c.Idx] = c.clone()
        delete(e.execCells, c.Idx)
    }

    var i int64
    config := e.GetConfig()
    for idx := range e.liveCells {
        c := e.cells[idx]
        if c.Generation >= config.ViableCellGeneration {
            i++
        }
    }
    dt.Stats["ViableLiveCells"] = i
    dt.Stats["LiveCells"] = int64(len(e.liveCells))

    e.mutex.Unlock()
}

func (e *Env) GetCell(x, y int32) *Cell {
    e.mutex.RLock()
    defer e.mutex.RUnlock()
    return e.cells[x + e.Width * y].clone()
}

func (e *Env) GetCellByIdx(idx int32) *Cell {
    e.mutex.RLock()
    defer e.mutex.RUnlock()
    return e.cells[idx].clone()
}

func (e *Env) getRandomCell(ctx *Context, state int) *Cell {
    fillBuf := func(idx int32, live bool, i *int) {
        if _, exec := e.execCells[idx]; exec {
            return
        }
        if !live {
            if _, live = e.liveCells[idx]; live {
                return
            }
        }
        ctx.cellsBuf[*i] = idx
        *i++
    }

    i := 0
    e.mutex.RLock()

    switch state {
    case cellDead:
        for _, c := range e.cells {
            fillBuf(c.Idx, false, &i)
        }
    case cellLive:
        for idx := range e.liveCells {
            fillBuf(idx, true, &i)
        }
    case cellAny:
        for _, c := range e.cells {
            fillBuf(c.Idx, true, &i)
        }
    }

    if i == 0 {
        e.mutex.RUnlock()
        return nil
    }

    c := e.cells[ctx.cellsBuf[ctx.rand.Intn(i)]].clone()
    e.mutex.RUnlock()

    e.mutex.Lock()
    e.execCells[c.Idx] = true
    e.mutex.Unlock()

    return c
}

func (e *Env) getNeighborIdx(c *Cell, dir int) int32 {
    x, y := c.X, c.Y

    switch dir {
    case dirLeft:
        if x == 0 {
            x = e.Width - 1
        } else {
            x--
        }
    case dirRight:
        if x == e.Width - 1 {
            x = 0
        } else {
            x++
        }
    case dirUp:
        if y == 0 {
            y = e.Height - 1
        } else {
            y--
        }
    case dirDown:
        if y == e.Height - 1 {
            y = 0
        } else {
            y++
        }
    }

    return x + e.Width * y
}

func (e *Env) process(wg *sync.WaitGroup, context context.Context,
    exec <-chan int64, inflow chan int64, dts chan<- *Delta) {
    defer wg.Done()

    ctx := newContext(e)

    for {
        select {
        case <-context.Done():
            return
        case ticks := <-inflow:
            var c *Cell
            if !e.GetConfig().SeedLiveCells {
                if c = e.getRandomCell(ctx, cellDead); c == nil {
                    break
                }
            } else {
                c = e.getRandomCell(ctx, cellAny)
            }
            dt := c.seed(ctx)
            dt.Stats["Ticks"] = ticks
            dts <- dt
        case ticks := <-exec:
            if c := e.getRandomCell(ctx, cellLive); c != nil {
                dt := c.exec(ctx)
                dt.Stats["Ticks"] = ticks
                dts <- dt
            } else {
                go func() {
                    inflow <- ticks
                }()
            }
        }
    }
}

func (e *Env) WithCells(f func([]*Cell)) {
    e.mutex.RLock()
    defer e.mutex.RUnlock()
    f(e.cells)
}

func (e *Env) Run(processN int, tick time.Duration, deltas chan<- *Delta) {
    exec := make(chan int64)
    inflow := make(chan int64)
    dts := make(chan *Delta, processN)

    context, stop := context.WithCancel(context.Background())
    e.Stop = stop

    var wg sync.WaitGroup
    wg.Add(processN)

    for i := 0; i < processN; i++ {
        go e.process(&wg, context, exec, inflow, dts)
    }

    go func() {
        defer close(e.nextCellID)
        var id int64 = 1
        for {
            select {
            case <-context.Done():
                return
            default:
                e.nextCellID <- id
                id++
            }
        }
    }()

    defer close(inflow)
    defer close(dts)
    defer close(deltas)

    ticker := time.NewTicker(tick)
    defer ticker.Stop()

    var ticks int64 = 0

    inflowTick := e.GetConfig().InflowFrequency
    sendInflow := func () {
        inflow <- ticks
        inflowTick = e.GetConfig().InflowFrequency
    }

    defer wg.Wait()

    for {
        select {
        case <-context.Done():
            return
        case <-ticker.C:
            ticks++
            if e.initPop > 0 {
                sendInflow()
                e.initPop--
            }
            inflowTick--
            if inflowTick == 0 {
                sendInflow()
            }
            exec <- ticks
        case dt := <-dts:
            e.applyDelta(dt)
            deltas <- dt
        }
    }
}
