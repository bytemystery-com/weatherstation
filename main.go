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

package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"weatherd/cfg"
	"weatherd/data"
	"weatherd/filter"
	"weatherd/gpio"
	"weatherd/helper"
	"weatherd/i2c"
	"weatherd/raspi"
	"weatherd/sd"
	"weatherd/sensors"
	"weatherd/webserver"

	_ "net/http/pprof"
)

const (
	TRACE_MESURE_DATA = false
)

var Data data.Data

type WebCallbacksImpl struct{}

var _ webserver.WebCallBacks = (*WebCallbacksImpl)(nil)

func (w WebCallbacksImpl) GetData() *webserver.WebDataAll {
	luxErrCount, luxIsValid := Data.Veml7700.GetError()
	humErrCount, humIsValid := Data.Sht45.GetError()
	presErrCount, presIsValid := Data.Bmp581.GetError()
	data := webserver.WebDataAll{
		Lux: webserver.WebDataLux{
			WebData: webserver.WebData{
				SensorName: Data.Veml7700.GetName(),
				Errors:     uint(luxErrCount),
				Valid:      luxIsValid,
			},
			Lux:  Data.F_lux.GetValue(),
			Lux2: Data.F2_lux.GetValue(),
		},

		Pressure: webserver.WebDataPressure{
			WebData: webserver.WebData{
				SensorName: Data.Bmp581.GetName(),
				Errors:     uint(presErrCount),
				Valid:      presIsValid,
			},
			Pressure:               Data.F_pressure.GetValue() / 100.0,
			PressureNormal:         Data.F_pressureNormal.GetValue() / 100.0,
			Pressure2:              Data.F2_pressure.GetValue() / 100.0,
			Pressure2Normal:        Data.F2_pressureNormal.GetValue() / 100.0,
			Temperature:            Data.F_temp2.GetValue(),
			Temperature2:           Data.F2_temp2.GetValue(),
			PressureTrend1:         math.Float32frombits(Data.PressureTrendData.Trend1.Load()) / 100.0,
			PressureTrend1Goodness: math.Float32frombits(Data.PressureTrendData.Goodnness1.Load()),
			PressureTrend3:         math.Float32frombits(Data.PressureTrendData.Trend3.Load()) / 100.0,
			PressureTrend3Goodness: math.Float32frombits(Data.PressureTrendData.Goodnness3.Load()),
		},

		Humidity: webserver.WebDataHumidity{
			WebData: webserver.WebData{
				SensorName: Data.Sht45.GetName(),
				Errors:     uint(humErrCount),
				Valid:      humIsValid,
			},
			Temperature:  Data.F_temp1.GetValue(),
			Temperature2: Data.F2_temp1.GetValue(),
			Humidity:     Data.F_humidity.GetValue(),
			Humidity2:    Data.F2_humidity.GetValue(),
			DewPoint:     Data.F_dp.GetValue(),
			DewPoint2:    Data.F2_dp.GetValue(),
		},
		WetBulb:  Data.F_wetBulb.GetValue(),
		WetBulb2: Data.F2_wetBulb.GetValue(),
	}
	return &data
}

