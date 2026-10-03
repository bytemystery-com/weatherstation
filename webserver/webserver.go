package webserver

import (
	"crypto/subtle"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"weatherd/cfg"

	"golang.org/x/crypto/argon2"
)

type WebData struct {
	SensorName string `json:"name"`
	Errors     int    `json:"errors"`
	Valid      bool   `json:"valid"`
}

type WebDataLux struct {
	WebData
	Lux  float32 `json:"lux"`
	Lux2 float32 `json:"lux2"`
}

type WebDataPressure struct {
	WebData
	Temperature    float32 `json:"temperature"`
	Temperature2   float32 `json:"temperature2"`
	Pressure       float32 `json:"pressure"`
	PressureNormal float32 `json:"pressure_normal"`
}

type WebDataAll struct {
	Lux      WebDataLux
	Pressure WebDataPressure
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
}

func NewWebServer(port int, callBacks WebCallBacks, config *cfg.Cfg) *WebServer {
	return &WebServer{
		callBacks: callBacks,
		port:      port,
		cfg:       config,
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
		Free        uint64 `json:"free"`
		Total       uint64 `json:"total"`
		Size        uint64 `json:"size"`
		ClusterSize uint   `json:"cluster_size"`
		SectorSize  uint   `json:"sector_size"`
	}

	type status struct {
		Version                string  `json:"version"`
		Uptime                 string  `json:"uptime"`
		PressureTrend1         float32 `json:"pressure_trend_1"`
		PressureTrend3         float32 `json:"pressure_trend_3"`
		PressureTrend1Goodness float32 `json:"pressure_trend_1_goodness"`
		PressureTrend3Goodness float32 `json:"pressure_trend_3_goodness"`
		Time                   timed   `json:"time"`
		Sd                     sd      `json:"sd"`
		Resets                 int     `json:"resets"`
		Errors                 int     `json:"errors"`
	}

	type dataJson struct {
		Config  config `json:"config"`
		Status  status `json:"status"`
		Sensors []any  `json:"sensors"`
	}
	list := make([]any, 0, 3)
	list = append(list, data.Pressure)
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

	dj := dataJson{
		Config: config{
			UpdateInterval: ws.cfg.GetIntValue(cfg.SECTION_GENERAL, cfg.KEY_GENERAL_UPDATE_INTERVAL, cfg.DEF_UPDATE_INTERVAL),
		},
		Status: status{
			Version:        cfg.WEATHER_STATION_VERSION,
			Uptime:         ws.getUptime(),
			Time:           t,
			PressureTrend1: 0.1,
			PressureTrend3: -0.3,
		},
		Sensors: list,
	}
	b, _ := json.Marshal(dj)
	log.Print(string(b))
	w.Write(b)
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
	handler := ws.auth(mux)

	http.ListenAndServe(":8080", handler)

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
			if !ok || !ws.checkPassword(password, ws.cfg.GetStringValue(cfg.SECTION_GENERAL, cfg.KEY_GENERAL_PASSWORD, "")) {
				w.Header().Set("WWW-Authenticate", realm)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (ws *WebServer) checkPassword(password, encodedHash string) bool {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false
	}

	if parts[1] != "argon2id" || parts[2] != "v=19" {
		return false
	}

	// m=65536,t=1,p=4
	params := strings.Split(parts[3], ",")
	if len(params) != 3 {
		return false
	}

	memory, err := strconv.ParseUint(strings.TrimPrefix(params[0], "m="), 10, 32)
	if err != nil {
		return false
	}

	iterations, err := strconv.ParseUint(strings.TrimPrefix(params[1], "t="), 10, 32)
	if err != nil {
		return false
	}

	parallelism, err := strconv.ParseUint(strings.TrimPrefix(params[2], "p="), 10, 8)
	if err != nil {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}

	actualHash := argon2.IDKey([]byte(password), salt, uint32(iterations),
		uint32(memory), uint8(parallelism), uint32(len(expectedHash)))

	return subtle.ConstantTimeCompare(actualHash, expectedHash) == 1
}
