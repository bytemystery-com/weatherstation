package main

import (
	"fmt"
	"log"

	"weatherd/i2c"
)

func main() {
	bus, _ := i2c.NewI2c(1)
	if bus == nil {
		return
	}
	dev, _ := i2c.NewI2cDevice(0x76, bus)
	data := make([]i2c.I2cData, 2)
	data[0].Mode = i2c.I2c_write
	data[0].Data = make([]byte, 1)
	data[0].Data[0] = 0xD0
	data[1].Mode = i2c.I2c_read
	data[1].Data = make([]byte, 1)

	err := dev.Transfer(data)
	if err != nil {
		return
	}
	log.Printf("Chip id: 0x%02x\n", data[1].Data[0])

	data = make([]i2c.I2cData, 2)
	data[0].Mode = i2c.I2c_write
	data[0].Data = make([]byte, 1)
	data[0].Data[0] = 0xE1
	data[1].Mode = i2c.I2c_read
	data[1].Data = make([]byte, 7)

	err = dev.Transfer(data)
	if err != nil {
		return
	}
	for index, item := range data[1].Data {
		log.Printf("Calib %d : 0x%02x", index, item)
	}

	fmt.Println("Exit")
}