func main() {
	var callBacks WebCallbacksImpl
	var debug bool
	str := os.Getenv("WEATHERD_DEBUG")
	if strings.EqualFold(str, "true") || strings.EqualFold(str, "on") || strings.EqualFold(str, "yes") || strings.EqualFold(str, "1") {
		debug = true
	}

	d := flag.Bool("d", false, "debug")
	flag.Parse()
	if *d {
		debug = true
	}
	/*
		w, err := syslog.New(syslog.LOG_INFO|syslog.LOG_DAEMON, "weatherd")
		if err != nil {
			log.Fatal(err)
		}
		log.SetOutput(w)
	*/
	log.Printf("Debug: %v", debug)
	// journalctl -u weatherd.service

	if !debug {
		log.SetOutput(io.Discard)
	} else {
		// curl http://localhost:6060/debug/pprof/
		//  go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
		go func() {
			http.ListenAndServe("0.0.0.0:6060", nil)
		}()
	}

	cfgFile := os.Getenv("WEATHERD_INI_FILE")
	if cfgFile == "" {
		cfgFile = cfg.DEF_CFG_FILE
	}
	var err error
	Data.Config, err = cfg.NewCfg(cfgFile, func() {
		Data.F_lux.SetFilterConstant(Data.Config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_L, cfg.DEF_LUX_FC))
		Data.F2_lux.SetFilterConstant(Data.Config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_L2, cfg.DEF_LUX_FC2))
		Data.F_temp1.SetFilterConstant(Data.Config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_T, cfg.DEF_TEMPERATURE_FC))
		Data.F2_temp1.SetFilterConstant(Data.Config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_T2, cfg.DEF_TEMPERATURE_FC2))
		Data.F_humidity.SetFilterConstant(Data.Config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_H, cfg.DEF_HUMIDITY_FC))
		Data.F2_humidity.SetFilterConstant(Data.Config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_H2, cfg.DEF_HUMIDITY_FC2))
		Data.F_dp.SetFilterConstant(Data.Config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_DP, cfg.DEF_DEWPOINT_FC))
		Data.F2_dp.SetFilterConstant(Data.Config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_DP2, cfg.DEF_DEWPOINT_FC2))
		Data.F_pressure.SetFilterConstant(Data.Config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_P, cfg.DEF_PRESSURE_FC))
		Data.F_pressureNormal.SetFilterConstant(Data.Config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_P, cfg.DEF_PRESSURE_FC))
		Data.F2_pressure.SetFilterConstant(Data.Config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_P2, cfg.DEF_PRESSURE_FC2))
		Data.F2_pressureNormal.SetFilterConstant(Data.Config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_P2, cfg.DEF_PRESSURE_FC2))
		Data.F_temp2.SetFilterConstant(Data.Config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_T, cfg.DEF_TEMPERATURE_FC))
		Data.F2_temp2.SetFilterConstant(Data.Config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_T2, cfg.DEF_TEMPERATURE_FC2))
		Data.F_wetBulb.SetFilterConstant(Data.Config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_WB, cfg.DEF_WET_BULB_FC))
		Data.F2_wetBulb.SetFilterConstant(Data.Config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_WB2, cfg.DEF_WET_BULB_FC2))
	})
	if err != nil {
		return
	}

	// Set Pin to high / low - could be used to trigger LED or switch on LTC4311 (I2C terminator)
	pin := Data.Config.GetIntValue(cfg.SECTION_HARDWARE, cfg.KEY_HARDWARE_PIN_ON_START_GPIODEV_NUMBER, cfg.DEF_SET_PIN_ON_START)
	if pin > 0 {
		pinLine, err := gpio.NewGpioInPin(uint(Data.Config.GetIntValue(cfg.SECTION_HARDWARE, cfg.KEY_HARDWARE_PIN_ON_START_GPIODEV_NUMBER, cfg.DEF_PIN_ON_START_GPIODEV_NUMBER)),
			uint(pin), gpio.GpioPullDown)
		if err != nil {
			return
		}
		Data.PinOnStart = pinLine
		time.Sleep(250 * time.Millisecond)
		if Data.Config.GetBoolValue(cfg.SECTION_HARDWARE, cfg.KEY_HARDWARE_SET_PIN_ON_START_HIGH, cfg.DEF_SET_PIN_ON_START_HIGH) {
			Data.PinOnStart.Set(true)
			time.Sleep(250 * time.Millisecond)
		}
	}

	bus, _ := i2c.NewI2c(Data.Config.GetIntValue(cfg.SECTION_HARDWARE, cfg.KEY_HARDWARE_I2C_DEV_NUMBER, cfg.DEF_I2C_DEV_NUMBER))
	if bus == nil {
		return
	}

	// VEML7700
	Data.Veml7700 = sensors.NewVeml7700(bus, true, true)
	Data.Veml7700.Start(1000 * time.Millisecond)
	Data.F_lux, err = filter.NewAvgFilter(Data.Config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_L, cfg.DEF_LUX_FC), 0)
	if err != nil {
		return
	}
	Data.F2_lux, err = filter.NewAvgFilter(Data.Config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_L2, cfg.DEF_LUX_FC2), 0)
	if err != nil {
		return
	}

	Data.Sht45 = sensors.NewSht45(bus, Data.Config)
	Data.Sht45.Start(1000*time.Millisecond, sensors.Sht45MeasureMode_high)

	// SHT45
	Data.F_temp1, err = filter.NewAvgFilter(Data.Config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_T, cfg.DEF_TEMPERATURE_FC), 0)
	if err != nil {
		return
	}
	Data.F2_temp1, err = filter.NewAvgFilter(Data.Config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_T2, cfg.DEF_TEMPERATURE_FC2), 0)
	if err != nil {
		return
	}
	Data.F_humidity, err = filter.NewAvgFilter(Data.Config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_H, cfg.DEF_HUMIDITY_FC), 0)
	if err != nil {
		return
	}
	Data.F2_humidity, err = filter.NewAvgFilter(Data.Config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_H2, cfg.DEF_HUMIDITY_FC2), 0)
	if err != nil {
		return
	}
	Data.F_dp, err = filter.NewAvgFilter(Data.Config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_DP, cfg.DEF_DEWPOINT_FC), 0)
	if err != nil {
		return
	}
	Data.F2_dp, err = filter.NewAvgFilter(Data.Config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_DP2, cfg.DEF_DEWPOINT_FC2), 0)
	if err != nil {
		return
	}

	Data.Bmp581 = sensors.NewBmp581(sensors.BMP581_ADDR1, bus)
	Data.Bmp581.Start(1000 * time.Millisecond)

	// BMP581
	Data.F_pressure, err = filter.NewAvgFilter(Data.Config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_P, cfg.DEF_PRESSURE_FC), 0)
	if err != nil {
		return
	}
	Data.F_pressureNormal, err = filter.NewAvgFilter(Data.Config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_P, cfg.DEF_PRESSURE_FC), 0)
	if err != nil {
		return
	}
	Data.F2_pressure, err = filter.NewAvgFilter(Data.Config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_P2, cfg.DEF_PRESSURE_FC2), 0)
	if err != nil {
		return
	}
	Data.F2_pressureNormal, err = filter.NewAvgFilter(Data.Config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_P2, cfg.DEF_PRESSURE_FC2), 0)
	if err != nil {
		return
	}
	Data.F_temp2, err = filter.NewAvgFilter(Data.Config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_T, cfg.DEF_TEMPERATURE_FC), 0)
	if err != nil {
		return
	}
	Data.F2_temp2, err = filter.NewAvgFilter(Data.Config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_T2, cfg.DEF_TEMPERATURE_FC2), 0)
	if err != nil {
		return
	}
	Data.PressureTrendData.Values = make([]float64, 0, 10800/Data.Config.GetIntValue(cfg.SECTION_GENERAL, cfg.KEY_GENERAL_PRESSURE_TREND_INTERVAL, cfg.DEF_PRESSURE_TREND_INTERVAL))

	// Wet bulb
	Data.F_wetBulb, err = filter.NewAvgFilter(Data.Config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_WB, cfg.DEF_WET_BULB_FC), 0)
	if err != nil {
		return
	}
	Data.F2_wetBulb, err = filter.NewAvgFilter(Data.Config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_WB2, cfg.DEF_WET_BULB_FC2), 0)
	if err != nil {
		return
	}

	// SD
	Data.SdData, err = sd.NewSdData(Data.Config)
	if err != nil {
		return
	}

	Data.WebServer = webserver.NewWebServer(callBacks, Data.Config, Data.SdData)
	webError := make(chan error, 2)
	Data.WebServer.StartWebServer(webError)

	go doMeasure()
	go doSd()
	go doPressureTrend()

	x := <-webError
	fmt.Printf("WebError %v", x)

	fmt.Println("Exit")
}

