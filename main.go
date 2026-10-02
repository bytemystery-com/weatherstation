package main

import (
	"fmt"
	"log"
	"time"

	"weatherd/cfg"
	"weatherd/i2c"
	"weatherd/sensors"
	"weatherd/shm"
)

var Data shm.ShmData

func main() {
	cfg, err := cfg.NewCfg("weatherd.cfg")
	if err != nil {
		return
	}
	cfg.SetDeaultCfg()
	// cfg.SetFloatValue("filterdata", "filter_constant_t", DEF_TEMPERATURE_FC, "Temperatur filter")

	err = cfg.Save()
	if err != nil {
		log.Printf("%s", err)
	}

	bus, _ := i2c.NewI2c(1)
	if bus == nil {
		return
	}

	veml7700 := sensors.NewVeml7700(sensors.VEML7700_ADDR, bus)
	veml7700.Start(1000 * time.Millisecond)

	webError := make(chan error, 1)
	startWebServer(8080, webError, cfg)

	<-webError

	fmt.Println("Exit")
}
