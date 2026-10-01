package main

import (
	"fmt"
	"log"

	"weatherd/cfg"
	"weatherd/i2c"
	"weatherd/shm"
)

var Data shm.ShmData

func main() {
	cfg, err := cfg.NewCfg("weatherd.cfg")
	if err != nil {
		return
	}
	cfg.SetDeaultCfg()
	cfg.Save()
	// cfg.SetFloatValue("filterdata", "filter_constant_t", DEF_TEMPERATURE_FC, "Temperatur filter")

	err = cfg.Save()
	if err != nil {
		log.Printf("%s", err)
	}

	webError := make(chan error, 1)
	startWebServer(8080, webError)

	<-webError

	bus, _ := i2c.NewI2c(1)
	if bus == nil {
		return
	}

	fmt.Println("Exit")
}
