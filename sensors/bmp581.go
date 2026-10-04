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

// I2C implementation for BMP581 sensor

package sensors

import (
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"weatherd/i2c"
)

const (
	BMP581_REG_CHIPID     = 0x01
	BMP581_REG_REVISIONID = 0x02
	BMP581_REG_IRQ_STATUS = 0x27
	BMP581_REG_STATUS     = 0x28
	BMP581_REG_DSP        = 0x30
	BMP581_REG_IIR        = 0x31
	BMP581_REG_OSR        = 0x36
	BMP581_REG_ODR        = 0x37
	BMP581_REG_CMD        = 0x7E
	BMP581_REG_DATA       = 0x1D

	BMP581_CMD_RESET = 0xB6

	BMP581_CHIPID = 0x50

	BMP581_STATUS_MASK          = 0x9F
	BMP581_STATUS_CORE_READY    = (1 << 0)
	BMP581_STATUS_NVM_READY     = (1 << 1)
	BMP581_STATUS_NVM_ERROR     = (1 << 2)
	BMP581_STATUS_NVM_CMD_ERROR = (1 << 3)
	BMP581_STATUS_BOOT_ERROR    = (1 << 4)
	BMP581_STATUS_CRACK_PASS    = (1 << 7)

	BMP581_IRQ_STATUS_POR        = (1 << 4)
	BMP581_IRQ_STATUS_DATA_READY = (1 << 0)

	BMP581_MODE_STANDBY       = 0
	BMP581_MODE_NORMAL        = 1
	BMP581_MODE_FORCED        = 2
	BMP581_MODE_NONSTOP       = 2
	BMP581_MODE_SEEP_DISABLED = (1 << 7)

	BMP581_MODE_ODR_240    = (0x0 << 2)
	BMP581_MODE_ODR_218    = (0x1 << 2)
	BMP581_MODE_ODR_199    = (0x2 << 2)
	BMP581_MODE_ODR_179    = (0x3 << 2)
	BMP581_MODE_ODR_160    = (0x4 << 2)
	BMP581_MODE_ODR_149    = (0x5 << 2)
	BMP581_MODE_ODR_140    = (0x6 << 2)
	BMP581_MODE_ODR_129    = (0x7 << 2)
	BMP581_MODE_ODR_120    = (0x8 << 2)
	BMP581_MODE_ODR_110    = (0x9 << 2)
	BMP581_MODE_ODR_100    = (0xA << 2)
	BMP581_MODE_ODR_089    = (0xB << 2)
	BMP581_MODE_ODR_080    = (0xC << 2)
	BMP581_MODE_ODR_070    = (0xD << 2)
	BMP581_MODE_ODR_060    = (0xE << 2)
	BMP581_MODE_ODR_050    = (0xF << 2)
	BMP581_MODE_ODR_045    = (0x10 << 2)
	BMP581_MODE_ODR_040    = (0x11 << 2)
	BMP581_MODE_ODR_035    = (0x12 << 2)
	BMP581_MODE_ODR_030    = (0x13 << 2)
	BMP581_MODE_ODR_025    = (0x14 << 2)
	BMP581_MODE_ODR_020    = (0x15 << 2)
	BMP581_MODE_ODR_015    = (0x16 << 2)
	BMP581_MODE_ODR_010    = (0x17 << 2)
	BMP581_MODE_ODR_005    = (0x18 << 2)
	BMP581_MODE_ODR_004    = (0x19 << 2)
	BMP581_MODE_ODR_003    = (0x1A << 2)
	BMP581_MODE_ODR_002    = (0x1B << 2)
	BMP581_MODE_ODR_001    = (0x1C << 2)
	BMP581_MODE_ODR_0005   = (0x1D << 2)
	BMP581_MODE_ODR_00025  = (0x1E << 2)
	BMP581_MODE_ODR_000125 = (0x1F << 2)

	BMP581_OSRT_1   = (0x00)
	BMP581_OSRT_2   = (0x01)
	BMP581_OSRT_4   = (0x02)
	BMP581_OSRT_8   = (0x03)
	BMP581_OSRT_16  = (0x04)
	BMP581_OSRT_32  = (0x05)
	BMP581_OSRT_64  = (0x06)
	BMP581_OSRT_128 = (0x07)

	BMP581_OSRP_1   = (0x00 << 3)
	BMP581_OSRP_2   = (0x01 << 3)
	BMP581_OSRP_4   = (0x02 << 3)
	BMP581_OSRP_8   = (0x03 << 3)
	BMP581_OSRP_16  = (0x04 << 3)
	BMP581_OSRP_32  = (0x05 << 3)
	BMP581_OSRP_64  = (0x06 << 3)
	BMP581_OSRP_128 = (0x07 << 3)

	BMP581_OSR_PRESSURE_ENABLED = (1 << 6)

	BMP581_DSP_COMPENSATION_NONE = (0x0)
	BMP581_DSP_COMPENSATION_T    = (0x01)
	BMP581_DSP_COMPENSATION_PT   = (0x02)
	BMP581_DSP_COMPENSATION_PT2  = (0x03)

	BMP581_DSP_IIR_FLUSH   = (1 << 2)
	BMP581_DSP_AFTER_IIR_T = (1 << 3)
	BMP581_DSP_AFTER_IIR_P = (1 << 5)

	BMP581_IIR_T_0   = (0)
	BMP581_IIR_T_1   = (1)
	BMP581_IIR_T_3   = (2)
	BMP581_IIR_T_7   = (3)
	BMP581_IIR_T_15  = (4)
	BMP581_IIR_T_31  = (5)
	BMP581_IIR_T_63  = (6)
	BMP581_IIR_T_127 = (7)

	BMP581_IIR_P_0   = (0 << 3)
	BMP581_IIR_P_1   = (1 << 3)
	BMP581_IIR_P_3   = (2 << 3)
	BMP581_IIR_P_7   = (3 << 3)
	BMP581_IIR_P_15  = (4 << 3)
	BMP581_IIR_P_31  = (5 << 3)
	BMP581_IIR_P_63  = (6 << 3)
	BMP581_IIR_P_127 = (7 << 3)
)

