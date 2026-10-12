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

package cfg

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"sync"

	"weatherd/helper"

	"gopkg.in/ini.v1"
)

const (
	WEATHER_STATION_VERSION = "2.7"

	DEF_CFG_FILE = "/etc/weatherd/weatherd.cfg"

	DEF_TEMPERATURE_FC  = 0.2   // nach ~ 6 Werten 75%,
	DEF_TEMPERATURE_FC2 = 0.002 // nach ~ 693 Werten 75%,
	DEF_HUMIDITY_FC     = 0.1   // nach 13 Werten 75%
	DEF_HUMIDITY_FC2    = 0.002
	DEF_DEWPOINT_FC     = 0.1
	DEF_DEWPOINT_FC2    = 0.002
	DEF_PRESSURE_FC     = 0.1
	DEF_PRESSURE_FC2    = 0.002
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
	KEY_FILTER_CONST_P2  = "filter_constant_p2"
	KEY_FILTER_CONST_L   = "filter_constant_l"
	KEY_FILTER_CONST_L2  = "filter_constant_l2"

	SECTION_GENERAL                     = "general"
	KEY_GENERAL_HEIGHT                  = "height"
	KEY_GENERAL_NAME                    = "Name"
	KEY_GENERAL_USE_PASSWORD            = "UsePassword"
	KEY_GENERAL_PASSWORD                = "Password"
	KEY_GENERAL_UPDATE_INTERVAL         = "UpdateInterval"
	KEY_GENERAL_MEASURE_INTERVAL        = "MeasureInterval"
	KEY_GENERAL_HEATER_INTERVAL         = "HeaterInterval"
	KEY_GENERAL_HEATER_MAINT_INTERVAL   = "HeaterMaintInterval"
	KEY_GENERAL_PRESSURE_TREND_INTERVAL = "PressureTrendInterval"

	SECTION_NETWORK             = "network"
	KEY_NETWORK_WEB_PORT        = "WebServerPort"
	KEY_NETWORK_WEB_PORT_NOROOT = "WebServerPortNonRoot"

	SECTION_SYSTEM               = "system"
	KEY_SYSTEM_RESTART_POFF_TIME = "PoweroffTime"

	SECTION_HARDWARE                         = "hardware"
	KEY_HARDWARE_I2C_DEV_NUMBER              = "I2CDevNumber"
	KEY_HARDWARE_SET_PIN_ON_START            = "SetPinOnStart"
	KEY_HARDWARE_PIN_ON_START_GPIODEV_NUMBER = "PinOnStartDeviceNumber"
	KEY_HARDWARE_SET_PIN_ON_START_HIGH       = "SetPinOnStartToHigh"

	SECTION_RUN    = "run"
	KEY_RUN_REBOOT = "Reboots"
	KEY_RUN_HEATER = "LastHeater"

	SECTION_DATA      = "data"
	KEY_DATA_FOLDER   = "Folder"
	KEY_DATA_INTERVAL = "Interval"

	DEF_HEIGHT                      = 522
	DEF_NAME                        = "Weather Station"
	DEF_USE_PASSWORD                = true
	DEF_UPDATE_INTERVAL             = 1000 // msec
	DEF_RESTART_POFF_TIME           = 60   // sec
	DEF_DATA_FOLDER                 = "/data"
	DEF_DATA_INTERVAL               = 300  // sec
	DEF_MEASURE_INTERVAL            = 1000 // msec
	DEF_HEATER_INTERVAL             = 60   // min
	DEF_HEATER_MAINT_INTERVAL       = 168  // Hours
	DEF_PRESSURE_TREND_INTERVAL     = 60   // sec
	DEF_WEBSERVER_PORT              = 80
	DEF_WEBSERVER_PORT_NOROOT       = 8080
	DEF_REBOOT_WAIT                 = 60 // sec
	DEF_I2C_DEV_NUMBER              = 1
	DEF_SET_PIN_ON_START            = -1 // off
	DEF_PIN_ON_START_GPIODEV_NUMBER = 15 // gpiodev15
	DEF_SET_PIN_ON_START_HIGH       = true
)

