package data

import (
	"weatherd/cfg"
	"weatherd/filter"
	"weatherd/sd"
	"weatherd/sensors"
	"weatherd/webserver"
)

type Data struct {
	F_lux     *filter.AvgFilter
	F_lux2    *filter.AvgFilter
	Veml7700  *sensors.Veml7700
	Bmp581    *sensors.Bmp581
	WebServer *webserver.WebServer
	SdData    *sd.SdData
	Config    *cfg.Cfg
}
