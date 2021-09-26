// This project is licensed under the MIT License (see LICENSE).

package tidepool

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
    liveCells map[int32]struct{}
    execCells map[int32]struct{}

    nextCellID chan int64

    context context.Context
    Stop context.CancelFunc
}

type Config struct {
    InflowFrequency int64
    ViableCellGeneration int64
    FailedKillPenalty int64
    SeedViableCells bool
}

const (
    cellDead = (1 << 0)
    cellLive = (1 << 1)
    cellNonviable = (1 << 2)
    cellAny = cellDead | cellLive
)

var defaultConfig = Config{
    InflowFrequency: 10,
    ViableCellGeneration: 2,
    FailedKillPenalty: 3,
    SeedViableCells: false,
}

func getIdx(x, y, width int32) int32 {
	return y*width + x
}

func getCoords(idx, width int32) (int32, int32) {
	return idx % width, idx / width
}

func getNeighbors(idx, width, height int32) (ns [8]int32) {
	x, y := getCoords(idx, width)
	i := 0

	for _, w := range [...]int32{width - 1, 0, 1} {
		for _, h := range [...]int32{height - 1, 0, 1} {
			if w == 0 && h == 0 {
				continue
			}
			ns[i] = getIdx((x+w)%width, (y+h)%height, width)
			i++
		}
	}

	return ns
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
        liveCells: make(map[int32]struct{}),
        execCells: make(map[int32]struct{}),
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

    e.context, e.Stop = context.WithCancel(context.Background())

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
            e.liveCells[c.Idx] = struct{}{}
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
        if c.viable(config) {
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
    config := e.GetConfig()

    fillBuf := func(idx int32, s int, i *int) {
        if _, exec := e.execCells[idx]; exec {
            return
        }
        if s & cellLive == 0 {
            if _, live := e.liveCells[idx]; live {
                return
            }
        }
        if s & cellNonviable == 1 && e.cells[idx].viable(config) {
            return
        }
        ctx.cellsBuf[*i] = idx
        *i++
    }

    i := 0
    e.mutex.RLock()

    if state & cellLive == state {
        for idx := range e.liveCells {
            fillBuf(idx, cellLive, &i)
        }
    } else {
        for _, c := range e.cells {
            fillBuf(c.Idx, state, &i)
        }
    }

    if i == 0 {
        e.mutex.RUnlock()
        return nil
    }

    c := e.cells[ctx.cellsBuf[ctx.rand.Intn(i)]].clone()
    e.mutex.RUnlock()

    e.mutex.Lock()
    e.execCells[c.Idx] = struct{}{}
    e.mutex.Unlock()

    return c
}

func (e *Env) getNeighborIdx(c *Cell, dir int) int32 {
    return getNeighbors(c.Idx, e.Width, e.Height)[dir]
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
            if !e.GetConfig().SeedViableCells {
                if c = e.getRandomCell(ctx, cellAny | cellNonviable); c == nil {
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

    var wg sync.WaitGroup
    wg.Add(processN)

    for i := 0; i < processN; i++ {
        go e.process(&wg, e.context, exec, inflow, dts)
    }

    go func() {
        defer close(e.nextCellID)
        var id int64 = 1
        for {
            select {
            case <-e.context.Done():
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
        case <-e.context.Done():
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