type Cfg struct {
	ini      *ini.File
	doReload func()
	cfgFile  string
	lock     sync.RWMutex
}

func NewCfg(file string, doReload func()) (*Cfg, error) {
	ok, err := helper.ExistFile(file)
	if err != nil {
		return nil, err
	}
	c := Cfg{
		cfgFile: file,
		ini:     ini.Empty(),
	}
	c.SetDeaultCfg()
	if !ok {
		log.Printf("CFG: New created in %s", file)
		c.Save()
	} else {
		log.Printf("CFG: Loaded from %s", file)
		err := c.ini.Append(file)
		if err != nil {
			return nil, err
		}
		c.Save()
	}
	return &c, nil
}

func (c *Cfg) Save() error {
	c.lock.Lock()
	defer c.lock.Unlock()
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
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.ini.Section(section).Key(key).MustInt(def)
}

func (c *Cfg) GetInt64Value(section, key string, def int64) int64 {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.ini.Section(section).Key(key).MustInt64(def)
}

func (c *Cfg) GetStringValue(section, key string, def string) string {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.ini.Section(section).Key(key).MustString(def)
}

func (c *Cfg) GetFloatValue(section, key string, def float32) float32 {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return float32(c.ini.Section(section).Key(key).MustFloat64(float64(def)))
}

func (c *Cfg) GetBoolValue(section, key string, def bool) bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.ini.Section(section).Key(key).MustBool(def)
}

// Setter
func (c *Cfg) SetStringValue(section, key, value, comment string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.ini.Section(section).Key(key).SetValue(value)
	if comment != "" {
		c.ini.Section(section).Key(key).Comment = comment
	}
}

func (c *Cfg) SetIntValue(section, key string, value int, comment string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.ini.Section(section).Key(key).SetValue(strconv.Itoa(value))
	if comment != "" {
		c.ini.Section(section).Key(key).Comment = comment
	}
}

func (c *Cfg) SetInt64Value(section, key string, value int64, comment string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.ini.Section(section).Key(key).SetValue(strconv.FormatInt(value, 10))
	if comment != "" {
		c.ini.Section(section).Key(key).Comment = comment
	}
}

func (c *Cfg) SetFloatValue(section, key string, value float32, comment string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.ini.Section(section).Key(key).SetValue(strconv.FormatFloat(float64(value), 'f', -1, 32))
	if comment != "" {
		c.ini.Section(section).Key(key).Comment = comment
	}
}

