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
	"fmt"
	"log"
	"os"
	"time"

	"weatherd/cfg"
	"weatherd/data"
	"weatherd/filter"
	"weatherd/helper"
	"weatherd/i2c"
	"weatherd/raspi"
	"weatherd/sd"
	"weatherd/sensors"
	"weatherd/webserver"
)

var Data data.Data

type WebCallbacksImpl struct{}

var _ webserver.WebCallBacks = (*WebCallbacksImpl)(nil)

func (w WebCallbacksImpl) GetData() *webserver.WebDataAll {
	luxErrCount, luxIsValid := Data.Veml7700.GetError()
	humErrCount, humIsValid := Data.Sht45.GetError()
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
				Errors:     0,
				Valid:      true,
			},
			Pressure:       0,
			PressureNormal: 0,
		},

		Humidity: webserver.WebDataHumidity{
			WebData: webserver.WebData{
				SensorName: "",
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
	}
	return &data
}

var callBacks WebCallbacksImpl

func main() {
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
	})
	if err != nil {
		return
	}

	bus, _ := i2c.NewI2c(1)
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

	// SD
	Data.SdData, err = sd.NewSdData(Data.Config)
	if err != nil {
		return
	}

	Data.WebServer = webserver.NewWebServer(8080, callBacks, Data.Config)
	webError := make(chan error, 1)
	Data.WebServer.StartWebServer(webError)

	go doMeasure()
	go doSd()

	<-webError

	fmt.Println("Exit")
}

func doMeasure() {
	for !raspi.RebbotRequested.Load() {
		time.Sleep(time.Duration(Data.Config.GetIntValue(cfg.SECTION_GENERAL, cfg.KEY_GENERAL_MEASURE_INTERVAL, cfg.DEF_MEASURE_INTERVAL)) * time.Millisecond)
		// VEML7700
		lux, _, _, isValid := Data.Veml7700.GetData()
		if isValid {
			Data.F_lux.Add(lux)
			Data.F2_lux.Add(lux)
			// log.Printf("Lux: %.2f, Lux2: %.2f", Data.F_lux.GetValue(), Data.F2_lux.GetValue())
		}
		// SHT45
		temperature, humidity, _, isValid := Data.Sht45.GetData()
		if isValid {
			Data.F_temp1.Add(temperature)
			Data.F2_temp1.Add(temperature)
			Data.F_humidity.Add(humidity)
			Data.F2_humidity.Add(humidity)
			dp := float32(helper.DewPoint(float64(temperature), float64(humidity)))
			Data.F_dp.Add(dp)
			Data.F2_dp.Add(dp)
			log.Printf("Temperature: %.1f, Humidity: %.1f, DewPoint: %.1f", Data.F_temp1.GetValue(), Data.F_humidity.GetValue(), Data.F_dp.GetValue())
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
	for !raspi.RebbotRequested.Load() {
		sdEntry.L_1, valid_l1 = Data.F_lux.GetValueExt()
		sdEntry.L_2, valid_l2 = Data.F2_lux.GetValueExt()
		sdEntry.T1_1, valid_t1 = Data.F_temp1.GetValueExt()
		sdEntry.T1_2, valid_t12 = Data.F2_temp1.GetValueExt()
		sdEntry.H_1, valid_h1 = Data.F_humidity.GetValueExt()
		sdEntry.H_2, valid_h2 = Data.F2_humidity.GetValueExt()
		sdEntry.Dp_1, valid_dp1 = Data.F_dp.GetValueExt()
		sdEntry.Dp_2, valid_dp2 = Data.F2_dp.GetValueExt()
		// sdEntry.E_2, _ = Data.Bmp581.GetError()
		sdEntry.E_1, _ = Data.Sht45.GetError()
		sdEntry.E_3, _ = Data.Veml7700.GetError()
		if valid_l1 && valid_l2 && valid_t1 && valid_t12 && valid_h1 && valid_h2 && valid_dp1 && valid_dp2 {
			Data.SdData.WriteEntry(&sdEntry)
		}
		time.Sleep(time.Duration(Data.Config.GetIntValue(cfg.SECTION_DATA, cfg.KEY_DATA_INTERVAL, cfg.DEF_DATA_INTERVAL)) * time.Second)
	}
}
