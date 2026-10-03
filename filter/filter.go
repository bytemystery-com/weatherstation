package filter

import (
	"fmt"
	"math"
	"sync/atomic"
)

type AvgFilter struct {
	value atomic.Uint32
	fc    atomic.Uint32 // e.g 0.15
	first bool
}

func NewAvgFilter(filterConstant, value float32) (*AvgFilter, error) {
	if filterConstant <= 1.0 {
		a := &AvgFilter{
			first: true,
		}
		a.value.Store(math.Float32bits(value))
		a.fc.Store(math.Float32bits(filterConstant))
		return a, nil
	} else {
		return nil, fmt.Errorf("Invalid filter constant")
	}
}

func (f *AvgFilter) Reset(value float32) {
	f.value.Store(math.Float32bits(value))
	f.first = true
}

func (f *AvgFilter) SetFilterConstant(fc float32) {
	f.fc.Store(math.Float32bits(fc))
	f.first = true
}

func (f *AvgFilter) Add(value float32) {
	if f.first {
		f.value.Store(math.Float32bits(value))
		f.first = false
	} else {
		fc := math.Float32frombits(f.fc.Load())
		f.value.Store(math.Float32bits(value*fc + (1.0-fc)*math.Float32frombits(f.value.Load())))
	}
}

func (f *AvgFilter) GetValue() float32 {
	return math.Float32frombits(f.value.Load())
}
