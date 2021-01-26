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

var (
    indexTemp = template.Must(template.ParseFiles("index.html"))
    upgrader = websocket.Upgrader{}
    conn = &Conn{
        mutex: &sync.RWMutex{},
        channels: make(map[int]chan []byte),
    }
    stats = &petri.Stats{}
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
    indexTemp.Execute(w, "ws://" + r.Host + "/ws")
}

func wsHandle(w http.ResponseWriter, r *http.Request) {
    c, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println(err)
        return
    }
    defer c.Close()

    json, err := json.Marshal(stats)
    if err != nil {
        log.Println(err)
        return
    }
    c.WriteMessage(websocket.TextMessage, json)

    ch := make(chan []byte)
    id := conn.addChannel(ch)

    go func() {
        for json := range ch {
            c.WriteMessage(websocket.TextMessage, json)
        }
    }()

    if _, _, err := c.ReadMessage(); err != nil {
        conn.delChannel(id)
    }
}

func main() {
    u := flag.Duration("update", time.Second, "Stats update frequency")
    a := flag.String("addr", ":3000", "http service address")

    env, dts := cmd.ParseAndRun()
    defer env.Stop()

    http.HandleFunc("/", indexHandle)
    http.HandleFunc("/ws", wsHandle)

    update := time.Tick(*u)

    go func() {
        defer conn.Close()
        for {
            select {
            case dt, ok := <-dts:
                if !ok {
                    return
                }
                stats.Add(dt.Stats)
            case <-update:
                json, err := json.Marshal(stats)
                if err != nil {
                    log.Println(err)
                    break
                }
                conn.mutex.RLock()
                for _, ch := range conn.channels {
                    ch <- json
                }
                conn.mutex.RUnlock()
            }
        }
    }()

    if err := http.ListenAndServe(*a, nil); err != nil {
        log.Fatal(err)
    }
}
