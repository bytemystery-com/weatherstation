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

// I2C implementation for VEML7700 sensor

package sensors

import (
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"weatherd/i2c"
)

const (
	VEML7700_ADDR = 0x10

	VEML7700_REG_CHIPID     = 0x07
	VEML7700_REG_CFG        = 0x00
	VEML7700_REG_PSM        = 0x03
	VEML7700_REG_LUX_DATA   = 0x04
	VEML7700_REG_WHITE_DATA = 0x05

	VEML7700_CHIPID  = 0x81
	VEML7700_SUBID_1 = 0xC4
	VEML7700_SUBID_2 = 0xD4

	VEML7700_GAIN_1    = (0x00 << 11)
	VEML7700_GAIN_2    = (0x01 << 11)
	VEML7700_GAIN_08   = (0x02 << 11)
	VEML7700_GAIN_04   = (0x03 << 11)
	VEML7700_GAIN_MASK = (0x03 << 11)

	VEML7700_INTEG_TIME_25   = (0x0C << 6)
	VEML7700_INTEG_TIME_50   = (0x08 << 6)
	VEML7700_INTEG_TIME_100  = (0x00 << 6)
	VEML7700_INTEG_TIME_200  = (0x01 << 6)
	VEML7700_INTEG_TIME_400  = (0x02 << 6)
	VEML7700_INTEG_TIME_800  = (0x03 << 6)
	VEML7700_INTEG_TIME_MASK = (0x0F << 6)

	VEML7700_DEFAULT_CFG = (VEML7700_GAIN_08 | VEML7700_INTEG_TIME_100)

	VEML7700_SHUTDOWN_ON   = (0x01)
	VEML7700_SHUTDOWN_OFF  = (0x00)
	VEML7700_SHUTDOWN_MASK = (0x01)

	VEML7700_POWERSAVING_MODE_1 = (0x00 << 1)
	VEML7700_POWERSAVING_MODE_2 = (0x01 << 1)
	VEML7700_POWERSAVING_MODE_3 = (0x02 << 1)
	VEML7700_POWERSAVING_MODE_4 = (0x03 << 1)

	VEML7700_POWERSAVING_ON  = (0x01)
	VEML7700_POWERSAVING_OFF = (0x00)
)

type veml7700States int

const (
	veml7700States_init = iota
	veml7700States_cfg
	veml7700States_measure
)

type veml7700ShmData struct {
	lux           atomic.Uint32 // lux*100
	white         atomic.Uint32 // white*100
	overrun       atomic.Bool   // ausserhalb spec
	whiteAlsRatio atomic.Uint32 // ratio*1000
}

type Veml7700 struct {
	dev          *i2c.I2cDevice
	state        veml7700States
	shm          veml7700ShmData
	measureCycle time.Duration
	waitDuration time.Duration
	doAdjust     bool
}

func NewVeml7700(bus *i2c.I2c, doAdjust bool) *Veml7700 {
	dev, err := i2c.NewI2cDevice(VEML7700_ADDR, bus)
	if err != nil {
		return nil
	}
	return &Veml7700{
		dev:      dev,
		state:    veml7700States_init,
		doAdjust: doAdjust,
	}
}

// Interface for web server
func (veml *Veml7700) GetData() (float32, float32) {
	return float32(veml.shm.lux.Load() / 100), float32(veml.shm.lux.Load() / 100)
}

func (veml *Veml7700) Start(measureCycle time.Duration) {
	veml.measureCycle = measureCycle
	go veml.loop()
}

func (veml *Veml7700) isValidChipId(chipId uint16) bool {
	var subId uint8 = uint8((chipId >> 8) & 0xFF)
	return (chipId&0xFF) == VEML7700_CHIPID && (subId == VEML7700_SUBID_1 || subId == VEML7700_SUBID_2)
}

func (veml *Veml7700) getChipId() (uint16, error) {
	data := make([]i2c.I2cData, 2)
	data[0].Mode = i2c.I2c_write
	data[0].Data = []byte{VEML7700_REG_CHIPID}
	data[1].Mode = i2c.I2c_read
	data[1].Data = make([]byte, 2)

	err := veml.dev.Transfer(data)
	if err != nil {
		return 0, err
	}
	return uint16(data[1].Data[1])<<8 | uint16(data[1].Data[0]), nil
}

func (veml *Veml7700) init() error {
	id, err := veml.getChipId()
	if err != nil {
		return err
	}
	log.Printf("ChipID: %04x", id)
	if !veml.isValidChipId(id) {
		return fmt.Errorf("Invalid chip id %d", id)
	}
	return nil
}

