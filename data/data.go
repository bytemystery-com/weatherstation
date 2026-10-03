package data

import (
	"weatherd/filter"
	"weatherd/sensors"
	"weatherd/webserver"
)

type Data struct {
	F_lux     *filter.AvgFilter
	F_lux2    *filter.AvgFilter
	Veml7700  *sensors.Veml7700
	Bmp581    *sensors.Bmp581
	WebServer *webserver.WebServer
}
