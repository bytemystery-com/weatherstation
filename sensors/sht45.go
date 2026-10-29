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

// I2C implementation for SHT45 sensor

package sensors

import (
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/bytemystery-com/i2c"
	"weatherd/cfg"
	"weatherd/helper"
)

// https://sensirion.com/media/documents/33FD6951/6A7C10A0/HT_DS_Datasheet_SHT4x_V7.3.pdf

const (
	SHT44_ADDR = 0x44

	SHT45_REG_SERIAL         = 0x89
	SHT45_REG_RESET          = 0x94
	SHT45_REG_MEASURE_LOW    = 0xE0
	SHT45_REG_MEASURE_MEDIUM = 0xF6
	SHT45_REG_MEASURE_HIGH   = 0xFD

	SHT45_REG_HEATER_20_01  = 0x15
	SHT45_REG_HEATER_20_1   = 0x1E
	SHT45_REG_HEATER_110_01 = 0x24
	SHT45_REG_HEATER_110_1  = 0x2F
	SHT45_REG_HEATER_200_01 = 0x32
	SHT45_REG_HEATER_200_1  = 0x39

	SHT45_CMD_DELAY            = 1 // msec
	SHT45_MEASURE_DELAY_LOW    = 2 // msec
	SHT45_MEASURE_DELAY_MEDIUM = 5 // msec
	SHT45_MEASURE_DELAY_HIGH   = 9 // msec

	SHT45_HEATER_COOOL_DOWN  = 20   // sec
	SHT45_HEATER_TEMP_DELTA  = 1.5  // °C
	SHT45_HEATER_MIN_HUM     = 95.0 // %
	SHT45_HEATER_MIN_HUM_ICE = 90.0 // %
	SHT45_HEATER_TEMP_ICE    = 0.0  // °C
	SHT45_TEST_HEATER        = false
)

type sht45States int

const (
	sht45States_init sht45States = iota
	sht45States_cfg
	sht45States_measure
	sht45States_wait_heater
	sht45States_wait_heater_cooldown
)

type sht45HeaterPower int

const (
	sht45HeaterPower20 sht45HeaterPower = iota
	sht45HeaterPower110
	sht45HeaterPower200
)

type sht45HeaterTime int

const (
	sht45HeaterTime01 sht45HeaterTime = iota
	sht45HeaterTime1
)

type Sht45MeasureModes int

const (
	Sht45MeasureMode_low Sht45MeasureModes = iota
	Sht45MeasureMode_medium
	Sht45MeasureMode_high
)

type sht45ShmData struct {
	temperature atomic.Int32  // temperature*10
	humidity    atomic.Uint32 // %*10
	errors      atomic.Uint32
	isValid     atomic.Bool
}

type Sht45 struct {
	dev          *i2c.I2cDevice
	state        sht45States
	shm          sht45ShmData
	measureCycle time.Duration
	waitDuration time.Duration
	measureMode  Sht45MeasureModes
	config       *cfg.Cfg
}

func NewSht45(bus *i2c.I2c, config *cfg.Cfg) *Sht45 {
	dev, err := i2c.NewI2cDevice(SHT44_ADDR, bus)
	if err != nil {
		return nil
	}
	sht := Sht45{
		dev:    dev,
		state:  sht45States_init,
		config: config,
	}
	sht.shm.isValid.Store(false)
	return &sht
}

// Interface for web server
func (sht *Sht45) GetName() string {
	return "SHT45"
}

func (sht *Sht45) GetData() (float32, float32, uint32, bool) {
	return float32(sht.shm.temperature.Load()) / 10.0, float32(sht.shm.humidity.Load()) / 10.0, sht.shm.errors.Load(), sht.shm.isValid.Load()
}

func (sht *Sht45) GetError() (uint32, bool) {
	return sht.shm.errors.Load(), sht.shm.isValid.Load()
}

func (sht *Sht45) Start(measureCycle time.Duration, measureMode Sht45MeasureModes) {
	sht.measureCycle = measureCycle
	sht.measureMode = measureMode
	go sht.loop()
}

func crc8(data []byte) byte {
	crc := byte(0xFF)

	for _, b := range data {
		crc ^= b

		for i := 0; i < 8; i++ {
			if crc&0x80 != 0 {
				crc = (crc << 1) ^ 0x31
			} else {
				crc <<= 1
			}
		}
	}

	return crc
}

func getHeaterRegister(power sht45HeaterPower, time sht45HeaterTime) (byte, error) {
	switch power {
	case sht45HeaterPower20:
		switch time {
		case sht45HeaterTime01:
			return SHT45_REG_HEATER_20_01, nil
		case sht45HeaterTime1:
			return SHT45_REG_HEATER_20_1, nil
		default:
			return 0, fmt.Errorf("Invalid heater time")
		}
	case sht45HeaterPower110:
		switch time {
		case sht45HeaterTime01:
			return SHT45_REG_HEATER_110_01, nil
		case sht45HeaterTime1:
			return SHT45_REG_HEATER_110_1, nil
		default:
			return 0, fmt.Errorf("Invalid heater time")
		}
	case sht45HeaterPower200:
		switch time {
		case sht45HeaterTime01:
			return SHT45_REG_HEATER_200_01, nil
		case sht45HeaterTime1:
			return SHT45_REG_HEATER_200_1, nil
		default:
			return 0, fmt.Errorf("Invalid heater time")
		}
	default:
		return 0, fmt.Errorf("Invalid heater power")
	}
}