func (veml *Veml7700) setStandby(on bool) error {
	data := make([]i2c.I2cData, 2)
	data[0].Mode = i2c.I2c_write
	data[0].Data = []byte{VEML7700_REG_CFG}
	data[1].Mode = i2c.I2c_read
	data[1].Data = make([]byte, 2)
	err := veml.dev.Transfer(data)
	if err != nil {
		return err
	}

	cfg := uint16(data[1].Data[0]) | ((uint16(data[1].Data[1])) << 8)
	cfg &= ^uint16(VEML7700_SHUTDOWN_MASK)
	if on {
		cfg |= VEML7700_SHUTDOWN_ON
	} else {
		cfg |= VEML7700_SHUTDOWN_OFF
	}

	data = make([]i2c.I2cData, 1)
	data[0].Mode = i2c.I2c_write
	data[0].Data = []byte{VEML7700_REG_CFG, byte(cfg & 0xFF), byte((cfg >> 8) & 0xFF)}
	err = veml.dev.Transfer(data)
	if err != nil {
		return err
	}
	return nil
}

func (veml *Veml7700) configure(GainIntCfg uint16) error {
	err := veml.setStandby(true)
	if err != nil {
		return err
	}

	data := make([]i2c.I2cData, 1)
	data[0].Mode = i2c.I2c_write
	var cfg uint16 = GainIntCfg | VEML7700_SHUTDOWN_ON
	data[0].Data = []byte{VEML7700_REG_CFG, byte(cfg & 0xFF), byte((cfg >> 8) & 0xFF)}
	err = veml.dev.Transfer(data)
	if err != nil {
		return err
	}

	// Does not work in one call .....

	var psm uint16 = VEML7700_POWERSAVING_MODE_1 | VEML7700_POWERSAVING_OFF
	data[0].Data = []byte{VEML7700_REG_PSM, byte(psm & 0xFF), byte((psm >> 8) & 0xFF)}
	err = veml.dev.Transfer(data)
	if err != nil {
		return err
	}

	err = veml.setStandby(false)
	if err != nil {
		return err
	}
	return nil
}

// returns abstract values as noted in the application note
// Gain selection G:
//     2 → 4
//     1 → 3
// (1/4) → 2
// (1/8) → 1
//
// LS integration time setting IT:
// 25 ms → -2
// 50 ms → -1
// 100 ms → 0
// 200 ms → 1
// 400 ms → 2
// 800 ms → 3

func (veml *Veml7700) getCfgAsAbstract(cfg uint16) (int, int, error) {
	intTime := (cfg & VEML7700_INTEG_TIME_MASK)
	gain := (cfg & VEML7700_GAIN_MASK)
	var i int = 0
	var g int = 0
	switch intTime {
	case VEML7700_INTEG_TIME_800:
		i = 3
	case VEML7700_INTEG_TIME_400:
		i = 2
	case VEML7700_INTEG_TIME_200:
		i = 1
	case VEML7700_INTEG_TIME_100:
		i = 0
	case VEML7700_INTEG_TIME_50:
		i = -1
	case VEML7700_INTEG_TIME_25:
		i = -2
	}

	switch gain {
	case VEML7700_GAIN_2:
		g = 4
	case VEML7700_GAIN_1:
		g = 3
	case VEML7700_GAIN_04:
		g = 2
	case VEML7700_GAIN_08:
		g = 1
	}
	return g, i, nil
}

func (veml *Veml7700) getCfgValueFromAbstract(G int, I int) (uint16, uint16, error) {
	var i uint16 = 0
	var g uint16 = 0

	switch I {
	case 3:
		i = VEML7700_INTEG_TIME_800
	case 2:
		i = VEML7700_INTEG_TIME_400
	case 1:
		i = VEML7700_INTEG_TIME_200
	case 0:
		i = VEML7700_INTEG_TIME_100
	case -1:
		i = VEML7700_INTEG_TIME_50
	case -2:
		i = VEML7700_INTEG_TIME_25
	default:
		return 0, 0, fmt.Errorf("Integration time out of range %d", I)
	}
	switch G {
	case 4:
		g = VEML7700_GAIN_2
	case 3:
		g = VEML7700_GAIN_1
	case 2:
		g = VEML7700_GAIN_04
	case 1:
		g = VEML7700_GAIN_08
	default:
		return 0, 0, fmt.Errorf("Gain time out of range %d", I)
	}
	return g, i, nil
}

