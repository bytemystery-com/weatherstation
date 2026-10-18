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
	"crypto/tls"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"weatherd/cfg"
	"weatherd/helper"
	"weatherd/raspi"
	"weatherd/sd"
)

//go:embed ssl/weatherd.crt
var internalSslCert []byte

//go:embed ssl/weatherd.key
var internalSslKey []byte

const (
	WS_SESSION_MAX_AGE         = 14400 // sec
	WS_REMOVE_SESSION_INTERVAL = 900   // sec
	WS_SESSION_COOKIE_NAME     = "session_id"
	WS_SESSION_COOKIE_NAME_SSL = "session_id-ssl"
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
	WetBulb  float32
	WetBulb2 float32
}

type WebCallBacks interface {
	GetData() *WebDataAll
}

//go:embed web/*
var webFiles embed.FS

type WebServer struct {
	callBacks    WebCallBacks
	httpServer   *http.Server
	httpsServer  *http.Server
	cfg          *cfg.Cfg
	sdData       *sd.SdData
	sessions     map[string]time.Time
	sessionsLock sync.RWMutex
}

func NewWebServer(callBacks WebCallBacks, config *cfg.Cfg, sdData *sd.SdData) *WebServer {
	return &WebServer{
		callBacks: callBacks,
		cfg:       config,
		sdData:    sdData,
		sessions:  make(map[string]time.Time, 10),
	}
}

func (ws *WebServer) getUptime() string {
	seconds, err := raspi.GetUptime()
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
		Fan          int    `json:"fan"`
		CpuUsage     int    `json:"cpu"`
		CpuFrequency int    `json:"cpu_freq"` // MHz
		Temperature  int    `json:"temp"`
		Voltage_5v   int    `json:"voltage_5"`
		Voltage_3v   int    `json:"voltage_3"`
		Serial       string `json:"serial"`
		Version      uint16 `json:"version"`
		Throttle     uint32 `json:"throttle"`
		Ram_used     uint32 `json:"ram_u"`
		Ram_total    uint32 `json:"ram_t"`
		Max_current  uint16 `json:"max_current"`
		WifiStrength int    `json:"wifi_strength"`
		WifiRx       uint64 `json:"wifi_rx"`
		WifiTx       uint64 `json:"wifi_tx"`
	}
	type status struct {
		Version                string      `json:"version"`
		Uptime                 string      `json:"uptime"`
		PressureTrend1         float32     `json:"pressure_trend_1"`
		PressureTrend3         float32     `json:"pressure_trend_3"`
		PressureTrend1Goodness float32     `json:"pressure_trend_1_goodness"`
		PressureTrend3Goodness float32     `json:"pressure_trend_3_goodness"`
		WetBulb                float32     `json:"wet_bulb"`
		WetBulb2               float32     `json:"wet_bulb2"`
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

	var errors uint
	for _, item := range list {
		data, ok := item.(HasErrors)
		if ok {
			errors += data.GetErrors()
		}
	}
	raspiRpm, _ := raspi.GetFanRPM()
	raspiCpu, _ := raspi.GetCpuUsage()
	raspiCpuFreq, _ := raspi.GetCPUFrequency()
	raspiTemp, _ := raspi.GetTemperature()
	raspiVoltage5, raspiVoltage3, _ := raspi.GetVoltages()
	raspiVersion, _ := raspi.GetSysVersion()
	raspiThrottle, _ := raspi.GetThrottled()
	raspiSerial, _ := raspi.GetSerial()
	raspiRamUsed, raspiRamTotal, _ := raspi.GetRam()
	raspiDiskFree, raspiDiskTotal, _ := raspi.GetDiskSpace()
	raspiMaxCurrent, _ := raspi.GetMaxCurrent()
	raspiWifiStrength, raspiWifiRx, raspiWifiTx, _ := raspi.GetWifiInfo("wlan0")

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
			WetBulb:                data.WetBulb,
			WetBulb2:               data.WetBulb2,
			Errors:                 errors,
			Resets:                 uint(ws.cfg.GetIntValue(cfg.SECTION_RUN, cfg.KEY_RUN_REBOOT, 0)),
			Sd: sd{
				Free:  raspiDiskFree,
				Total: raspiDiskTotal,
			},
			RaspiStatus: raspiStatus{
				Fan:          raspiRpm,
				CpuUsage:     int(raspiCpu),
				CpuFrequency: int(raspiCpuFreq / 1000000),
				Temperature:  raspiTemp / 100,
				Voltage_5v:   raspiVoltage5,
				Voltage_3v:   raspiVoltage3,
				Serial:       raspiSerial,
				Version:      raspiVersion,
				Throttle:     raspiThrottle,
				Ram_used:     raspiRamUsed,
				Ram_total:    raspiRamTotal,
				Max_current:  raspiMaxCurrent,
				WifiStrength: raspiWifiStrength,
				WifiRx:       raspiWifiRx,
				WifiTx:       raspiWifiTx,
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
		raspi.Restart(ws.cfg)
	}()
}

