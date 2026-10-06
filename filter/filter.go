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

package filter

import (
	"fmt"
	"math"
	"sync/atomic"
)

type AvgFilter struct {
	value atomic.Uint32
	fc    atomic.Uint32 // e.g 0.15
	first atomic.Bool
}

func NewAvgFilter(filterConstant, value float32) (*AvgFilter, error) {
	if filterConstant <= 1.0 {
		a := &AvgFilter{}
		a.first.Store(true)
		a.value.Store(math.Float32bits(value))
		a.fc.Store(math.Float32bits(filterConstant))
		return a, nil
	} else {
		return nil, fmt.Errorf("Invalid filter constant")
	}
}

func (f *AvgFilter) Reset(value float32) {
	f.value.Store(math.Float32bits(value))
	f.first.Store(true)
}

func (f *AvgFilter) SetFilterConstant(fc float32) {
	f.fc.Store(math.Float32bits(fc))
}

func (f *AvgFilter) Add(value float32) {
	if f.first.Load() {
		f.value.Store(math.Float32bits(value))
		f.first.Store(false)
	} else {
		fc := math.Float32frombits(f.fc.Load())
		f.value.Store(math.Float32bits(value*fc + (1.0-fc)*math.Float32frombits(f.value.Load())))
	}
}

func (f *AvgFilter) GetValue() float32 {
	return math.Float32frombits(f.value.Load())
}

func (f *AvgFilter) GetValueExt() (float32, bool) {
	return math.Float32frombits(f.value.Load()), !f.first.Load()
}