type bmp581States int

const (
	bmp581States_init = iota
	bmp581States_cfg
	bmp581States_measure
)

type bmp851ShmData struct {
	temperature atomic.Int32  // temperature*100
	pressure    atomic.Uint32 // druck*10
	errors      atomic.Uint32
	isValid     atomic.Bool
}

type Bmp581 struct {
	dev          *i2c.I2cDevice
	state        bmp581States
	shm          bmp851ShmData
	measureCycle time.Duration
	waitDuration time.Duration
}

func NewBmp581(addr uint16, bus *i2c.I2c) *Bmp581 {
	dev, err := i2c.NewI2cDevice(addr, bus)
	if err != nil {
		return nil
	}
	bmp := Bmp581{
		dev:          dev,
		state:        bmp581States_init,
		waitDuration: time.Duration(10 * time.Millisecond),
	}
	bmp.shm.isValid.Store(false)
	return &bmp
}

// Interface for web server
func (bmp *Bmp581) GetName() string {
	return "BMP581"
}

func (bmp *Bmp581) GetData() (float32, float32) {
	return float32(bmp.shm.temperature.Load() / 100), float32(bmp.shm.pressure.Load() / 10.0)
}

func (bmp *Bmp581) Start(measureCycle time.Duration) {
	bmp.measureCycle = measureCycle
	go bmp.loop()
}

// Implementation
func (bmp *Bmp581) isValidChipId(chipId uint8) bool {
	return chipId == BMP581_CHIPID
}

func (bmp *Bmp581) getChipId() (uint8, error) {
	data := make([]i2c.I2cData, 2)
	data[0].Mode = i2c.I2c_write
	data[0].Data = []byte{BMP581_REG_CHIPID}
	data[1].Mode = i2c.I2c_read
	data[1].Data = make([]byte, 1)

	err := bmp.dev.Transfer(data)
	if err != nil {
		return 0, err
	}
	return data[1].Data[0], nil
}

