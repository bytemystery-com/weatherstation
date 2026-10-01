package sensors

import (
	"fmt"

	"weatherd/i2c"
)

const (
	VEML7700_ADDR = 0x10

	VEML7700_REG_CHIPID = 0x07

	VEML7700_CHIPID  = 0x81
	VEML7700_SUBID_1 = 0xC4
	VEML7700_SUBID_2 = 0xD4
)

type veml7700States int

const (
	veml7700States_init = iota
	veml7700States_cfg
	veml7700States_measure
)

type Veml7700 struct {
	dev   *i2c.I2cDevice
	state veml7700States
}

func NewVeml7700(addr uint16, bus *i2c.I2c) *Veml7700 {
	dev, err := i2c.NewI2cDevice(addr, bus)
	if err != nil {
		return nil
	}
	return &Veml7700{
		dev:   dev,
		state: veml7700States_init,
	}
}

func (veml *Veml7700) IsValidChipId(chipId uint16) bool {
	var subId uint8 = uint8((chipId >> 8) & 0xFF)
	return (chipId&0xFF) == VEML7700_CHIPID && (subId == VEML7700_SUBID_1 || subId == VEML7700_SUBID_2)
}

func (veml *Veml7700) GetChipId() (uint16, error) {
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
	id, err := veml.GetChipId()
	if err != nil {
		return err
	}
	if !veml.IsValidChipId(id) {
		return fmt.Errorf("Invalid chip id %d", id)
	}
	return nil
}
