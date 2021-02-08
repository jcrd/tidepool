package petri

type Stats map[string]int64

func (s Stats) inc(name string, i int64) {
    if v, ok := s[name]; ok {
        i = i + v
    }
    s[name] = i
}

func (s Stats) update(name string, i int64) {
    if max, ok := s[name]; !ok || i > max {
        s[name] = i
    }
}

func (s Stats) set(name string, i int64) {
    s[name] = i
}

func (s Stats) Add(a Stats) {
    for n, i := range a {
        switch n {
        case "Ticks":
            fallthrough
        case "MaxGeneration":
            s.update(n, i)
        case "ViableLiveCells":
            fallthrough
        case "LiveCells":
            s.set(n, i)
        default:
            s.inc(n, i)
        }
    }
}