func getHeaterDuration(time sht45HeaterTime) (int, error) {
	switch time {
	case sht45HeaterTime01:
		return 100, nil
	case sht45HeaterTime1:
		return 1000, nil
	default:
		return 0, fmt.Errorf("Invalid heater time")
	}
}

func (sht *Sht45) getSerial() (uint16, error) {
	data := make([]i2c.I2cData, 1)
	data[0].Mode = i2c.I2c_write
	data[0].Data = []byte{SHT45_REG_SERIAL}
	err := sht.dev.Transfer(data)
	if err != nil {
		return 0, err
	}
	time.Sleep(SHT45_CMD_DELAY * time.Millisecond)

	data[0].Mode = i2c.I2c_read
	data[0].Data = make([]byte, 3)
	err = sht.dev.Transfer(data)
	if err != nil {
		return 0, err
	}

	if data[0].Data[2] == crc8(data[0].Data[:2]) {
		return uint16(data[0].Data[0])<<8 | uint16(data[0].Data[1]), nil
	} else {
		return 0, fmt.Errorf("CRC error")
	}
}

func (sht *Sht45) init() error {
	data := make([]i2c.I2cData, 1)
	data[0].Mode = i2c.I2c_write
	data[0].Data = []byte{SHT45_REG_RESET}
	err := sht.dev.Transfer(data)
	if err != nil {
		return err
	}
	time.Sleep(SHT45_CMD_DELAY * time.Millisecond)

	serial, err := sht.getSerial()
	if err != nil {
		return err
	}
	log.Printf("SHT45: Serial: %04x", serial)
	return nil
}

func (sht *Sht45) configure() error {
	return nil
}

func (sht *Sht45) measureGetResults() (float32, float32, error) {
	data := make([]i2c.I2cData, 1)
	data[0].Mode = i2c.I2c_read
	data[0].Data = make([]byte, 6)
	err := sht.dev.Transfer(data)
	if err != nil {
		return 0, 0, err
	}
	if data[0].Data[2] != crc8(data[0].Data[:2]) || data[0].Data[5] != crc8(data[0].Data[3:5]) {
		return 0, 0, fmt.Errorf("CRC error")
	}

	temp_raw := uint16(data[0].Data[0])<<8 | uint16(data[0].Data[1])
	hum_raw := uint16(data[0].Data[3])<<8 | uint16(data[0].Data[4])
	temp := -45.0 + 175*float64(temp_raw)/float64(0xFFFF)
	hum := -6.0 + 125*float64(hum_raw)/float64(0xFFFF)
	if hum < 0 {
		hum = 0
	}
	if hum > 100 {
		hum = 100
	}
	return float32(temp), float32(hum), nil
}

// returns temperature, humidity, error
func (sht *Sht45) measure() (float32, float32, error) {
	var reg byte = 0
	var delay byte = 0

	switch sht.measureMode {
	case Sht45MeasureMode_low:
		reg = SHT45_REG_MEASURE_LOW
		delay = SHT45_MEASURE_DELAY_LOW
	case Sht45MeasureMode_medium:
		reg = SHT45_REG_MEASURE_MEDIUM
		delay = SHT45_MEASURE_DELAY_MEDIUM
	case Sht45MeasureMode_high:
		reg = SHT45_REG_MEASURE_HIGH
		delay = SHT45_MEASURE_DELAY_HIGH
	default:
		return 0, 0, fmt.Errorf("Invalid measure mode")
	}
	data := make([]i2c.I2cData, 1)
	data[0].Mode = i2c.I2c_write
	data[0].Data = []byte{reg}
	err := sht.dev.Transfer(data)
	if err != nil {
		return 0, 0, err
	}
	time.Sleep(time.Duration(delay) * time.Millisecond)

	temp, hum, err := sht.measureGetResults()
	if err != nil {
		return 0, 0, err
	}
	return float32(temp), float32(hum), nil
}