func (veml *Veml7700) adjust(raw, cfg uint16) (bool, error) {
	G, I, err := veml.getCfgAsAbstract(cfg)
	GOld := G
	IOld := I
	if err != nil {
		return false, err
	}
	if raw < 100 {
		if G < 4 {
			G++
		} else if I < 3 {
			I++
		}
	} else {
		if raw > 10000 {
			if G > 1 {
				G--
			} else if I > -2 {
				I--
			}
		}
	}
	if I != IOld || G != GOld {
		g, i, err := veml.getCfgValueFromAbstract(G, I)
		if err != nil {
			return false, err
		}
		log.Printf("VEML7700, adjust from G: %d, I: %d to G: %d, I: %d", GOld, IOld, G, I)
		err = veml.configure(g | i)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

// returns lux, white, luxRaw, whiteRaw, overrun, cfg, error
func (veml *Veml7700) measure() (float32, float32, uint16, uint16, bool, uint16, error) {
	data := make([]i2c.I2cData, 2)
	data[0].Mode = i2c.I2c_write
	data[0].Data = []byte{VEML7700_REG_LUX_DATA}
	data[1].Mode = i2c.I2c_read
	data[1].Data = make([]byte, 4)
	err := veml.dev.Transfer(data)
	if err != nil {
		return 0, 0, 0, 0, false, 0, err
	}
	var luxRaw uint16 = uint16(data[1].Data[0]) | (uint16(data[1].Data[1]) << 8)
	var whiteRaw uint16 = uint16(data[1].Data[2]) | (uint16(data[1].Data[3]) << 8)

	data[0].Data = []byte{VEML7700_REG_CFG}
	data[1].Data = make([]byte, 2)
	err = veml.dev.Transfer(data)
	if err != nil {
		return 0, 0, luxRaw, whiteRaw, false, 0, err
	}
	cfg := uint16(data[1].Data[0]) | ((uint16(data[1].Data[1])) << 8)
	intTime := (cfg & VEML7700_INTEG_TIME_MASK)
	gain := (cfg & VEML7700_GAIN_MASK)
	var gainFactor float64 = 1.0
	var intTimeFactor float64 = 1.0

	switch gain {
	case VEML7700_GAIN_2:
		gainFactor = 16.0
	case VEML7700_GAIN_1:
		gainFactor = 8.0
	case VEML7700_GAIN_04:
		gainFactor = 2.0
	case VEML7700_GAIN_08:
		gainFactor = 1.0

	}

	switch intTime {
	case VEML7700_INTEG_TIME_800:
		intTimeFactor = 32.0
	case VEML7700_INTEG_TIME_400:
		intTimeFactor = 16.0
	case VEML7700_INTEG_TIME_200:
		intTimeFactor = 8.0
	case VEML7700_INTEG_TIME_100:
		intTimeFactor = 4.0
	case VEML7700_INTEG_TIME_50:
		intTimeFactor = 2.0
	case VEML7700_INTEG_TIME_25:
		intTimeFactor = 1.0
	}

	resolution := 2.1504 / (intTimeFactor * gainFactor)
	maxLux := 140926.0 / (intTimeFactor * gainFactor)
	lux := float64(luxRaw) * resolution
	white := float64(whiteRaw) * resolution
	// https://www.vishay.com/docs/84323/designingveml7700.pdf?utm_source=chatgpt.com

	overrun := false
	if lux >= maxLux {
		// log.Printf("Lux > Maxlux - 1")
		overrun = true
		lux = maxLux
	} else if lux > 1000 && (gain == VEML7700_GAIN_04 || gain == VEML7700_GAIN_08) {
		lux = (((6.0135e-13*lux-9.3924e-9)*lux+8.1488e-5)*lux + 1.0023) * lux
	}
	if lux >= maxLux {
		// log.Printf("Lux > Maxlux - 2")
		overrun = true
		lux = maxLux
	}

	return float32(lux), float32(white), luxRaw, whiteRaw, overrun, cfg, nil
}

func (veml *Veml7700) loop() {
	var start time.Time
	var duration time.Duration
	sleep := veml.measureCycle
	defaultSleep := time.Duration(250 * time.Millisecond)
	for {
		time.Sleep(sleep)
		switch veml.state {
		case veml7700States_init:
			err := veml.init()
			if err != nil {
				log.Printf("VEML7700 init failed with %v", err)
			} else {
				veml.state = veml7700States_cfg
			}
			sleep = defaultSleep

		case veml7700States_cfg:
			err := veml.configure(VEML7700_DEFAULT_CFG)
			if err != nil {
				veml.state = veml7700States_init
				log.Printf("VEML7700 configure failed with %v", err)
			} else {
				veml.state = veml7700States_measure
			}
			sleep = defaultSleep

		case veml7700States_measure:
			start = time.Now()
			lux, white, luxRaw, whiteRaw, overrun, cfg, err := veml.measure()
			if err != nil {
				veml.state = veml7700States_init
				log.Printf("VEML7700 measure failed with %v", err)
			} else {
				var ratio float64
				if luxRaw > 0 {
					ratio = float64(whiteRaw) / float64(luxRaw)
				}
				// log.Printf("VEML7700 lux: %.3f, Overrun: %v, Ratio: %.3f", lux, overrun, ratio)
				veml.shm.lux.Store(uint32(lux * 100))
				veml.shm.white.Store(uint32(white * 100))
				veml.shm.overrun.Store(overrun)
				veml.shm.whiteAlsRatio.Store(uint32(ratio * 1000.0))

				if veml.doAdjust {
					veml.adjust(luxRaw, cfg)
				}

				duration = time.Since(start)
				// log.Printf("VEML7700 measure cycle time %d µsec", duration.Microseconds())
				if duration < veml.measureCycle {
					sleep = veml.measureCycle - duration
				} else {
					log.Printf("VEML7700 measure tooks too long %d nsec", duration)
					sleep = veml.measureCycle
				}
			}
		}
	}
}