func doMeasure() {
	for !raspi.RebbotRequested.Load() {
		time.Sleep(time.Duration(Data.Config.GetIntValue(cfg.SECTION_GENERAL, cfg.KEY_GENERAL_MEASURE_INTERVAL, cfg.DEF_MEASURE_INTERVAL)) * time.Millisecond)
		// VEML7700
		lux, _, _, isValidL := Data.Veml7700.GetData()
		if isValidL {
			Data.F_lux.Add(lux)
			Data.F2_lux.Add(lux)
			if TRACE_MESURE_DATA {
				log.Printf("Lux: %.2f lux", lux)
			}
		}
		// SHT45
		temperature, humidity, _, isValidH := Data.Sht45.GetData()
		if isValidH {
			Data.F_temp1.Add(temperature)
			Data.F2_temp1.Add(temperature)
			Data.F_humidity.Add(humidity)
			Data.F2_humidity.Add(humidity)
			dp := float32(helper.DewPoint(float64(temperature), float64(humidity)))
			Data.F_dp.Add(dp)
			Data.F2_dp.Add(dp)
			if TRACE_MESURE_DATA {
				log.Printf("Temperature: %.1f °C, Humidity: %.1f %%, DewPoint: %.1f °C", temperature, humidity, dp)
			}
		}
		// BMP581
		temperatureP, pressure, isValidP := Data.Bmp581.GetData()
		if isValidP {
			temp := temperatureP
			if isValidH {
				temp = temperature
			}
			pressureNormal := helper.PressureAtSeaLevel(float64(pressure),
				float64(Data.Config.GetIntValue(cfg.SECTION_GENERAL, cfg.KEY_GENERAL_HEIGHT, cfg.DEF_HEIGHT)), float64(temp), float64(humidity))
			Data.F_pressure.Add(pressure)
			Data.F2_pressure.Add(pressure)
			Data.F_pressureNormal.Add(float32(pressureNormal))
			Data.F2_pressureNormal.Add(float32(pressureNormal))
			Data.F_temp2.Add(temperatureP)
			Data.F2_temp2.Add(temperatureP)
			if TRACE_MESURE_DATA {
				log.Printf("TemperatureP: %.1f °C, Pressure: %.1f hPa, PressureNormal: %.1f hPa", temperatureP, pressure/100.0, pressureNormal/100.0)
			}
		}
		if isValidH && isValidP {
			wb, err := helper.GetRompsWetBulb(float64(temperature), float64(humidity), float64(pressure))
			if err == nil {
				Data.F_wetBulb.Add(float32(wb))
				Data.F2_wetBulb.Add(float32(wb))
			} else {
				log.Printf("Wet bulb error: %v", err)
			}
		}
	}
}