func (sht *Sht45) handleHeater(temperature, humidity float64) time.Duration {
	now := time.Now()
	lastHeater := time.Unix(sht.config.GetInt64Value(cfg.SECTION_RUN, cfg.KEY_RUN_HEATER, 0), 0)
	if lastHeater.IsZero() || now.Sub(lastHeater) >= time.Duration(sht.config.GetIntValue(cfg.SECTION_GENERAL, cfg.KEY_GENERAL_HEATER_INTERVAL, cfg.DEF_HEATER_INTERVAL))*time.Minute {
		needHeater := false
		var heaterPower sht45HeaterPower
		var heaterTime sht45HeaterTime
		if SHT45_TEST_HEATER {
			needHeater = true
			heaterPower = sht45HeaterPower200
			heaterTime = sht45HeaterTime1
		} else {
			if now.Sub(lastHeater) >= time.Duration(sht.config.GetIntValue(cfg.SECTION_GENERAL, cfg.KEY_GENERAL_HEATER_MAINT_INTERVAL, cfg.DEF_HEATER_MAINT_INTERVAL))*time.Hour {
				// HeaterOn
				needHeater = true
				heaterTime = sht45HeaterTime1
				heaterPower = sht45HeaterPower110
			} else {
				dewPoint := helper.DewPoint(temperature, humidity)
				delta := temperature - dewPoint
				if delta <= SHT45_HEATER_TEMP_DELTA {
					if humidity >= SHT45_HEATER_MIN_HUM {
						// HeaterOn
						needHeater = true
						heaterTime = sht45HeaterTime1
						if delta > 0 {
							heaterPower = sht45HeaterPower110
						} else {
							heaterPower = sht45HeaterPower200
						}
					}
				}
				if temperature <= SHT45_HEATER_TEMP_ICE {
					if humidity >= SHT45_HEATER_MIN_HUM_ICE {
						// HeaterOn
						needHeater = true
						heaterTime = sht45HeaterTime1
						heaterPower = sht45HeaterPower200
					}
				}
			}
		}
		if needHeater {
			data := make([]i2c.I2cData, 1)
			data[0].Mode = i2c.I2c_write
			reg, err := getHeaterRegister(heaterPower, heaterTime)
			if err != nil {
				return 0
			}
			data[0].Data = []byte{reg}
			err = sht.dev.Transfer(data)
			if err != nil {
				sht.shm.errors.Add(1)
				return 0
			}
			heatDuration, err := getHeaterDuration(heaterTime)
			if err != nil {
				return 0
			}
			sht.state = sht45States_wait_heater
			sht.config.SetInt64Value(cfg.SECTION_RUN, cfg.KEY_RUN_HEATER, now.Unix(), "")
			sht.config.Save()
			log.Printf("SHT45: Heater was switched on with (%d, %d)", heaterPower, heaterTime)
			return time.Duration(SHT45_MEASURE_DELAY_HIGH+heatDuration) * time.Millisecond
		}
	}
	return 0
}

func (sht *Sht45) loop() {
	var start time.Time
	var duration time.Duration
	defaultSleep := time.Duration(250 * time.Millisecond) // > 100 msec
	sleep := defaultSleep
	for {
		time.Sleep(sleep)
		switch sht.state {
		case sht45States_init:
			err := sht.init()
			if err != nil {
				sht.shm.errors.Add(1)
				sht.shm.isValid.Store(false)
				log.Printf("SHT45: init failed with %v", err)
			} else {
				sht.state = sht45States_cfg
			}
			sleep = defaultSleep

		case sht45States_cfg:
			err := sht.configure()
			if err != nil {
				sht.state = sht45States_init
				sht.shm.errors.Add(1)
				sht.shm.isValid.Store(false)
				log.Printf("SHT45: configure failed with %v", err)
			} else {
				sht.state = sht45States_measure
			}
			sleep = defaultSleep

		case sht45States_measure:
			start = time.Now()
			temperature, humidity, err := sht.measure()
			if err != nil {
				sht.shm.errors.Add(1)
				sht.shm.isValid.Store(false)
				sht.state = sht45States_init
				log.Printf("SHT45: measure failed with %v", err)
			} else {
				sht.shm.temperature.Store(int32(temperature * 10))
				sht.shm.humidity.Store(uint32(humidity * 10))
				sht.shm.isValid.Store(true)
			}
			duration = time.Since(start)
			if duration < sht.measureCycle {
				sleep = sht.measureCycle - duration
			} else {
				log.Printf("SHT45: measure tooks too long %d nsec", duration)
				sleep = sht.measureCycle
			}
			// Heater
			if sht.state == sht45States_measure {
				delay := sht.handleHeater(float64(temperature), float64(humidity))
				if delay > 0 {
					sleep = delay
				}
			}

		case sht45States_wait_heater:
			temp, hum, err := sht.measureGetResults()
			if err == nil {
				log.Printf("SHT45: measure after heater - temp: %.1f, hum: %.1f", temp, hum)
				sht.state = sht45States_wait_heater_cooldown
			} else {
				sht.shm.errors.Add(1)
				sht.state = sht45States_init
			}
			sleep = time.Duration(SHT45_HEATER_COOOL_DOWN) * time.Second
			sht.shm.isValid.Store(false)

		case sht45States_wait_heater_cooldown:
			sht.state = sht45States_measure
			sleep = sht.measureCycle
		}
	}
}
