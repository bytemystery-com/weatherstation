package main

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
)

//go:embed web/*
var webFiles embed.FS

type FilterData struct {
	TemperatureFilterConstant  string `json:"t_fc"`
	TemperatureFilterConstant2 string `json:"t_fc2"`
	HumidityFilterConstant     string `json:"h_fc"`
	HumidityFilterConstant2    string `json:"h_fc2"`
	DewPointFilterConstant     string `json:"d_fc"`
	DewPointFilterConstant2    string `json:"d_fc2"`
	PressureFilterConstant     string `json:"p_fc"`
	LuxFilterConstant          string `json:"l_fc"`
	LuxFilterConstant2         string `json:"l_fc2"`
}

type GeneralData struct {
	Height  string `json:"height"`
	Name    string `json:"name"`
	DNSName string `json:"dns_name"`
	UsePass bool   `json:"use_pass"`
}

type ConfigResponse struct {
	FilterData FilterData  `json:"filterdata"`
	General    GeneralData `json:"general"`
}

func getCfg(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}"))
}

func startWebServer(port int, done chan<- error) *http.Server {
	mux := http.NewServeMux()
	webFS, err := fs.Sub(webFiles, "web")
	if err != nil {
		done <- err
		return nil
	}
	mux.Handle("GET /", http.FileServer(http.FS(webFS)))
	mux.HandleFunc("GET /getcfg", getCfg)

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