func doSd() {
	sdEntry := sd.SdDataEntry{}
	valid_l1 := false
	valid_l2 := false
	valid_t1 := false
	valid_t12 := false
	valid_h1 := false
	valid_h2 := false
	valid_dp1 := false
	valid_dp2 := false
	valid_p := false
	valid_pn := false
	valid_t2 := false
	valid_p2 := false
	valid_pn2 := false
	valid_t22 := false
	valid_wb := false
	valid_wb2 := false
	for !raspi.RebbotRequested.Load() {
		sdEntry.L_1, valid_l1 = Data.F_lux.GetValueExt()
		sdEntry.L_2, valid_l2 = Data.F2_lux.GetValueExt()
		sdEntry.T1_1, valid_t1 = Data.F_temp1.GetValueExt()
		sdEntry.T1_2, valid_t12 = Data.F2_temp1.GetValueExt()
		sdEntry.H_1, valid_h1 = Data.F_humidity.GetValueExt()
		sdEntry.H_2, valid_h2 = Data.F2_humidity.GetValueExt()
		sdEntry.Dp_1, valid_dp1 = Data.F_dp.GetValueExt()
		sdEntry.Dp_2, valid_dp2 = Data.F2_dp.GetValueExt()
		sdEntry.P_1, valid_p = Data.F_pressure.GetValueExt()
		sdEntry.P_2, valid_p2 = Data.F2_pressure.GetValueExt()
		sdEntry.Pn_1, valid_pn = Data.F_pressureNormal.GetValueExt()
		sdEntry.Pn_2, valid_pn2 = Data.F2_pressureNormal.GetValueExt()
		sdEntry.T2_1, valid_t2 = Data.F_temp2.GetValueExt()
		sdEntry.T2_2, valid_t22 = Data.F2_temp2.GetValueExt()
		sdEntry.WB_1, valid_wb = Data.F_wetBulb.GetValueExt()
		sdEntry.WB_2, valid_wb2 = Data.F2_wetBulb.GetValueExt()
		sdEntry.P_1 /= 100.0
		sdEntry.P_2 /= 100.0
		sdEntry.Pn_1 /= 100.0
		sdEntry.Pn_2 /= 100.0

		sdEntry.E_1, _ = Data.Sht45.GetError()
		sdEntry.E_2, _ = Data.Bmp581.GetError()
		sdEntry.E_3, _ = Data.Veml7700.GetError()

		if valid_l1 && valid_l2 && valid_t1 && valid_t12 && valid_h1 && valid_h2 && valid_dp1 && valid_dp2 &&
			valid_p && valid_pn && valid_t2 && valid_p2 && valid_pn2 && valid_t22 && valid_wb && valid_wb2 {
			Data.SdData.WriteEntry(&sdEntry)
		}
		time.Sleep(time.Duration(Data.Config.GetIntValue(cfg.SECTION_DATA, cfg.KEY_DATA_INTERVAL, cfg.DEF_DATA_INTERVAL)) * time.Second)
	}
}

