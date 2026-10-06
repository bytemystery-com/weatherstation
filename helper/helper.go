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

package helper

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

func ExistFile(file string) (bool, error) {
	_, err := os.Stat(file)
	if err == nil {
		return true, nil
	} else if errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else {
		return false, err
	}
}

func HashPassword(password string) (string, error) {
	const (
		saltLen  = 16
		timeCost = 1
		memory   = 64 * 1024 // 64 MiB
		threads  = 4
		keyLen   = 32
	)

	salt := make([]byte, saltLen)

	_, err := rand.Read(salt)
	if err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, timeCost, memory, threads, keyLen)
	salt64 := base64.RawStdEncoding.EncodeToString(salt)
	hash64 := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", memory, timeCost, threads, salt64, hash64), nil
}

func CheckPassword(password, encodedHash string) bool {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return false
	}

	if parts[1] != "argon2id" || parts[2] != "v=19" {
		return false
	}

	// m=65536,t=1,p=4
	params := strings.Split(parts[3], ",")
	if len(params) != 3 {
		return false
	}

	memory, err := strconv.ParseUint(strings.TrimPrefix(params[0], "m="), 10, 32)
	if err != nil {
		return false
	}

	iterations, err := strconv.ParseUint(strings.TrimPrefix(params[1], "t="), 10, 32)
	if err != nil {
		return false
	}

	parallelism, err := strconv.ParseUint(strings.TrimPrefix(params[2], "p="), 10, 8)
	if err != nil {
		return false
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}

	actualHash := argon2.IDKey([]byte(password), salt, uint32(iterations),
		uint32(memory), uint8(parallelism), uint32(len(expectedHash)))

	return subtle.ConstantTimeCompare(actualHash, expectedHash) == 1
}

func Fahrenheit2celsius(f float64) float64 {
	return ((f - 32.0) * 5.0) / 9.0
}

func Celsius2fahrenheit(c float64) float64 {
	return (c*9.0)/5.0 + 32.0
}

// https://www.dwd.de/DE/leistungen/pbfb_verlag_leitfaeden/pdf_einzelbaende/leitfaden6_pdf.pdf?__blob=publicationFile&v=3&utm_source=chatgpt.com
func DewPoint(temperature, humidity float64) float64 {
	gamma := math.Log(humidity/100.0) + (17.62*temperature)/(243.12+temperature)
	return (243.12 * gamma) / (17.62 - gamma)
}

func saturationVaporPressure(temp float64) float64 {
	if temp >= 0 {
		return 6.112 * math.Exp(17.62*temp/(243.12+temp))
	}
	return 6.112 * math.Exp(22.46*temp/(272.62+temp))
}

// PressureAtSeaLevel calculates the pressure reduced to mean sea level (QFF)
// according to the DWD reduction formula.
//
// pressure: station pressure in hPa
// height:   station height above sea level in m
// temp:     station temperature in °C
// humidity: relative humidity in %
//
// Result: pressure reduced to sea level in hPa.
func PressureAtSeaLevel(pressure, height, temp, humidity float64) float64 {
	const (
		g  = 9.80665 // normal gravitational acceleration [m/s²]
		R  = 287.058 // specific gas constant for dry air [J/(kg K)]
		a  = 0.0065  // temperature gradient [K/m]
		Ch = 0.12    // humidity correction [K/hPa]
	)

	// Saturation vapour pressure [hPa]
	es := saturationVaporPressure(temp)

	// Actual vapour pressure [hPa]
	e := humidity / 100.0 * es

	// Temperature at station [K]
	T := temp + 273.15

	// Mean virtual temperature of the air column.
	Tmean := T + Ch*e + a*height/2.0

	// Reduction to sea level.
	x := g * height / (R * Tmean)

	return pressure * math.Exp(x)
}
