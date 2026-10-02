package main

import (
	"fmt"
	"log"
	"time"

	"weatherd/cfg"
	"weatherd/filter"
	"weatherd/i2c"
	"weatherd/sensors"
	"weatherd/shm"
)

var Data shm.ShmData

func main() {
	config, err := cfg.NewCfg("weatherd.cfg")
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

	veml7700 := sensors.NewVeml7700(bus, true)
	veml7700.Start(1000 * time.Millisecond)

	webError := make(chan error, 1)
	startWebServer(8080, webError, config)

	f_lux, err := filter.NewAvgFilter(config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_L, cfg.DEF_LUX_FC), 0)
	f_lux2, err := filter.NewAvgFilter(config.GetFloatValue(cfg.SECTION_FILTERDATA, cfg.KEY_FILTER_CONST_L2, cfg.DEF_LUX_FC2), 0)
	for {
		time.Sleep(1000 * time.Millisecond)
		lux, _ := veml7700.GetData()
		f_lux.Add(lux)
		f_lux2.Add(lux)
		log.Printf("Lux: %.2f, Lux2: %.2f", f_lux.GetValue(), f_lux2.GetValue())
	}

	<-webError

	fmt.Println("Exit")
}
