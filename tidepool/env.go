// This project is licensed under the MIT License (see LICENSE).

package tidepool

import (
    "context"
    "errors"
    "math/rand"
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

    running uint32

    cells []*Cell
    cellsBuf []*Cell

    rand *rand.Rand

    nextCellID chan int64

    context context.Context
    Stop context.CancelFunc

    WithCells chan func([]*Cell)
}

type Config struct {
    InflowFrequency int64
    ViableCellGeneration int64
    FailedKillPenalty int64
}

var defaultConfig = Config{
    InflowFrequency: 10,
    ViableCellGeneration: 2,
    FailedKillPenalty: 3,
}

func getIdx(x, y, width int32) int32 {
	return y*width + x
}

func getCoords(idx, width int32) (int32, int32) {
	return idx % width, idx / width
}

func NewEnv(width, height, genomeSize, pop int32, seed int64) *Env {
    if seed < 1 {
        seed = time.Now().UnixNano()
    }

    e := &Env{
        Width: width,
        Height: height,
        GenomeSize: genomeSize,
        Seed: seed,
        initPop: pop,
        cells: make([]*Cell, width * height),
        cellsBuf: make([]*Cell, width * height),
        rand: rand.New(rand.NewSource(seed)),
        nextCellID: make(chan int64),
        WithCells: make(chan func([]*Cell)),
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

func (e *Env) applyDelta(dt *Delta, exec Refs, live Refs) {
    for _, c := range dt.Cells {
        if !c.live() {
            live.dec(c)
        } else if _, ok := live[c.Idx]; !ok {
            live.inc(c)
        }
        c.overwrite(e.cells[c.Idx])
    }

    for _, c := range dt.Neighborhood {
        exec.dec(c)
    }

    var i int64
    config := e.GetConfig()
    for idx := range live {
        c := e.cells[idx]
        if c.viable(config) {
            i++
        }
    }
    dt.Stats["ViableLiveCells"] = i
    dt.Stats["LiveCells"] = int64(len(live))
}

func (e *Env) getNeighborhood(c *Cell) (nh Neighborhood) {
	x, y := getCoords(c.Idx, e.Width)
    // Center cell is at index 0.
    nh[0] = e.cells[c.Idx]
	i := 1

	for _, w := range [...]int32{e.Width - 1, 0, 1} {
		for _, h := range [...]int32{e.Height - 1, 0, 1} {
			if w == 0 && h == 0 {
				continue
			}
			nh[i] = e.cells[getIdx((x+w)%e.Width, (y+h)%e.Height, e.Width)]
			i++
		}
	}

	return nh
}

func (e *Env) getRandomCell(exec Refs) *Cell {
    i := 0
    for _, c := range e.cells {
        if _, ref := exec[c.Idx]; !ref {
            e.cellsBuf[i] = c
            i++
        }
    }

    return e.cellsBuf[e.rand.Intn(i)]
}

func (e *Env) getExecNeighborhood(exec Refs) Neighborhood {
    nh := e.getNeighborhood(e.getRandomCell(exec))

    for i, c := range nh {
        nh[i] = c.clone()
        exec.inc(c)
    }

    return nh
}

func (e *Env) process(wg *sync.WaitGroup, exec <-chan int64, inflow <-chan int64,
    execNeighborhoods <-chan Neighborhood, dts chan<- *Delta) {

    defer wg.Done()
    ctx := newContext(e)

    handle := func (fn func(Neighborhood) *Delta, ticks int64) {
        dt := fn(<-execNeighborhoods)
        dt.Stats["Ticks"] = ticks
        dts <- dt
    }

    for {
        select {
        case <-e.context.Done():
            return
        case ticks := <-inflow:
            handle(ctx.seed, ticks)
        case ticks := <-exec:
            handle(ctx.vm.exec, ticks)
        }
    }
}

func (e *Env) Run(processN int, tick time.Duration, deltas chan<- *Delta) {
    exec := make(chan int64)
    inflow := make(chan int64)
    execNeighborhoods := make(chan Neighborhood, processN)
    dts := make(chan *Delta, processN)

    defer close(exec)
    defer close(inflow)

    var wg sync.WaitGroup
    wg.Add(processN)
    defer wg.Wait()

    for i := 0; i < processN; i++ {
        go e.process(&wg, exec, inflow, execNeighborhoods, dts)
    }

    go func() {
        defer close(dts)
        defer close(execNeighborhoods)
        defer close(deltas)
        defer close(e.WithCells)

        execRefs := make(Refs)
        liveRefs := make(Refs)

        for {
            select {
                case <-e.context.Done():
                    return
                case f := <-e.WithCells:
                    f(e.cells)
                case dt := <-dts:
                    e.applyDelta(dt, execRefs, liveRefs)
                    deltas <- dt
                default:
                    if len(execNeighborhoods) < processN {
                        nh := e.getExecNeighborhood(execRefs)
                        go func() {
                            execNeighborhoods <- nh
                        }()
                    }
            }
        }
    }()

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

    var ticks int64 = 0

    inflowTick := e.GetConfig().InflowFrequency
    sendInflow := func () {
        inflow <- ticks
        inflowTick = e.GetConfig().InflowFrequency
    }

    ticker := time.NewTicker(tick)
    defer ticker.Stop()

    atomic.StoreUint32(&e.running, 1)

    for {
        select {
        case <-e.context.Done():
            atomic.StoreUint32(&e.running, 0)
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
        }
    }
}

func (e *Env) GetCells() ([]*Cell, error) {
    if atomic.LoadUint32(&e.running) == 1 {
        return nil, errors.New("Env is running")
    }
    return e.cells, nil
}
