package cfg

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"golang.org/x/crypto/argon2"
	"gopkg.in/ini.v1"
)

const (
	WEATHER_STATION_VERSION = "2.0"

	DEF_TEMPERATURE_FC  = 0.2   // nach ~ 6 Werten 75%,
	DEF_TEMPERATURE_FC2 = 0.002 // nach ~ 693 Werten 75%,
	DEF_HUMIDITY_FC     = 0.1   // nach 13 Werten 75%
	DEF_HUMIDITY_FC2    = 0.002
	DEF_DEWPOINT_FC     = 0.1
	DEF_DEWPOINT_FC2    = 0.002
	DEF_PRESSURE_FC     = 0.1
	DEF_LUX_FC          = 0.3 // nach 4 Werten 75%
	DEF_LUX_FC2         = 0.004

	SECTION_FILTERDATA   = "filterdata"
	KEY_FILTER_CONST_T   = "filter_constant_t"
	KEY_FILTER_CONST_T2  = "filter_constant_t2"
	KEY_FILTER_CONST_H   = "filter_constant_h"
	KEY_FILTER_CONST_H2  = "filter_constant_h2"
	KEY_FILTER_CONST_DP  = "filter_constant_dp"
	KEY_FILTER_CONST_DP2 = "filter_constant_dp2"
	KEY_FILTER_CONST_P   = "filter_constant_p"
	KEY_FILTER_CONST_L   = "filter_constant_l"
	KEY_FILTER_CONST_L2  = "filter_constant_l2"

	SECTION_GENERAL             = "general"
	KEY_GENERAL_HEIGHT          = "height"
	KEY_GENERAL_NAME            = "Name"
	KEY_GENERAL_DNS_NAME        = "DnsName"
	KEY_GENERAL_USE_PASSWORD    = "UsePassword"
	KEY_GENERAL_PASSWORD        = "Password"
	KEY_GENERAL_UPDATE_INTERVAL = "UpdateInterval"

	DEF_HEIGHT          = 515
	DEF_NAME            = "Weather Station"
	DEF_DNS_NAME        = "weather-station"
	DEF_USE_PASSWORD    = true
	DEF_UPDATE_INTERVAL = 5000
)

type Cfg struct {
	ini      *ini.File
	doReload func()
	cfgFile  string
}

func NewCfg(file string, doReload func()) (*Cfg, error) {
	cfg, err := ini.Load(file)
	if err != nil {
		cfg = ini.Empty()
	}
	return &Cfg{
		ini:     cfg,
		cfgFile: file,
	}, nil
}

func (c *Cfg) Save() error {
	err := c.ini.SaveTo(c.cfgFile)
	if err != nil {
		return err
	}
	if c.doReload != nil {
		c.doReload()
	}
	return nil
}

// Getter
func (c *Cfg) GetIntValue(section, key string, def int) int {
	return c.ini.Section(section).Key(key).MustInt(def)
}

func (c *Cfg) GetStringValue(section, key string, def string) string {
	return c.ini.Section(section).Key(key).MustString(def)
}

func (c *Cfg) GetFloatValue(section, key string, def float32) float32 {
	return float32(c.ini.Section(section).Key(key).MustFloat64(float64(def)))
}

func (c *Cfg) GetBoolValue(section, key string, def bool) bool {
	return c.ini.Section(section).Key(key).MustBool(def)
}

// Setter
func (c *Cfg) SetStringValue(section, key, value, comment string) {
	c.ini.Section(section).Key(key).SetValue(value)
	if comment != "" {
		c.ini.Section(section).Key(key).Comment = comment
	}
}

func (c *Cfg) SetIntValue(section, key string, value int, comment string) {
	c.ini.Section(section).Key(key).SetValue(strconv.Itoa(value))
	if comment != "" {
		c.ini.Section(section).Key(key).Comment = comment
	}
}

func (c *Cfg) SetFloatValue(section, key string, value float32, comment string) {
	c.ini.Section(section).Key(key).SetValue(strconv.FormatFloat(float64(value), 'f', -1, 32))
	if comment != "" {
		c.ini.Section(section).Key(key).Comment = comment
	}
}

func (c *Cfg) SetBoolValue(section, key string, value bool, comment string) {
	c.ini.Section(section).Key(key).SetValue(strconv.FormatBool(value))
	if comment != "" {
		c.ini.Section(section).Key(key).Comment = comment
	}
}

