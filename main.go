package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"weatherd/cfg"
	"weatherd/data"
	"weatherd/filter"
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
	data := webserver.WebDataAll{
		Lux: webserver.WebDataLux{
			WebData: webserver.WebData{
				SensorName: Data.Veml7700.GetName(),
				Errors:     uint(luxErrCount),
				Valid:      luxIsValid,
			},
			Lux:  Data.F_lux.GetValue(),
			Lux2: Data.F_lux2.GetValue(),
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
				Errors:     0,
				Valid:      true,
			},
			Humidity:  0,
			Humidity2: 0,
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
		Data.F_lux2.SetFilterConstant(Data.Config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_L2, cfg.DEF_LUX_FC2))
	})
	if err != nil {
		return
	}

	bus, _ := i2c.NewI2c(1)
	if bus == nil {
		return
	}

	Data.Veml7700 = sensors.NewVeml7700(bus, true, true)
	Data.Veml7700.Start(1000 * time.Millisecond)

	Data.F_lux, err = filter.NewAvgFilter(Data.Config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_L, cfg.DEF_LUX_FC), 0)
	if err != nil {
		return
	}
	Data.F_lux2, err = filter.NewAvgFilter(Data.Config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_L2, cfg.DEF_LUX_FC2), 0)
	if err != nil {
		return
	}
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
		lux, _, _, isValid := Data.Veml7700.GetData()
		if isValid {
			Data.F_lux.Add(lux)
			Data.F_lux2.Add(lux)
			log.Printf("Lux: %.2f, Lux2: %.2f", Data.F_lux.GetValue(), Data.F_lux2.GetValue())
		}
	}
}

func doSd() {
	sdEntry := sd.SdDataEntry{}
	valid_l1 := false
	valid_l2 := false
	for !raspi.RebbotRequested.Load() {
		sdEntry.L_1, valid_l1 = Data.F_lux.GetValueExt()
		sdEntry.L_2, valid_l2 = Data.F_lux2.GetValueExt()
		// sdEntry.E_2, _ = Data.Bmp581.GetError()
		sdEntry.E_3, _ = Data.Veml7700.GetError()
		if valid_l1 && valid_l2 {
			Data.SdData.WriteEntry(&sdEntry)
		}
		time.Sleep(time.Duration(Data.Config.GetIntValue(cfg.SECTION_DATA, cfg.KEY_DATA_INTERVAL, cfg.DEF_DATA_INTERVAL)) * time.Second)
	}
}
