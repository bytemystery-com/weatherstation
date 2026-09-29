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

// I2C implementation for Linux

package i2c

import (
	"errors"
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

const (
	I2C_M_RD  = 0x0001
	I2C_RDWR  = 0x0707
	I2C_M_TEN = 0x0010
)

type i2cMsg struct {
	addr  uint16
	flags uint16
	len   uint16
	buf   uintptr
}

type i2cRdwrIoctlData struct {
	msgs  uintptr
	nmsgs uint32
}

type I2c struct {
	fd *os.File
}

func NewI2c(number int) (*I2c, error) {
	f, err := os.OpenFile(fmt.Sprintf("/dev/i2c-%d", number), os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}

	return &I2c{
		fd: f,
	}, nil
}

func (i2c *I2c) Close() error {
	if i2c != nil && i2c.fd != nil {
		err := i2c.fd.Close()
		if err != nil {
			return err
		}
		i2c.fd = nil
	}
	return nil
}

type I2cDevice struct {
	bus  *I2c
	addr uint16
}

func NewI2cDevice(addr uint16, i2c *I2c) (*I2cDevice, error) {
	if i2c == nil {
		return nil, errors.New("Bus must be given")
	}
	return &I2cDevice{
		bus:  i2c,
		addr: addr,
	}, nil
}

type I2cReadWrite int

const (
	I2c_write I2cReadWrite = iota
	I2c_read
)

type I2cData struct {
	Mode I2cReadWrite
	Data []byte
}

func (device *I2cDevice) Transfer(data []I2cData) error {
	msgs := make([]i2cMsg, len(data))
	var flag uint16 = 0
	if device.addr > 127 { // 10-bit
		flag = I2C_M_TEN
	}
	var mode uint16
	var buf uintptr

	for index, item := range data {
		switch item.Mode {
		case I2c_read:
			mode = I2C_M_RD
		case I2c_write:
			mode = 0
		default:
			return fmt.Errorf("Unknown mode %v", item.Mode)
		}
		if len(item.Data) > 0 {
			buf = uintptr(unsafe.Pointer(&item.Data[0]))
		} else {
			buf = 0
		}

		msgs[index] = i2cMsg{
			addr:  device.addr,
			flags: flag | mode,
			len:   uint16(len(item.Data)),
			buf:   buf,
		}
	}
	ioctlData := i2cRdwrIoctlData{
		msgs:  uintptr(unsafe.Pointer(&msgs[0])),
		nmsgs: uint32(len(msgs)),
	}
	// Do the sys call
	r1, _, errno := syscall.Syscall(syscall.SYS_IOCTL, device.bus.fd.Fd(), I2C_RDWR, uintptr(unsafe.Pointer(&ioctlData)))
	if errno != 0 {
		return fmt.Errorf("IOCTL failed with errno %w", errno)
	}
	if int(r1) != len(msgs) {
		return fmt.Errorf("IOCTL only transferred %d messages of %d", int(r1), len(msgs))
	}
	return nil
}
