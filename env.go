// This project is licensed under the MIT License (see LICENSE).

package petri

import (
    "sync"
    "sync/atomic"
    "time"
)

type Env struct {
    Width int32
    Height int32
    GenomeSize int32
    Seed int64

    config atomic.Value
    rng atomic.Value

    mutex *sync.RWMutex
    cells []*Cell

    clock int64

    nextCellID chan int64
}

type Config struct {
    InflowFrequency int64
    ViableCellGeneration int64
    FailedKillPenalty int
}

const (
    DIR_LEFT int = iota
    DIR_RIGHT
    DIR_UP
    DIR_DOWN
)

var defaultConfig = Config{
    InflowFrequency: 100,
    ViableCellGeneration: 3,
    FailedKillPenalty: 3,
}

func NewEnv(width, height, genomeSize int32, seed int64) *Env {
    e := &Env{
        Width: width,
        Height: height,
        GenomeSize: genomeSize,
        Seed: seed,
        mutex: &sync.RWMutex{},
        cells: make([]*Cell, width * height),
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
        e.cells[c.idx] = c.clone()
    }
    e.mutex.Unlock()
}

func (e *Env) GetCell(x, y int32) *Cell {
    e.mutex.RLock()
    defer e.mutex.RUnlock()
    return e.cells[x + e.Width * y].clone()
}

func (e *Env) getNeighbor(c *Cell, dir int) *Cell {
    x, y := c.X, c.Y

    switch dir {
    case DIR_LEFT:
        if x == 0 {
            x = e.Width - 1
        } else {
            x--
        }
    case DIR_RIGHT:
        if x == e.Width - 1 {
            x = 0
        } else {
            x++
        }
    case DIR_UP:
        if y == 0 {
            y = e.Height - 1
        } else {
            y--
        }
    case DIR_DOWN:
        if y == e.Height - 1 {
            y = 0
        } else {
            y++
        }
    }

    return e.GetCell(x, y)
}

func (e *Env) process(exec, inflow <-chan bool, dts chan<- *Delta) {
    ctx := newContext(e)
    for {
        select {
        case <-inflow:
            dts <- ctx.getRandomCell().seed(ctx)
        case <-exec:
            if dt := ctx.getRandomCell().exec(ctx); dt != nil {
                dts <- dt
            }
        }
    }
}

func (e *Env) Run(processN int, tick time.Duration, deltas chan<- *Delta) {
    exec := make(chan bool)
    inflow := make(chan bool)
    dts := make(chan *Delta, processN)

    for i := 0; i < processN; i++ {
        go e.process(exec, inflow, dts)
    }

    go func() {
        defer close(e.nextCellID)
        var id int64 = 1
        for {
            e.nextCellID <- id
            id++
        }
    }()

    go func() {
        defer close(dts)
        defer close(deltas)
        for dt := range dts {
            e.applyDelta(dt)
            deltas <- dt
        }
    }()

    defer close(inflow)
    defer close(exec)

    inflow <- true

    for range time.Tick(tick) {
        e.clock++
        if e.clock % e.GetConfig().InflowFrequency == 0 {
            inflow <- true
        }
        exec <- true
    }
}