func (c *Cfg) SetDeaultCfg() {
	c.ini = ini.Empty()
	c.SetFloatValue(SECTION_FILTERDATA, KEY_FILTER_CONST_T, DEF_TEMPERATURE_FC, "Temperatur filter")
	c.SetFloatValue(SECTION_FILTERDATA, KEY_FILTER_CONST_T2, DEF_TEMPERATURE_FC2, "Temperatur filter 2")
	c.SetFloatValue(SECTION_FILTERDATA, KEY_FILTER_CONST_H, DEF_HUMIDITY_FC, "Humidity filter")
	c.SetFloatValue(SECTION_FILTERDATA, KEY_FILTER_CONST_H2, DEF_HUMIDITY_FC2, "Humidity filter 2")
	c.SetFloatValue(SECTION_FILTERDATA, KEY_FILTER_CONST_DP, DEF_DEWPOINT_FC, "Dew point filter")
	c.SetFloatValue(SECTION_FILTERDATA, KEY_FILTER_CONST_DP2, DEF_DEWPOINT_FC2, "Dew point filter 2")
	c.SetFloatValue(SECTION_FILTERDATA, KEY_FILTER_CONST_L, DEF_LUX_FC, "Lux filter")
	c.SetFloatValue(SECTION_FILTERDATA, KEY_FILTER_CONST_L2, DEF_LUX_FC2, "Lux filter 2")
	c.SetFloatValue(SECTION_FILTERDATA, KEY_FILTER_CONST_P, DEF_PRESSURE_FC, "Pressure filter")

	c.SetIntValue(SECTION_GENERAL, KEY_GENERAL_HEIGHT, DEF_HEIGHT, "Height of station ")
	c.SetStringValue(SECTION_GENERAL, KEY_GENERAL_NAME, DEF_NAME, "Name of station (title in browser)")
	c.SetStringValue(SECTION_GENERAL, KEY_GENERAL_DNS_NAME, DEF_DNS_NAME, "DNS name")
	c.SetBoolValue(SECTION_GENERAL, KEY_GENERAL_USE_PASSWORD, DEF_USE_PASSWORD, "Use password")
	c.SetStringValue(SECTION_GENERAL, KEY_GENERAL_PASSWORD, "", "Password hash")
	c.SetIntValue(SECTION_GENERAL, KEY_GENERAL_UPDATE_INTERVAL, DEF_UPDATE_INTERVAL, "Main page update interval [msec]")
}

func (c *Cfg) GetJson() []byte {
	type filterdata struct {
		T_fc  float32 `json:"t_fc"`
		T_fc2 float32 `json:"t_fc2"`
		H_fc  float32 `json:"h_fc"`
		H_fc2 float32 `json:"h_fc2"`
		D_fc  float32 `json:"d_fc"`
		D_fc2 float32 `json:"d_fc2"`
		L_fc  float32 `json:"l_fc"`
		L_fc2 float32 `json:"l_fc2"`
		P_fc2 float32 `json:"p_fc"`
	}
	f := filterdata{
		T_fc:  c.GetFloatValue(SECTION_FILTERDATA, KEY_FILTER_CONST_T, DEF_TEMPERATURE_FC),
		T_fc2: c.GetFloatValue(SECTION_FILTERDATA, KEY_FILTER_CONST_T2, DEF_TEMPERATURE_FC2),
		H_fc:  c.GetFloatValue(SECTION_FILTERDATA, KEY_FILTER_CONST_H, DEF_HUMIDITY_FC),
		H_fc2: c.GetFloatValue(SECTION_FILTERDATA, KEY_FILTER_CONST_H2, DEF_HUMIDITY_FC2),
		D_fc:  c.GetFloatValue(SECTION_FILTERDATA, KEY_FILTER_CONST_DP, DEF_DEWPOINT_FC),
		D_fc2: c.GetFloatValue(SECTION_FILTERDATA, KEY_FILTER_CONST_DP2, DEF_DEWPOINT_FC2),
		L_fc:  c.GetFloatValue(SECTION_FILTERDATA, KEY_FILTER_CONST_L, DEF_LUX_FC),
		L_fc2: c.GetFloatValue(SECTION_FILTERDATA, KEY_FILTER_CONST_L2, DEF_LUX_FC2),
		P_fc2: c.GetFloatValue(SECTION_FILTERDATA, KEY_FILTER_CONST_P, DEF_PRESSURE_FC),
	}
	type general struct {
		Height      int    `json:"height"`
		Name        string `json:"name"`
		DnsName     string `json:"dns_name"`
		UsePassword bool   `json:"use_pass"`
	}
	g := general{
		Height:      c.GetIntValue(SECTION_GENERAL, KEY_GENERAL_HEIGHT, DEF_HEIGHT),
		Name:        c.GetStringValue(SECTION_GENERAL, KEY_GENERAL_NAME, DEF_NAME),
		DnsName:     c.GetStringValue(SECTION_GENERAL, KEY_GENERAL_DNS_NAME, DEF_DNS_NAME),
		UsePassword: c.GetBoolValue(SECTION_GENERAL, KEY_GENERAL_USE_PASSWORD, DEF_USE_PASSWORD),
	}
	data := map[string]any{
		"filterdata": f,
		"general":    g,
	}

	b, _ := json.Marshal(data)
	log.Print(string(b))
	return b
}