func (ws *WebServer) restart(w http.ResponseWriter, r *http.Request) {
	raspi.Restart(ws.cfg)
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Restart sheduled"))
}

func (ws *WebServer) setCfg(w http.ResponseWriter, r *http.Request) {
	oldPass := ws.cfg.GetStringValue(cfg.SECTION_GENERAL, cfg.KEY_GENERAL_PASSWORD, "")
	err := ws.cfg.SetCfg(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	} else {
		if oldPass != ws.cfg.GetStringValue(cfg.SECTION_GENERAL, cfg.KEY_GENERAL_PASSWORD, "") {
			ws.sessionsLock.Lock()
			clear(ws.sessions)
			ws.sessionsLock.Unlock()
		}
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

/*
func (ws *WebServer) log(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL.RequestURI())
		next.ServeHTTP(w, r)
	})
}
*/

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

	if strings.EqualFold(filepath.Ext(info.Name()), ".png") {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Disposition", "inline")
	} else {
		w.Header().Set("Content-Disposition",
			fmt.Sprintf(`attachment; filename="%s"`, info.Name()))
	}
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

func (ws *WebServer) query(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	q := r.URL.Query().Get("q")
	data := ws.callBacks.GetData()
	response := ""
	switch q {
	case "t":
		response = fmt.Sprintf("%.1f", data.Humidity.Temperature)
	case "t2":
		response = fmt.Sprintf("%.1f", data.Humidity.Temperature2)
	case "h":
		response = fmt.Sprintf("%.1f", data.Humidity.Humidity)
	case "h2":
		response = fmt.Sprintf("%.1f", data.Humidity.Humidity2)
	case "d":
		response = fmt.Sprintf("%.1f", data.Humidity.DewPoint)
	case "d2":
		response = fmt.Sprintf("%.1f", data.Humidity.DewPoint2)
	case "p":
		response = fmt.Sprintf("%.1f", data.Pressure.Pressure)
	case "pn":
		response = fmt.Sprintf("%.1f", data.Pressure.PressureNormal)
	case "p2":
		response = fmt.Sprintf("%.1f", data.Pressure.Pressure2)
	case "pn2":
		response = fmt.Sprintf("%.1f", data.Pressure.Pressure2Normal)
	case "l":
		response = fmt.Sprintf("%.1f", data.Lux.Lux)
	case "l2":
		response = fmt.Sprintf("%.1f", data.Lux.Lux2)
	case "pt1":
		response = fmt.Sprintf("%.2f", data.Pressure.PressureTrend1)
	case "pt3":
		response = fmt.Sprintf("%.2f", data.Pressure.PressureTrend3)
	case "tp":
		response = fmt.Sprintf("%.1f", data.Pressure.Temperature)
	case "tp2":
		response = fmt.Sprintf("%.1f", data.Pressure.Temperature2)
	case "v":
		response = fmt.Sprintf("%s", cfg.WEATHER_STATION_VERSION)
	case "r_temperature":
		val, err := raspi.GetTemperature()
		if err == nil {
			response = fmt.Sprintf("%.1f", float32(val)/1000.0)
		}
	case "r_fan":
		val, err := raspi.GetFanRPM()
		if err == nil {
			response = fmt.Sprintf("%d", val)
		}
	case "r_cpu_usage":
		val, err := raspi.GetCpuUsage()
		if err == nil {
			response = fmt.Sprintf("%.1f", float32(val)/10.0)
		}
	case "r_cpu_frequency":
		val, err := raspi.GetCPUFrequency()
		if err == nil {
			response = fmt.Sprintf("%.0f", float32(val)/1000000.0)
		}
	case "r_ram_used":
		val, _, err := raspi.GetRam()
		if err == nil {
			response = fmt.Sprintf("%d", val)
		}
	case "r_ram_total":
		_, val, err := raspi.GetRam()
		if err == nil {
			response = fmt.Sprintf("%d", val)
		}
	case "r_voltage_5":
		val, err := raspi.Get5VVoltage()
		if err == nil {
			response = fmt.Sprintf("%.2f", float32(val)/1000.0)
		}
	case "r_voltage_33":
		val, err := raspi.Get3V3()
		if err == nil {
			response = fmt.Sprintf("%.2f", float32(val)/1000.0)
		}
	case "r_throttle":
		val, err := raspi.GetThrottled()
		if err == nil {
			response = fmt.Sprintf("0x%08x", val)
		}
	case "r_serial":
		val, err := raspi.GetSerial()
		if err == nil {
			response = fmt.Sprintf("%s", val)
		}
	case "r_sd_free":
		val, _, err := raspi.GetDiskSpace()
		if err == nil {
			response = fmt.Sprintf("%d", val)
		}
	case "r_sd_total":
		_, val, err := raspi.GetDiskSpace()
		if err == nil {
			response = fmt.Sprintf("%d", val)
		}
	case "r_date":
		now := time.Now()
		response = now.Format(time.RFC3339)
	case "r_uptime":
		val, err := raspi.GetUptime()
		if err == nil {
			response = fmt.Sprintf("%d", val)
		}
	case "r_version":
		val, err := raspi.GetSysVersion()
		if err == nil {
			response = fmt.Sprintf("%d.%d", (val>>8)&0xFF, val&0xFF)
		}
	default:
		response = ""
	}
	if response == "" {
		http.Error(w, "Invalid or missing query parameter", http.StatusBadRequest)
		response = "\nMeasure data:\n" +
			"t		Temperature from humidity sensor (short) in °C\n" +
			"t2		Temperature from humidity sensor (long) in °C\n" +
			"h		Humidity (short) in %\n" +
			"h2		Humidity (long) in %\n" +
			"d		Dew point (short) in °C\n" +
			"d2		Dew point (long) in °C\n" +
			"p		Pressure (short) in hPa\n" +
			"p2		Pressure (long) in hPa\n" +
			"pn		Pressure normal (short) in hPa\n" +
			"pn2		Pressure normal (long) in hPa\n" +
			"l		Ambient light (short) in Lu\n" +
			"l2		Ambient light (long) in Lu\n" +
			"pt1		Pressurer trend - 1 hour\n" +
			"pt3		Pressurer trend - 3 hours\n" +
			"\n" +
			"Raspberry data:\n" +
			"r_temperature		Raspberry temperature\n" +
			"r_fan			Raspberry fan in rpm\n" +
			"r_cpu_usage		Raspberry CPU usage in %\n" +
			"r_cpu_frquency		Raspberry CPU frequency in MHz\n" +
			"r_ram_used		Raspberry used RAM in kB\n" +
			"r_ram_total		Raspberry Total RAM in kB\n" +
			"r_voltage_5		Raspberry 5 Volt monitoring in V\n" +
			"r_voltage_33		Raspberry 3.3 Volt monitoring in V\n" +
			"r_throttle		Raspberry throttle\n" +
			"r_serial		Raspberry serial number\n" +
			"r_sd_free		Raspberry free space of SD card in kB\n" +
			"r_sd_total		Raspberry total space of SD card in kB\n" +
			"r_date			Raspberry system date / time in  RFC3339\n" +
			"r_uptime		Raspberry system uptime in sec\n" +
			"r_version		Raspberry system version\n"
		_, _ = w.Write([]byte(response))
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(response))
}

func (ws *WebServer) StartWebServer(done chan<- error) {
	mux := http.NewServeMux()
	webFS, err := fs.Sub(webFiles, "web")
	if err != nil {
		done <- err
		return
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
	mux.HandleFunc("GET /query", ws.query)

	handler := ws.auth(mux)
	// handler = ws.log(handler)

	var httpPort int
	var httpsPort int
	if helper.IsRoot() {
		httpPort = ws.cfg.GetIntValue(cfg.SECTION_WEBSERVER, cfg.KEY_WEBSERVER_HTTP_PORT, cfg.DEF_WEBSERVER_HTTP_PORT)
		httpsPort = ws.cfg.GetIntValue(cfg.SECTION_WEBSERVER, cfg.KEY_WEBSERVER_HTTPS_PORT, cfg.DEF_WEBSERVER_HTTPS_PORT)
	} else {
		httpPort = ws.cfg.GetIntValue(cfg.SECTION_WEBSERVER, cfg.KEY_WEBSERVER_HTTP_PORT_NOROOT, cfg.DEF_WEBSERVER_HTTP_PORT_NOROOT)
		httpsPort = ws.cfg.GetIntValue(cfg.SECTION_WEBSERVER, cfg.KEY_WEBSERVER_HTTPS_PORT_NOROOT, cfg.DEF_WEBSERVER_HTTP_PORT_NOROOT)
	}

	if ws.cfg.GetBoolValue(cfg.SECTION_WEBSERVER, cfg.KEY_WEBSERVER_HTTP_ENABLED, cfg.DEF_WEBSERVER_ENABLED_HTTP) {

		ws.httpServer = &http.Server{
			Addr:    fmt.Sprintf(":%d", httpPort),
			Handler: handler,
		}

		go func() {
			err := ws.httpServer.ListenAndServe()
			done <- err
		}()
	}

	if ws.cfg.GetBoolValue(cfg.SECTION_WEBSERVER, cfg.KEY_WEBSERVER_HTTPS_ENABLED, cfg.DEF_WEBSERVER_ENABLED_HTTPS) {
		ws.httpsServer = &http.Server{
			Addr:    fmt.Sprintf(":%d", httpsPort),
			Handler: handler,
		}

		go func() {
			certFile := ws.cfg.GetStringValue(cfg.SECTION_WEBSERVER, cfg.KEY_WEBSERVER_CERT_FILE, cfg.DEF_WEBSERVER_CERT_FILE)
			keyFile := ws.cfg.GetStringValue(cfg.SECTION_WEBSERVER, cfg.KEY_WEBSERVER_KEY_FILE, cfg.DEF_WEBSERVER_KEY_FILE)
			_, err := tls.LoadX509KeyPair(certFile, keyFile)
			if err == nil {
				log.Print("WEB: Use external key / cert files")
				done <- ws.httpsServer.ListenAndServeTLS(certFile, keyFile)
			} else {
				cert, err := tls.X509KeyPair(internalSslCert, internalSslKey)
				if err == nil {
					ws.httpsServer.TLSConfig = &tls.Config{
						Certificates: []tls.Certificate{cert},
					}
					log.Print("WEB: Use internal key / cert data")
					done <- ws.httpsServer.ListenAndServeTLS("", "")
				} else {
					log.Printf("WEB: Internal key / cert data are invalid %v", err)
				}
			}
		}()
	}

	go func() {
		for {
			time.Sleep(WS_REMOVE_SESSION_INTERVAL * time.Second)
			now := time.Now()
			ws.sessionsLock.Lock()
			// removind during iteration ?
			// https://go.dev/doc/effective_go#for
			for id, ts := range ws.sessions {
				if ts.Add(WS_SESSION_MAX_AGE * time.Second).Before(now) {
					delete(ws.sessions, id)
					// log.Printf("Expired cookie %s removed", id)
				}
			}
			ws.sessionsLock.Unlock()
		}
	}()
}

func (ws *WebServer) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/query":
			next.ServeHTTP(w, r)
			return
		}
		if ws.cfg.GetBoolValue(cfg.SECTION_GENERAL, cfg.KEY_GENERAL_USE_PASSWORD, cfg.DEF_USE_PASSWORD) &&
			ws.cfg.GetStringValue(cfg.SECTION_GENERAL, cfg.KEY_GENERAL_PASSWORD, "") != "" {
			now := time.Now()

			cookieName := WS_SESSION_COOKIE_NAME
			if r.TLS != nil {
				cookieName = WS_SESSION_COOKIE_NAME_SSL
			}

			doPasswordCheck := func() {
				_, password, ok := r.BasicAuth()
				realm := fmt.Sprintf("Basic realm=\"%s\"", ws.cfg.GetStringValue(cfg.SECTION_GENERAL, cfg.KEY_GENERAL_NAME, cfg.DEF_NAME))
				if !ok || !helper.CheckPassword(password, ws.cfg.GetStringValue(cfg.SECTION_GENERAL, cfg.KEY_GENERAL_PASSWORD, "")) {
					w.Header().Set("WWW-Authenticate", realm)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					// log.Printf("WS: Need password %v", r.URL)
					return
				}
				sessionId, err := helper.NewSessionID()
				if err != nil {
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				ws.sessionsLock.Lock()
				ws.sessions[sessionId] = now
				ws.sessionsLock.Unlock()

				http.SetCookie(w, &http.Cookie{
					Name:     cookieName,
					Value:    sessionId,
					Path:     "/",
					HttpOnly: true,
					Secure:   r.TLS != nil,
					SameSite: http.SameSiteStrictMode,
					MaxAge:   WS_SESSION_MAX_AGE,
				})
				// log.Printf("WS: Create cookie %s %v", sessionId, r.URL)
				next.ServeHTTP(w, r)
			}

			cookie, err := r.Cookie(cookieName)
			if err != nil {
				// log.Printf("WS: No cookie send %v", r.URL)
				doPasswordCheck()
				return
			}
			ws.sessionsLock.RLock()
			ts, ok := ws.sessions[cookie.Value]
			ws.sessionsLock.RUnlock()
			if !ok {
				// log.Printf("WS: cookie (%v) %v not found", cookie.Value, r.URL)
				doPasswordCheck()
				return
			}
			if ts.Add(WS_SESSION_MAX_AGE * time.Second).Before(now) {
				ws.sessionsLock.Lock()
				delete(ws.sessions, cookie.Value)
				ws.sessionsLock.Unlock()
				// log.Printf("WS: cookie (%v) %v expired", cookie.Value, r.URL)
				doPasswordCheck()
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
