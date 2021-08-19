// This project is licensed under the MIT License (see LICENSE).

package main

import (
    "flag"
    "log"
    "net/http"
    _ "net/http/pprof"
    "runtime"
    "text/template"
    "time"

    "tidepool/cmd"
    "tidepool/web"
)

type Index struct {
    Host string
    Scale int
}

func init() {
    runtime.SetBlockProfileRate(1)
    runtime.SetMutexProfileFraction(1)
}

func main() {
    update := flag.Duration("update", time.Second, "Delta update frequency")
    addr := flag.String("addr", ":3000", "http service address")
    index := flag.String("index", "index.html", "Path to html index file")
    static := flag.String("static", "static", "Path to static directory")
    scale := flag.Int("scale", 1, "Scale of cell visualization")

    env, dts := cmd.ParseAndRun()
    defer env.Stop()

    conn := web.NewConn(env, dts, time.Tick(*update))
    defer conn.Close()

    http.HandleFunc("/ws", conn.WebsocketHandler)
    http.HandleFunc("/env", conn.EnvHandler)

    indexTemp := template.Must(template.ParseFiles(*index))

    http.Handle("/static/", http.StripPrefix("/static/",
        http.FileServer(http.Dir(*static))))

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