func (c *Cfg) SetCfg(r *http.Request) error {
	var allErrors error = nil

	val := r.FormValue("t_fc")
	if val != "" {
		f, err := strconv.ParseFloat(val, 32)
		if err == nil {
			c.SetFloatValue(SECTION_FILTERDATA, KEY_FILTER_CONST_T, float32(f), "")
		} else {
			allErrors = errors.Join(allErrors, err)
		}
	}
	val = r.FormValue("t_fc2")
	if val != "" {
		f, err := strconv.ParseFloat(val, 32)
		if err == nil {
			c.SetFloatValue(SECTION_FILTERDATA, KEY_FILTER_CONST_T2, float32(f), "")
		} else {
			allErrors = errors.Join(allErrors, err)
		}
	}

	val = r.FormValue("h_fc")
	if val != "" {
		f, err := strconv.ParseFloat(val, 32)
		if err == nil {
			c.SetFloatValue(SECTION_FILTERDATA, KEY_FILTER_CONST_H, float32(f), "")
		} else {
			allErrors = errors.Join(allErrors, err)
		}
	}
	val = r.FormValue("h_fc2")
	if val != "" {
		f, err := strconv.ParseFloat(val, 32)
		if err == nil {
			c.SetFloatValue(SECTION_FILTERDATA, KEY_FILTER_CONST_H2, float32(f), "")
		} else {
			allErrors = errors.Join(allErrors, err)
		}
	}

	val = r.FormValue("d_fc")
	if val != "" {
		f, err := strconv.ParseFloat(val, 32)
		if err == nil {
			c.SetFloatValue(SECTION_FILTERDATA, KEY_FILTER_CONST_DP, float32(f), "")
		} else {
			allErrors = errors.Join(allErrors, err)
		}
	}
	val = r.FormValue("d_fc2")
	if val != "" {
		f, err := strconv.ParseFloat(val, 32)
		if err == nil {
			c.SetFloatValue(SECTION_FILTERDATA, KEY_FILTER_CONST_DP2, float32(f), "")
		} else {
			allErrors = errors.Join(allErrors, err)
		}
	}

	val = r.FormValue("l_fc")
	if val != "" {
		f, err := strconv.ParseFloat(val, 32)
		if err == nil {
			c.SetFloatValue(SECTION_FILTERDATA, KEY_FILTER_CONST_L, float32(f), "")
		} else {
			allErrors = errors.Join(allErrors, err)
		}
	}
	val = r.FormValue("l_fc2")
	if val != "" {
		f, err := strconv.ParseFloat(val, 32)
		if err == nil {
			c.SetFloatValue(SECTION_FILTERDATA, KEY_FILTER_CONST_L2, float32(f), "")
		} else {
			allErrors = errors.Join(allErrors, err)
		}
	}

	val = r.FormValue("p_fc")
	if val != "" {
		f, err := strconv.ParseFloat(val, 32)
		if err == nil {
			c.SetFloatValue(SECTION_FILTERDATA, KEY_FILTER_CONST_P, float32(f), "")
		} else {
			allErrors = errors.Join(allErrors, err)
		}
	}

	val = r.FormValue("name")
	if val != "" {
		c.SetStringValue(SECTION_GENERAL, KEY_GENERAL_NAME, val, "")
	}
	val = r.FormValue("dns_name")
	if val != "" {
		c.SetStringValue(SECTION_GENERAL, KEY_GENERAL_DNS_NAME, val, "")
	}
	val = r.FormValue("height")
	if val != "" {
		h, err := strconv.Atoi(val)
		if err == nil {
			c.SetIntValue(SECTION_GENERAL, KEY_GENERAL_HEIGHT, h, "")
		} else {
			allErrors = errors.Join(allErrors, err)
		}
	}
	val = r.FormValue("use_pass")
	if val != "" {
		h, err := strconv.Atoi(val)
		if err == nil {
			c.SetBoolValue(SECTION_GENERAL, KEY_GENERAL_USE_PASSWORD, h == 1, "")
		} else {
			allErrors = errors.Join(allErrors, err)
		}
	}
	val = r.FormValue("pass")
	if val != "" {
		val, err := c.hashPassword(val)
		if err == nil {
			c.SetStringValue(SECTION_GENERAL, KEY_GENERAL_PASSWORD, val, "")
		} else {
			allErrors = errors.Join(allErrors, err)
		}
	}
	err := c.Save()
	if err != nil {
		allErrors = errors.Join(allErrors, err)
	}
	return allErrors
}

func (c *Cfg) hashPassword(password string) (string, error) {
	const (
		saltLen  = 16
		timeCost = 1
		memory   = 64 * 1024 // 64 MiB
		threads  = 4
		keyLen   = 32
	)

	salt := make([]byte, saltLen)

	_, err := rand.Read(salt)
	if err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, timeCost, memory, threads, keyLen)
	salt64 := base64.RawStdEncoding.EncodeToString(salt)
	hash64 := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", memory, timeCost, threads, salt64, hash64), nil
}
