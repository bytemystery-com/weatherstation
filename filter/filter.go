package filter

import "fmt"

type AvgFilter struct {
	value          float32
	filterConstant float32 // e.g 0.15
}

func NewAvgFilter(filterConstant, value float32) (*AvgFilter, error) {
	if filterConstant <= 1.0 {
		return &AvgFilter{
			value:          value,
			filterConstant: filterConstant,
		}, nil
	} else {
		return nil, fmt.Errorf("Invalid filter constant")
	}
}

func (f *AvgFilter) Reset(value float32) {
	f.value = value
}

func (f *AvgFilter) Add(value float32) {
	f.value = value*f.filterConstant + (1.0-f.filterConstant)*f.value
}

func (f *AvgFilter) GetValue() float32 {
	return f.value
}
