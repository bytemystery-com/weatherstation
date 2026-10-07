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

package data

import (
	"sync/atomic"

	"weatherd/cfg"
	"weatherd/filter"
	"weatherd/sd"
	"weatherd/sensors"
	"weatherd/webserver"
)

type PressureTrendData struct {
	Values     []float64
	Trend1     atomic.Uint32
	Goodnness1 atomic.Uint32
	Trend3     atomic.Uint32
	Goodnness3 atomic.Uint32
}

type Data struct {
	F_lux             *filter.AvgFilter
	F2_lux            *filter.AvgFilter
	F_temp1           *filter.AvgFilter
	F2_temp1          *filter.AvgFilter
	F_humidity        *filter.AvgFilter
	F2_humidity       *filter.AvgFilter
	F_dp              *filter.AvgFilter
	F2_dp             *filter.AvgFilter
	F_pressure        *filter.AvgFilter
	F2_pressure       *filter.AvgFilter
	F_pressureNormal  *filter.AvgFilter
	F2_pressureNormal *filter.AvgFilter
	F_temp2           *filter.AvgFilter
	F2_temp2          *filter.AvgFilter
	Veml7700          *sensors.Veml7700
	Sht45             *sensors.Sht45
	Bmp581            *sensors.Bmp581
	WebServer         *webserver.WebServer
	SdData            *sd.SdData
	Config            *cfg.Cfg
	PressureTrendData PressureTrendData
}
