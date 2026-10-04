package main

import (
	"fmt"
	"log"
	"time"

	"weatherd/cfg"
	"weatherd/data"
	"weatherd/filter"
	"weatherd/i2c"
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
	var config *cfg.Cfg
	config, err := cfg.NewCfg("weatherd.cfg", func() {
		Data.F_lux.SetFilterConstant(config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_L, cfg.DEF_LUX_FC))
		Data.F_lux2.SetFilterConstant(config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_L2, cfg.DEF_LUX_FC2))
	})
	if err != nil {
		return
	}
	config.SetDeaultCfg()
	// cfg.SetFloatValue("filterdata", "filter_constant_t", DEF_TEMPERATURE_FC, "Temperatur filter")

	err = config.Save()
	if err != nil {
		log.Printf("%s", err)
	}

	bus, _ := i2c.NewI2c(1)
	if bus == nil {
		return
	}

	Data.Veml7700 = sensors.NewVeml7700(bus, true, true)
	Data.Veml7700.Start(1000 * time.Millisecond)

	Data.F_lux, err = filter.NewAvgFilter(config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_L, cfg.DEF_LUX_FC), 0)
	Data.F_lux2, err = filter.NewAvgFilter(config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_L2, cfg.DEF_LUX_FC2), 0)
	log.Printf("l2: %f", config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_L2, cfg.DEF_LUX_FC2))

	Data.WebServer = webserver.NewWebServer(8080, callBacks, config)
	webError := make(chan error, 1)
	Data.WebServer.StartWebServer(webError)

	time.Sleep(1000 * time.Millisecond)
	for {
		time.Sleep(1000 * time.Millisecond)
		lux, _, _, isValid := Data.Veml7700.GetData()
		if isValid {
			Data.F_lux.Add(lux)
			Data.F_lux2.Add(lux)
		}

		log.Printf("Lux: %.2f, Lux2: %.2f", Data.F_lux.GetValue(), Data.F_lux2.GetValue())
	}

	<-webError

	fmt.Println("Exit")
}
