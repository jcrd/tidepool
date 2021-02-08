// This project is licensed under the MIT License (see LICENSE).

package main

import (
    "encoding/json"
    "flag"
    "log"
    "net/http"
    "sync"
    "text/template"
    "time"

    "petri"
    "petri/cmd"

    "github.com/gorilla/websocket"
)

type Conn struct {
    mutex *sync.RWMutex
    channels map[int]chan []byte
    nextID int
}

type EnvJSON struct {
    Width int32
    Height int32
    ViableCellGeneration int64
}

type Index struct {
    Host string
    Scale int
}

var (
    indexTemp *template.Template
    upgrader = websocket.Upgrader{}
    conn = &Conn{
        mutex: &sync.RWMutex{},
        channels: make(map[int]chan []byte),
    }

    env *petri.Env
    scale int
    stats = make(petri.Stats)
    cellMap = make(petri.CellMap)
    request = make(chan int)
)

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

func indexHandle(w http.ResponseWriter, r *http.Request) {
    indexTemp.Execute(w, Index{
        Host: r.Host,
        Scale: scale,
    })
}

func wsHandle(w http.ResponseWriter, r *http.Request) {
    c, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println(err)
        return
    }
    defer c.Close()

    ch := make(chan []byte)
    id := conn.addChannel(ch)

    go func() {
        request <- id
        for json := range ch {
            c.WriteMessage(websocket.TextMessage, json)
        }
    }()

    if _, _, err := c.ReadMessage(); err != nil {
        conn.delChannel(id)
    }
}

func envHandle(w http.ResponseWriter, r *http.Request) {
    config := env.GetConfig()
    j := EnvJSON{
        Width: env.Width,
        Height: env.Height,
        ViableCellGeneration: config.ViableCellGeneration,
    }
    json.NewEncoder(w).Encode(j)
}

func main() {
    u := flag.Duration("update", time.Second, "Stats update frequency")
    a := flag.String("addr", ":3000", "http service address")
    index := flag.String("index", "index.html", "Path to html index file")
    flag.IntVar(&scale, "scale", 1, "Scale of cell visualization")

    var dts <-chan *petri.Delta

    env, dts = cmd.ParseAndRun()
    defer env.Stop()

    indexTemp = template.Must(template.ParseFiles(*index))

    http.HandleFunc("/", indexHandle)
    http.HandleFunc("/ws", wsHandle)
    http.HandleFunc("/env", envHandle)

    update := time.Tick(*u)

    go func() {
        defer conn.Close()
        for {
            select {
            case dt, ok := <-dts:
                if !ok {
                    return
                }
                for _, c := range dt.Cells {
                    cellMap.AddCell(c)
                }
                stats.Add(dt.Stats)
            case id := <-request:
                var js []byte
                var err error
                env.WithCells(func(cs []*petri.Cell) {
                    dt := &petri.Delta{
                        Cells: cs,
                        Stats: stats,
                    }
                    js, err = json.Marshal(dt)
                })
                if err != nil {
                    log.Println(err)
                    break
                }
                conn.mutex.RLock()
                if ch, ok := conn.channels[id]; ok {
                    ch <- js
                }
                conn.mutex.RUnlock()
            case <-update:
                dt := &petri.Delta{
                    Cells: cellMap.Cells(),
                    Stats: stats,
                }
                js, err := json.Marshal(dt)
                if err != nil {
                    log.Println(err)
                    break
                }
                cellMap.Reset()
                conn.mutex.RLock()
                for _, ch := range conn.channels {
                    ch <- js
                }
                conn.mutex.RUnlock()
            }
        }
    }()

    if err := http.ListenAndServe(*a, nil); err != nil {
        log.Fatal(err)
    }
}
