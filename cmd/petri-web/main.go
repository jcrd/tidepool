// This project is licensed under the MIT License (see LICENSE).

package main

import (
    "flag"
    "log"
    "net/http"
    "text/template"
    "time"

    "petri/cmd"
    "petri/web"
)

type Index struct {
    Host string
    Scale int
}

func main() {
    update := flag.Duration("update", time.Second, "Delta update frequency")
    addr := flag.String("addr", ":3000", "http service address")
    index := flag.String("index", "index.html", "Path to html index file")
    scale := flag.Int("scale", 1, "Scale of cell visualization")

    env, dts := cmd.ParseAndRun()
    defer env.Stop()

    conn := web.NewConn(env, dts, time.Tick(*update))
    defer conn.Close()

    http.HandleFunc("/ws", conn.WebsocketHandler)
    http.HandleFunc("/env", conn.EnvHandler)

    indexTemp := template.Must(template.ParseFiles(*index))

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        indexTemp.Execute(w, Index{
            Host: r.Host,
            Scale: *scale,
        })
    })

    go conn.Run()

    if err := http.ListenAndServe(*addr, nil); err != nil {
        log.Fatal(err)
    }
}