func doPressureTrend() {
	for !raspi.RebbotRequested.Load() {
		pressure, isValid := Data.F_pressure.GetValueExt()
		interval := Data.Config.GetIntValue(cfg.SECTION_GENERAL, cfg.KEY_GENERAL_PRESSURE_TREND_INTERVAL, cfg.DEF_PRESSURE_TREND_INTERVAL)
		if isValid {
			if len(Data.PressureTrendData.Values) >= cap(Data.PressureTrendData.Values) {
				copy(Data.PressureTrendData.Values, Data.PressureTrendData.Values[1:])
				Data.PressureTrendData.Values = Data.PressureTrendData.Values[:len(Data.PressureTrendData.Values)-1]
			}
			Data.PressureTrendData.Values = append(Data.PressureTrendData.Values, float64(pressure))
			if len(Data.PressureTrendData.Values) >= 2 {
				// Caculate trend
				x := make([]float64, len(Data.PressureTrendData.Values))
				for i := range len(Data.PressureTrendData.Values) {
					x[i] = float64(i) * float64(interval)
				}
				a1, _, g := helper.CalculateRegression(x, Data.PressureTrendData.Values)
				Data.PressureTrendData.Trend3.Store(math.Float32bits(float32(a1 * 3600.0)))
				Data.PressureTrendData.Goodnness3.Store(math.Float32bits(float32(g)))
				l1 := 3600 / interval
				if len(x) > l1 {
					l1 = len(x) - l1
				} else {
					l1 = 0
				}
				a1, _, g = helper.CalculateRegression(x[l1:], Data.PressureTrendData.Values[l1:])
				Data.PressureTrendData.Trend1.Store(math.Float32bits(float32(a1 * 3600.0)))
				Data.PressureTrendData.Goodnness1.Store(math.Float32bits(float32(g)))
			}
		}
		time.Sleep(time.Duration(interval) * time.Second)
	}
}