func (c *Cfg) SetBoolValue(section, key string, value bool, comment string) {
	c.lock.Lock()
	defer c.lock.Unlock()
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
	c.SetFloatValue(SECTION_FILTERDATA, KEY_FILTER_CONST_P2, DEF_PRESSURE_FC2, "Pressure filter 2")

	c.SetIntValue(SECTION_GENERAL, KEY_GENERAL_HEIGHT, DEF_HEIGHT, "Height of station ")
	c.SetStringValue(SECTION_GENERAL, KEY_GENERAL_NAME, DEF_NAME, "Name of station (title in browser)")
	c.SetBoolValue(SECTION_GENERAL, KEY_GENERAL_USE_PASSWORD, DEF_USE_PASSWORD, "Use password")
	c.SetStringValue(SECTION_GENERAL, KEY_GENERAL_PASSWORD, "", "Password hash")
	c.SetIntValue(SECTION_GENERAL, KEY_GENERAL_UPDATE_INTERVAL, DEF_UPDATE_INTERVAL, "Main page update interval [msec]")
	c.SetIntValue(SECTION_GENERAL, KEY_GENERAL_MEASURE_INTERVAL, DEF_MEASURE_INTERVAL, "Measure interval [msec]")
	c.SetIntValue(SECTION_GENERAL, KEY_GENERAL_HEATER_INTERVAL, DEF_HEATER_INTERVAL, "Heater interval [min]")
	c.SetIntValue(SECTION_GENERAL, KEY_GENERAL_HEATER_MAINT_INTERVAL, DEF_HEATER_MAINT_INTERVAL, "Heater maintenance interval [hour]")
	c.SetIntValue(SECTION_GENERAL, KEY_GENERAL_PRESSURE_TREND_INTERVAL, DEF_PRESSURE_TREND_INTERVAL, "Interval for pressure trend data [sec]")

	c.SetIntValue(SECTION_NETWORK, KEY_NETWORK_WEB_PORT, DEF_WEBSERVER_PORT, "Port for webser if running as root")
	c.SetIntValue(SECTION_NETWORK, KEY_NETWORK_WEB_PORT_NOROOT, DEF_WEBSERVER_PORT_NOROOT, "Port for webser if not running as root")

	c.SetIntValue(SECTION_SYSTEM, KEY_SYSTEM_RESTART_POFF_TIME, DEF_RESTART_POFF_TIME, "Restart after restart command [sec]")

	c.SetIntValue(SECTION_HARDWARE, KEY_HARDWARE_I2C_DEV_NUMBER, DEF_I2C_DEV_NUMBER, "I2C bus number")
	c.SetIntValue(SECTION_HARDWARE, KEY_HARDWARE_SET_PIN_ON_START, DEF_SET_PIN_ON_START, "Set pin to high on start (-1=disabled)")
	c.SetIntValue(SECTION_HARDWARE, KEY_HARDWARE_PIN_ON_START_GPIODEV_NUMBER, DEF_PIN_ON_START_GPIODEV_NUMBER, "Gpio device used for set pin to high on start")
	c.SetBoolValue(SECTION_HARDWARE, KEY_HARDWARE_SET_PIN_ON_START_HIGH, DEF_SET_PIN_ON_START_HIGH, "If true set pin on start to high, else to low if false")

	c.SetStringValue(SECTION_DATA, KEY_DATA_FOLDER, DEF_DATA_FOLDER, "Folder for data (csv files)")
	c.SetIntValue(SECTION_DATA, KEY_DATA_INTERVAL, DEF_DATA_INTERVAL, "Interval for data write [sec]")

	c.SetIntValue(SECTION_RUN, KEY_RUN_REBOOT, 0, "Number of reboots")
	c.SetInt64Value(SECTION_RUN, KEY_RUN_HEATER, int64(0), "Last timestamp when heater was running")
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
		P_fc  float32 `json:"p_fc"`
		P_fc2 float32 `json:"p_fc2"`
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
		P_fc:  c.GetFloatValue(SECTION_FILTERDATA, KEY_FILTER_CONST_P, DEF_PRESSURE_FC),
		P_fc2: c.GetFloatValue(SECTION_FILTERDATA, KEY_FILTER_CONST_P2, DEF_PRESSURE_FC2),
	}
	type general struct {
		Height      int    `json:"height"`
		Name        string `json:"name"`
		UsePassword bool   `json:"use_pass"`
	}
	g := general{
		Height:      c.GetIntValue(SECTION_GENERAL, KEY_GENERAL_HEIGHT, DEF_HEIGHT),
		Name:        c.GetStringValue(SECTION_GENERAL, KEY_GENERAL_NAME, DEF_NAME),
		UsePassword: c.GetBoolValue(SECTION_GENERAL, KEY_GENERAL_USE_PASSWORD, DEF_USE_PASSWORD),
	}
	data := map[string]any{
		"filterdata": f,
		"general":    g,
	}

	b, _ := json.Marshal(data)
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

	val = r.FormValue("p_fc2")
	if val != "" {
		f, err := strconv.ParseFloat(val, 32)
		if err == nil {
			c.SetFloatValue(SECTION_FILTERDATA, KEY_FILTER_CONST_P2, float32(f), "")
		} else {
			allErrors = errors.Join(allErrors, err)
		}
	}

	val = r.FormValue("name")
	if val != "" {
		c.SetStringValue(SECTION_GENERAL, KEY_GENERAL_NAME, val, "")
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
		val, err := helper.HashPassword(val)
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
