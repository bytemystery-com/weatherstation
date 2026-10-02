package main

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"

	"weatherd/cfg"
)

//go:embed web/*
var webFiles embed.FS

func getCfg(w http.ResponseWriter, r *http.Request, cfg *cfg.Cfg) {
	w.Header().Set("Content-Type", "application/json")
	w.Write(cfg.GetJson())
}

func startWebServer(port int, done chan<- error, cfg *cfg.Cfg) *http.Server {
	mux := http.NewServeMux()
	webFS, err := fs.Sub(webFiles, "web")
	if err != nil {
		done <- err
		return nil
	}
	mux.Handle("GET /", http.FileServer(http.FS(webFS)))
	mux.HandleFunc("GET /getcfg", func(w http.ResponseWriter, r *http.Request) {
		getCfg(w, r, cfg)
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	go func() {
		err := server.ListenAndServe()
		done <- err
	}()

	return server
}