func (bmp *Bmp581) getRevisionId() (uint8, error) {
	data := make([]i2c.I2cData, 2)
	data[0].Mode = i2c.I2c_write
	data[0].Data = []byte{BMP581_REG_REVISIONID}
	data[1].Mode = i2c.I2c_read
	data[1].Data = make([]byte, 1)

	err := bmp.dev.Transfer(data)
	if err != nil {
		return 0, err
	}
	return data[1].Data[0], nil
}

func (bmp *Bmp581) isStatusOk(status uint8) (bool, error) {
	status &= BMP581_STATUS_MASK
	if (status & BMP581_STATUS_CORE_READY) == 0 {
		return false, fmt.Errorf("Core not ready")
	}
	if (status & BMP581_STATUS_NVM_READY) == 0 {
		return false, fmt.Errorf("Nvm not ready")
	}
	if (status & BMP581_STATUS_NVM_ERROR) != 0 {
		return false, fmt.Errorf("Nvm error")
	}
	if (status & BMP581_STATUS_NVM_CMD_ERROR) != 0 {
		return false, fmt.Errorf("Nvm cmd error")
	}
	if (status & BMP581_STATUS_BOOT_ERROR) != 0 {
		return false, fmt.Errorf("Boot error")
	}
	if (status & BMP581_STATUS_CRACK_PASS) == 0 {
		return false, fmt.Errorf("Crack test not passed")
	}
	return true, nil
}

func (bmp *Bmp581) getStatus() (uint8, error) {
	data := make([]i2c.I2cData, 2)
	data[0].Mode = i2c.I2c_write
	data[0].Data = []byte{BMP581_REG_STATUS}
	data[1].Mode = i2c.I2c_read
	data[1].Data = make([]byte, 1)

	err := bmp.dev.Transfer(data)
	if err != nil {
		return 0, err
	}
	return data[1].Data[0], nil
}

func (bmp *Bmp581) getIrqStatus() (uint8, error) {
	data := make([]i2c.I2cData, 2)
	data[0].Mode = i2c.I2c_write
	data[0].Data = []byte{BMP581_REG_IRQ_STATUS}
	data[1].Mode = i2c.I2c_read
	data[1].Data = make([]byte, 1)

	err := bmp.dev.Transfer(data)
	if err != nil {
		return 0, err
	}
	return data[1].Data[0], nil
}

func (bmp *Bmp581) init() error {
	id, err := bmp.getChipId()
	if err != nil {
		return err
	}
	if !bmp.isValidChipId(id) {
		return fmt.Errorf("Invalid chip id %d", id)
	}

	data := make([]i2c.I2cData, 1)
	data[0].Mode = i2c.I2c_write
	data[0].Data = []byte{BMP581_REG_CMD, BMP581_CMD_RESET}

	err = bmp.dev.Transfer(data)
	if err != nil {
		return err
	}

	var iter int = 0
	for {
		time.Sleep(bmp.waitDuration)
		status, err := bmp.getIrqStatus()
		if err != nil {
			return err
		}
		if (status & BMP581_IRQ_STATUS_POR) != 0 {
			break
		}
		if iter > 50 {
			return fmt.Errorf("Status ok timeout")
		}
		iter++
	}

	status, err := bmp.getStatus()
	if err != nil {
		return err
	}
	ok, err := bmp.isStatusOk(status)
	if ok {
		return nil
	}
	return err
}

func (bmp *Bmp581) configure() error {
	// Mode
	data := make([]i2c.I2cData, 2)
	data[0].Mode = i2c.I2c_write
	data[0].Data = []byte{
		// Mode
		BMP581_REG_ODR, BMP581_MODE_FORCED | BMP581_MODE_ODR_001,
		// OSR
		BMP581_REG_OSR, BMP581_OSRT_4 | BMP581_OSRP_8 | BMP581_OSR_PRESSURE_ENABLED,
		// DSP
		BMP581_REG_DSP, BMP581_DSP_COMPENSATION_PT | BMP581_DSP_AFTER_IIR_P | BMP581_DSP_AFTER_IIR_T,
		// IIR
		BMP581_REG_IIR, BMP581_IIR_T_0 | BMP581_IIR_P_0,
	}
	err := bmp.dev.Transfer(data)
	if err != nil {
		return err
	}

	// OSR
	data = make([]i2c.I2cData, 2)
	data[0].Mode = i2c.I2c_write
	data[0].Data = []byte{BMP581_REG_OSR, BMP581_OSRT_4 | BMP581_OSRP_8 | BMP581_OSR_PRESSURE_ENABLED}
	err = bmp.dev.Transfer(data)
	if err != nil {
		return err
	}

	// DSP
	data = make([]i2c.I2cData, 2)
	data[0].Mode = i2c.I2c_write
	data[0].Data = []byte{BMP581_REG_DSP, BMP581_OSRT_4 | BMP581_OSRP_8 | BMP581_OSR_PRESSURE_ENABLED}
	err = bmp.dev.Transfer(data)
	if err != nil {
		return err
	}

	return nil
}

