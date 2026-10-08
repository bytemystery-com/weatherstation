// Copyright (c) 2026 Reiner Pröls
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
//
// SPDX-License-Identifier: MIT
//
// Author: Reiner Pröls

// Webserver stuff

package webserver

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"weatherd/cfg"
	"weatherd/helper"
	"weatherd/raspi"
	"weatherd/sd"
)

type HasErrors interface {
	GetErrors() uint
}

type WebData struct {
	SensorName string `json:"name"`
	Errors     uint   `json:"errors"`
	Valid      bool   `json:"valid"`
}

func (w *WebData) GetErrors() uint {
	return w.Errors
}

type WebDataLux struct {
	WebData
	Lux  float32 `json:"lux"`
	Lux2 float32 `json:"lux2"`
}

type WebDataPressure struct {
	WebData
	Temperature            float32 `json:"temperature"`
	Temperature2           float32 `json:"temperature2"`
	Pressure               float32 `json:"pressure"`
	PressureNormal         float32 `json:"pressure_normal"`
	Pressure2              float32 `json:"pressure2"`
	Pressure2Normal        float32 `json:"pressure2_normal"`
	PressureTrend1         float32 `json:"-"`
	PressureTrend1Goodness float32 `json:"-"`
	PressureTrend3         float32 `json:"-"`
	PressureTrend3Goodness float32 `json:"-"`
}

type WebDataHumidity struct {
	WebData
	Temperature  float32 `json:"temperature"`
	Temperature2 float32 `json:"temperature2"`
	Humidity     float32 `json:"humidity"`
	Humidity2    float32 `json:"humidity2"`
	DewPoint     float32 `json:"dewpoint"`
	DewPoint2    float32 `json:"dewpoint2"`
}

type WebDataAll struct {
	Lux      WebDataLux
	Pressure WebDataPressure
	Humidity WebDataHumidity
}

type WebCallBacks interface {
	GetData() *WebDataAll
}

//go:embed web/*
var webFiles embed.FS

type WebServer struct {
	callBacks WebCallBacks
	port      int
	server    *http.Server
	cfg       *cfg.Cfg
	sdData    *sd.SdData
}

func NewWebServer(port int, callBacks WebCallBacks, config *cfg.Cfg, sdData *sd.SdData) *WebServer {
	return &WebServer{
		callBacks: callBacks,
		port:      port,
		cfg:       config,
		sdData:    sdData,
	}
}

func (ws *WebServer) getUptime() string {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return ""
	}

	var seconds float64
	_, err = fmt.Sscanf(string(data), "%f", &seconds)
	if err != nil {
		return ""
	}

	d := int64(seconds)
	days := d / 86400
	hours := (d % 86400) / 3600
	minutes := (d % 3600) / 60
	secs := d % 60

	return fmt.Sprintf("%dd %02dh %02dm %02ds", days, hours, minutes, secs)
}

func (ws *WebServer) getCfg(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write(ws.cfg.GetJson())
}

func (ws *WebServer) isDST(t time.Time) int {
	_, offset := t.Zone()
	_, standardOffset := time.Date(t.Year(), time.January, 1, 0, 0, 0, 0, t.Location()).Zone()
	if offset != standardOffset {
		return 1
	}
	return 0
}

