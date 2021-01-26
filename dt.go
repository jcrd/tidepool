package petri

type Delta struct {
    Cells map[int32]*Cell
    Stats *Stats
}

func newDelta() *Delta {
    return &Delta{
        Cells: make(map[int32]*Cell),
        Stats: NewStats(),
    }
}

func (dt *Delta) addCell(c *Cell) {
    dt.Cells[c.Idx] = c
}

func (dt *Delta) getCell(e *Env, idx int32) *Cell {
    c, ok := dt.Cells[idx]
    if !ok {
        c = e.GetCellByIdx(idx)
    }
    return c
}
