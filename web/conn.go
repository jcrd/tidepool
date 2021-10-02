// This project is licensed under the MIT License (see LICENSE).

package web

import (
    "encoding/json"
    "log"
    "net/http"
    "sync"
    "time"

    tp "tidepool/tidepool"

    "github.com/gorilla/websocket"
)

type Conn struct {
    env *tp.Env
    stats tp.Stats
    cellMap tp.CellMap
    request chan int
    deltas <-chan *tp.Delta
    update <-chan time.Time

    upgrader websocket.Upgrader
    mutex *sync.RWMutex
    channels map[int]chan []byte
    nextID int
}

type EnvJSON struct {
    Width int32
    Height int32
    ViableCellGeneration int64
}

func NewConn(e *tp.Env, d <-chan *tp.Delta, u <-chan time.Time) *Conn {
    return &Conn{
        env: e,
        stats: make(tp.Stats),
        cellMap: make(tp.CellMap),
        request: make(chan int),
        deltas: d,
        update: u,

        upgrader: websocket.Upgrader{},
        mutex: &sync.RWMutex{},
        channels: make(map[int]chan []byte),
    }
}

func (c *Conn) addChannel(ch chan []byte) int {
    c.mutex.Lock()
    id := c.nextID
    c.nextID++
    c.channels[id] = ch
    c.mutex.Unlock()
    return id
}

func (c *Conn) delChannel(id int) {
    c.mutex.Lock()
    close(c.channels[id])
    delete(c.channels, id)
    c.mutex.Unlock()
}

func (c *Conn) Close() {
    c.mutex.Lock()
    for _, ch := range c.channels {
        close(ch)
    }
    c.mutex.Unlock()
}

func (c *Conn) WebsocketHandler(w http.ResponseWriter, r *http.Request) {
    s, err := c.upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println(err)
        return
    }
    defer s.Close()

    ch := make(chan []byte)
    id := c.addChannel(ch)

    go func() {
        c.request <- id
        for json := range ch {
            s.WriteMessage(websocket.TextMessage, json)
        }
    }()

    if _, _, err := s.ReadMessage(); err != nil {
        c.delChannel(id)
    }
}

func (c *Conn) EnvHandler(w http.ResponseWriter, r *http.Request) {
    config := c.env.GetConfig()
    j := EnvJSON{
        Width: c.env.Width,
        Height: c.env.Height,
        ViableCellGeneration: config.ViableCellGeneration,
    }
    json.NewEncoder(w).Encode(j)
}

func (c *Conn) Run() {
    for {
        select {
        case dt, ok := <-c.deltas:
            if !ok {
                return
            }
            for _, cell := range dt.Cells {
                c.cellMap.AddCell(cell)
            }
            c.stats.Add(dt.Stats)
        case id := <-c.request:
            ret := make(chan []byte)
            go func() {
                c.env.WithCells <- func(cs []*tp.Cell) {
                    dt := &tp.Delta{
                        Cells: cs,
                        Stats: c.stats,
                    }
                    if js, err := json.Marshal(dt); err == nil {
                        ret <- js
                    } else {
                        log.Println(err)
                        close(ret)
                    }
                }
            }()
            js, ok := <-ret
            if !ok {
                break
            }
            close(ret)
            c.mutex.RLock()
            if ch, ok := c.channels[id]; ok {
                ch <- js
            }
            c.mutex.RUnlock()
        case <-c.update:
            dt := &tp.Delta{
                Cells: c.cellMap.Cells(),
                Stats: c.stats,
            }
            js, err := json.Marshal(dt)
            if err != nil {
                log.Println(err)
                break
            }
            c.cellMap.Reset()
            c.mutex.RLock()
            for _, ch := range c.channels {
                ch <- js
            }
            c.mutex.RUnlock()
        }
    }
}