func (ws *WebServer) getUpdate(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	data := ws.callBacks.GetData()
	type config struct {
		UpdateInterval int `json:"update_interval"`
	}

	type timed struct {
		Day   int `json:"day"`
		Month int `json:"month"`
		Year  int `json:"year"`
		Wday  int `json:"wday"`
		Hour  int `json:"hour"`
		Min   int `json:"min"`
		Sec   int `json:"sec"`
		Dst   int `json:"dst"`
	}

	type sd struct {
		Free  uint64 `json:"free"`
		Total uint64 `json:"total"`
	}

	type raspiStatus struct {
		Fan         int    `json:"fan"`
		CpuUsage    int    `json:"cpu"`
		Temperature int    `json:"temp"`
		Voltage_5v  int    `json:"voltage_5"`
		Serial      string `json:"serial"`
		Version     uint16 `json:"version"`
		Throttle    uint32 `json:"throttle"`
		Ram_used    uint32 `json:"ram_u"`
		Ram_total   uint32 `json:"ram_t"`
	}
	type status struct {
		Version                string      `json:"version"`
		Uptime                 string      `json:"uptime"`
		PressureTrend1         float32     `json:"pressure_trend_1"`
		PressureTrend3         float32     `json:"pressure_trend_3"`
		PressureTrend1Goodness float32     `json:"pressure_trend_1_goodness"`
		PressureTrend3Goodness float32     `json:"pressure_trend_3_goodness"`
		Time                   timed       `json:"time"`
		Sd                     sd          `json:"sd"`
		Resets                 uint        `json:"resets"`
		Errors                 uint        `json:"errors"`
		RaspiStatus            raspiStatus `json:"raspi"`
	}

	type dataJson struct {
		Config  config `json:"config"`
		Status  status `json:"status"`
		Sensors []any  `json:"sensors"`
	}
	list := make([]any, 0, 3)
	list = append(list, data.Pressure)
	list = append(list, data.Humidity)
	list = append(list, data.Lux)

	var t timed
	now := time.Now()
	t.Day = now.Day()
	t.Wday = int(now.Weekday())
	t.Month = int(now.Month())
	t.Year = now.Year()
	t.Hour = now.Hour()
	t.Min = now.Minute()
	t.Sec = now.Second()
	t.Dst = ws.isDST(now)

	var errors uint = 0
	for _, item := range list {
		data, ok := item.(HasErrors)
		if ok {
			errors += data.GetErrors()
		}
	}
	raspiRpm, _ := raspi.GetFanRPM()
	raspiCpu, _ := raspi.GetCpuUsage()
	raspiTemp, _ := raspi.GetTemperature()
	raspiVoltage, _ := raspi.Get5VVoltage()
	raspiVersion, _ := raspi.GetSysVersion()
	raspiThrottle, _ := raspi.GetThrottled()
	raspiSerial, _ := raspi.GetSerial()
	raspiRamUsed, raspiRamTotal, _ := raspi.GetRam()
	raspiDiskFree, raspiDiskTotal, _ := raspi.GetDiskSpace()

	dj := dataJson{
		Config: config{
			UpdateInterval: ws.cfg.GetIntValue(cfg.SECTION_GENERAL, cfg.KEY_GENERAL_UPDATE_INTERVAL, cfg.DEF_UPDATE_INTERVAL),
		},
		Status: status{
			Version:                cfg.WEATHER_STATION_VERSION,
			Uptime:                 ws.getUptime(),
			Time:                   t,
			PressureTrend1:         data.Pressure.PressureTrend1,
			PressureTrend1Goodness: data.Pressure.PressureTrend1Goodness,
			PressureTrend3:         data.Pressure.PressureTrend3,
			PressureTrend3Goodness: data.Pressure.PressureTrend3Goodness,
			Errors:                 errors,
			Resets:                 uint(ws.cfg.GetIntValue(cfg.SECTION_RUN, cfg.KEY_RUN_REBOOT, 0)),
			Sd: sd{
				Free:  raspiDiskFree,
				Total: raspiDiskTotal,
			},
			RaspiStatus: raspiStatus{
				Fan:         raspiRpm,
				CpuUsage:    int(raspiCpu),
				Temperature: raspiTemp / 100,
				Voltage_5v:  raspiVoltage,
				Serial:      raspiSerial,
				Version:     raspiVersion,
				Throttle:    raspiThrottle,
				Ram_used:    raspiRamUsed,
				Ram_total:   raspiRamTotal,
			},
		},
		Sensors: list,
	}
	b, _ := json.Marshal(dj)
	// log.Print(string(b))
	w.Write(b)
}

func copyFile(src, dest string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()

	d, err := os.Create(dest)
	if err != nil {
		return err
	}

	_, err = io.Copy(d, s)
	if err != nil {
		d.Close()
		return err
	}
	return d.Close()
}