func (bmp *Bmp581) measure() (float32, float32, error) {
	// Mode
	data := make([]i2c.I2cData, 2)
	data[0].Mode = i2c.I2c_write
	data[0].Data = []byte{BMP581_REG_ODR}
	data[1].Data = make([]byte, 1)

	err := bmp.dev.Transfer(data)
	if err != nil {
		return 0, 0, err
	}
	data[1].Data[0] |= BMP581_MODE_FORCED
	err = bmp.dev.Transfer(data)
	if err != nil {
		return 0, 0, err
	}

	var counter int = 0
	for {
		time.Sleep(bmp.waitDuration)
		status, err := bmp.getIrqStatus()
		if err != nil {
			return 0, 0, err
		}
		if (status & BMP581_IRQ_STATUS_DATA_READY) != 0 {
			break
		}
		if counter > 20 {
			return 0, 0, fmt.Errorf("Measure timeout")
		}
		counter++
	}

	data = make([]i2c.I2cData, 2)
	data[0].Mode = i2c.I2c_write
	data[0].Data = []byte{BMP581_REG_DATA}
	data[1].Data = make([]byte, 6)

	err = bmp.dev.Transfer(data)
	if err != nil {
		return 0, 0, err
	}

	var tempRaw uint32 = uint32(data[1].Data[0]) + uint32(data[1].Data[1])<<8 + uint32(data[1].Data[2])<<16
	var presRaw uint32 = uint32(data[1].Data[3]) + uint32(data[1].Data[4])<<8 + uint32(data[1].Data[5])<<16
	temp := int32(tempRaw<<8) >> 8
	pres := int32(presRaw<<8) >> 8
	return float32(temp) / 0x10000, float32(pres) / 0x40, nil
}

func (bmp *Bmp581) loop() {
	var start time.Time
	var duration time.Duration
	sleep := bmp.measureCycle
	defaultSleep := time.Duration(250 * time.Millisecond)
	for {
		time.Sleep(sleep)
		switch bmp.state {
		case bmp581States_init:
			err := bmp.init()
			if err != nil {
				bmp.shm.errors.Add(1)
				bmp.shm.isValid.Store(false)
				log.Printf("BMP581 init failed with %v", err)
			}
			bmp.state = bmp581States_cfg
			sleep = defaultSleep

		case bmp581States_cfg:
			err := bmp.configure()
			if err != nil {
				bmp.state = bmp581States_init
				bmp.shm.errors.Add(1)
				bmp.shm.isValid.Store(false)
				log.Printf("BMP581 configure failed with %v", err)
			}
			bmp.state = bmp581States_measure
			sleep = defaultSleep

		case bmp581States_measure:
			start = time.Now()
			temp, pressure, err := bmp.measure()
			if err != nil {
				bmp.state = bmp581States_init
				log.Printf("BMP581 measure failed with %v", err)
				bmp.shm.errors.Add(1)
				bmp.shm.isValid.Store(false)
			} else {
				bmp.shm.temperature.Store(int32(temp * 100))
				bmp.shm.pressure.Store(uint32(pressure * 10))
				bmp.shm.isValid.Store(true)
			}
			duration = time.Since(start)
			if duration < bmp.measureCycle {
				sleep = bmp.measureCycle - duration
			} else {
				log.Printf("BMP581 measure tooks too long %d nsec", duration)
				sleep = bmp.measureCycle
			}
		}
	}
}
