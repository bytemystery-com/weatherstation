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

// Simple GPIO implementation for Linux

package gpio

import (
	"fmt"
	"os"
	"runtime"
	"unsafe"

	"golang.org/x/sys/unix"
)

// pinctrl
// 0-0 after start with pull up
// 9-27 after start with pull down

type GpioPull uint8

const (
	GpioPullFloating GpioPull = iota
	GpioPullUp
	GpioPullDown
)

type gpioDirection uint8

const (
	gpioOut gpioDirection = iota
	gpioIn
)

const (
	GpioV2LineFlagUsed               = 1 << 0
	GpioV2LineFlagActiveLow          = 1 << 1
	GpioV2LineFlagInput              = 1 << 2
	GpioV2LineFlagOutput             = 1 << 3
	GpioV2LineFlagEdgeRising         = 1 << 4
	GpioV2LineFlagEdgeFalling        = 1 << 5
	GpioV2LineFlagOpenDrain          = 1 << 6
	GpioV2LineFlagOpenSource         = 1 << 7
	GpioV2LineFlagBiasPullUp         = 1 << 8
	GpioV2LineFlagBiasPullDown       = 1 << 9
	GpioV2LineFlagBiasDisabled       = 1 << 10
	GpioV2LineFlagEventClockRealtime = 1 << 11
	GpioV2LineFlagEventClockHTE      = 1 << 12
)

type GpioPin struct {
	fdLine    int
	pin       uint
	direction gpioDirection
}

func (pin *GpioPin) Close() error {
	if pin.fdLine >= 0 {
		err := unix.Close(pin.fdLine)
		if err != nil {
			return err
		}
		pin.fdLine = -1
	}
	return nil
}

func (pin *GpioPin) Set(high bool) error {
	if pin.direction != gpioOut {
		return fmt.Errorf("can not set output value on input")
	}
	if pin.fdLine < 0 {
		return fmt.Errorf("invalid line fd")
	}
	val := 0
	if high {
		val = 1
	}
	values := unix.GPIOV2LineValues{
		Bits: uint64(val),
		Mask: 1,
	}
	_, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(pin.fdLine),
		uintptr(unix.GPIO_V2_LINE_SET_VALUES_IOCTL),
		uintptr(unsafe.Pointer(&values)),
	)
	if errno != 0 {
		return fmt.Errorf("Set GPIO failed: %w", errno)
	}

	return nil
}

func (pin *GpioPin) Get() (bool, error) {
	if pin.fdLine < 0 {
		return false, fmt.Errorf("invalid line fd")
	}

	values := unix.GPIOV2LineValues{
		Bits: 0,
		Mask: 1,
	}

	_, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(pin.fdLine),
		uintptr(unix.GPIO_V2_LINE_GET_VALUES_IOCTL),
		uintptr(unsafe.Pointer(&values)),
	)
	if errno != 0 {
		return false, fmt.Errorf("get GPIO failed: %w", errno)
	}

	return values.Bits&1 != 0, nil
}

func NewGpioOutPin(device, pin uint, initHigh bool) (*GpioPin, error) {
	fd, err := os.OpenFile(fmt.Sprintf("/dev/gpiochip%d", device), os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	var request unix.GPIOV2LineRequest
	request.Offsets[0] = uint32(pin)
	request.Num_lines = 1
	copy(request.Consumer[:], "GPIO lib")
	request.Config.Flags = GpioV2LineFlagOutput

	high := 0
	if initHigh {
		high = 1
	}

	request.Config.Num_attrs = 1
	request.Config.Attrs[0] = unix.GPIOV2LineConfigAttribute{
		Attr: unix.GPIOV2LineAttribute{
			Id:    unix.GPIO_V2_LINE_ATTR_ID_OUTPUT_VALUES,
			Flags: uint64(high),
		},
		Mask: 1,
	}

	_, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		fd.Fd(),
		uintptr(unix.GPIO_V2_GET_LINE_IOCTL),
		uintptr(unsafe.Pointer(&request)),
	)
	runtime.KeepAlive(fd)
	if errno != 0 {
		return nil, fmt.Errorf("GPIO request failed: %w", errno)
	}
	return &GpioPin{
		fdLine:    int(request.Fd),
		pin:       pin,
		direction: gpioOut,
	}, nil
}

func NewGpioInPin(device, pin uint, pull GpioPull) (*GpioPin, error) {
	fd, err := os.OpenFile(fmt.Sprintf("/dev/gpiochip%d", device), os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	flags := uint64(GpioV2LineFlagInput)

	switch pull {
	case GpioPullFloating:
		flags |= GpioV2LineFlagBiasDisabled
	case GpioPullUp:
		flags |= GpioV2LineFlagBiasPullUp
	case GpioPullDown:
		flags |= GpioV2LineFlagBiasPullDown
	default:
		return nil, fmt.Errorf("invalid GPIO pull mode")
	}

	var request unix.GPIOV2LineRequest
	request.Offsets[0] = uint32(pin)
	request.Num_lines = 1
	copy(request.Consumer[:], "GPIO lib")

	request.Config.Flags = 0
	request.Config.Num_attrs = 1
	request.Config.Attrs[0] = unix.GPIOV2LineConfigAttribute{
		Attr: unix.GPIOV2LineAttribute{
			Id:    unix.GPIO_V2_LINE_ATTR_ID_FLAGS,
			Flags: flags,
		},
		Mask: 1,
	}

	_, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		fd.Fd(),
		uintptr(unix.GPIO_V2_GET_LINE_IOCTL),
		uintptr(unsafe.Pointer(&request)),
	)
	runtime.KeepAlive(fd)
	if errno != 0 {
		return nil, fmt.Errorf("GPIO request failed: %w", errno)
	}
	return &GpioPin{
		fdLine:    int(request.Fd),
		pin:       pin,
		direction: gpioIn,
	}, nil
}