func (ws *WebServer) fwUpdate(w http.ResponseWriter, r *http.Request) {
	file, header, err := r.FormFile("firmwarefile")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	tmpFile := "/tmp/weatherd.new"

	dst, err := os.Create(tmpFile)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	n, err := io.Copy(dst, file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if n != header.Size {
		http.Error(w, fmt.Errorf("size mismatch").Error(), http.StatusInternalServerError)
		return
	}

	path, err := os.Readlink("/proc/self/exe")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	oldPath := path + "_old"

	err = os.Remove(oldPath)
	if err != nil && !os.IsNotExist(err) {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = os.Rename(path, oldPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = copyFile(tmpFile, path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = os.Chmod(path, 0o755)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Firmware update successful"))
	go func() {
		time.Sleep(1 * time.Second)
		exec.Command("systemctl", "poweroff").Run()
	}()
}

func (ws *WebServer) restart(w http.ResponseWriter, r *http.Request) {
	raspi.Restart(ws.cfg)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Restart sheduled"))
}

func (ws *WebServer) setCfg(w http.ResponseWriter, r *http.Request) {
	err := ws.cfg.SetCfg(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	} else {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}
}

func (ws *WebServer) indexHandler(w http.ResponseWriter, _ *http.Request) {
	data, err := fs.ReadFile(webFiles, "web/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	name := ws.cfg.GetStringValue(cfg.SECTION_GENERAL, cfg.KEY_GENERAL_NAME, cfg.DEF_NAME)
	data = []byte(strings.ReplaceAll(string(data), "@NAME@", name))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

func (ws *WebServer) log(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL.RequestURI())
		next.ServeHTTP(w, r)
	})
}

func (ws *WebServer) getSd(w http.ResponseWriter, r *http.Request) {
	file := r.URL.Query().Get("file")
	if file == "" {
		http.Error(w, "missing file", http.StatusBadRequest)
		return
	}
	f, info, err := ws.sdData.GetSd(file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Disposition",
		fmt.Sprintf(`attachment; filename="%s"`, info.Name()))
	http.ServeContent(w, r, info.Name(), info.ModTime(), f)
}

func (ws *WebServer) listSd(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	dir := r.URL.Query().Get("dir")
	json, err := ws.sdData.ListSd(dir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	_, _ = w.Write([]byte(json))
}

func (ws *WebServer) getData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	now := time.Now()
	start := r.URL.Query().Get("start")
	var startTime time.Time
	var endTime time.Time
	var err error
	if start == "" {
		startTime = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	} else {
		startTime, err = time.ParseInLocation("2006-01-02_15-04-05", start, time.Local)
		if err != nil {
			startTime, err = time.ParseInLocation("2006-01-02", start, time.Local)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
	}

	end := r.URL.Query().Get("end")
	if end == "" {
		endTime = now
	} else {
		endTime, err = time.ParseInLocation("2006-01-02_15-04-05", end, time.Local)
		if err != nil {
			endTime, err = time.ParseInLocation("2006-01-02", end, time.Local)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			endTime = endTime.Add(24*time.Hour - time.Second)
		}
	}
	data, err := ws.sdData.GetData(startTime, endTime)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_, _ = w.Write(data)
}

func (ws *WebServer) StartWebServer(done chan<- error) *http.Server {
	mux := http.NewServeMux()
	webFS, err := fs.Sub(webFiles, "web")
	if err != nil {
		done <- err
		return nil
	}

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			ws.indexHandler(w, r)
			return
		}
		http.FileServer(http.FS(webFS)).ServeHTTP(w, r)
	})
	mux.HandleFunc("GET /index.html", ws.indexHandler)

	mux.HandleFunc("GET /getcfg", ws.getCfg)
	mux.HandleFunc("GET /update", ws.getUpdate)
	mux.HandleFunc("POST /setcfg", ws.setCfg)
	mux.HandleFunc("POST /firmwareupdate", ws.fwUpdate)
	mux.HandleFunc("GET /restart", ws.restart)
	mux.HandleFunc("GET /listsd", ws.listSd)
	mux.HandleFunc("GET /getsd", ws.getSd)
	mux.HandleFunc("GET /getdata", ws.getData)

	handler := ws.auth(mux)
	handler = ws.log(handler)

	ws.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", ws.port),
		Handler: handler,
	}

	go func() {
		err := ws.server.ListenAndServe()
		done <- err
	}()

	return ws.server
}

func (ws *WebServer) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ws.cfg.GetBoolValue(cfg.SECTION_GENERAL, cfg.KEY_GENERAL_USE_PASSWORD, cfg.DEF_USE_PASSWORD) &&
			ws.cfg.GetStringValue(cfg.SECTION_GENERAL, cfg.KEY_GENERAL_PASSWORD, "") != "" {
			_, password, ok := r.BasicAuth()

			realm := fmt.Sprintf("Basic realm=\"%s\"", ws.cfg.GetStringValue(cfg.SECTION_GENERAL, cfg.KEY_GENERAL_NAME, cfg.DEF_NAME))
			if !ok || !helper.CheckPassword(password, ws.cfg.GetStringValue(cfg.SECTION_GENERAL, cfg.KEY_GENERAL_PASSWORD, "")) {
				w.Header().Set("WWW-Authenticate", realm)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
