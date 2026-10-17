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
	"encoding/hex"
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

// https://github.com/davidromps/heatindex
// https://journals.ametsoc.org/view/journals/apme/65/2/JAMC-D-25-0130.1.xml?utm_source=chatgpt.com

// Physikalische Konstanten nach den Rankine-Kirchhoff-Spezifikationen (Romps 2026)
const (
	cPd     = 1005.7  // Spezifische Wärmekapazität trockene Luft (J/kg/K)
	cPv     = 1867.0  // Spezifische Wärmekapazität Wasserdampf (J/kg/K)
	cCl     = 4182.0  // Spezifische Wärmekapazität flüssiges Wasser (J/kg/K)
	rD      = 287.04  // Gaskonstante trockene Luft (J/kg/K)
	rV      = 461.53  // Gaskonstante Wasserdampf (J/kg/K)
	pTrip   = 611.65  // Tripelpunkt-Druck von Wasser (Pa)
	tTrip   = 273.16  // Tripelpunkt-Temperatur von Wasser (K)
	e0      = 2.451e6 // Latente Wärme der Verdampfung am Tripelpunkt (J/kg)
	epsilon = rD / rV
)

// saturationVaporPressureRomps berechnet den Sättigungsdampfdruck exakt nach
// der integrierten Clausius-Clapeyron-Gleichung (Rankine-Kirchhoff-Approximation).
// Hierbei wird die Temperaturabhängigkeit der latenten Wärme mathematisch korrekt berücksichtigt.
func saturationVaporPressureRomps(tKelvin float64) float64 {
	// Exakte thermodynamische Integration nach Romps (2026)
	exponent := (e0/(rV*tTrip)-(cPv-cCl)/rV)*(1.0-tTrip/tKelvin) +
		((cPv-cCl)/rV)*math.Log(tKelvin/tTrip)

	return pTrip * math.Exp(exponent)
}

// RompsWetBulb berechnet die thermodynamische Feuchttemperatur in °C.
// tC   : Lufttemperatur (Trockentemperatur) in °C
// rhPct: Relative Luftfeuchtigkeit in % (0.0 bis 100.0)
// pPa  : Realer Stationsluftdruck in Pascal (z.B. 101325 für Meereshöhe)
func GetRompsWetBulb(tC, rhPct, pPa float64) (float64, error) {
	if rhPct < 0.0 || rhPct > 100.0 || pPa <= 0.0 {
		return 0.0, fmt.Errorf("ungültige Eingabewerte: RH=%v, Druck=%v", rhPct, pPa)
	}

	t := tC + 273.15
	rh := rhPct / 100.0

	// Aktuellen Dampfdruck (e) und Mischungsverhältnis (w) berechnen
	pSatT := saturationVaporPressureRomps(t)
	e := rh * pSatT

	if e >= pPa {
		return 0.0, fmt.Errorf("physikalischer Grenzfehler: Partialdampfdruck (%v Pa) übersteigt Gesamtluftdruck (%v Pa)", e, pPa)
	}
	w := epsilon * e / (pPa - e)

	// Iterativer Newton-Raphson Solver
	tW := t // Startwert
	const maxIterations = 100
	const tolerance = 1e-6

	for i := 0; i < maxIterations; i++ {
		pSatTw := saturationVaporPressureRomps(tW)
		if pSatTw >= pPa {
			pSatTw = pPa - 1e-3 // Schutz vor Singularität / Division durch Null
		}

		wSatTw := epsilon * pSatTw / (pPa - pSatTw)

		// Temperaturabhängige latente Wärme der Verdampfung exakt berechnet bei tW
		lTw := e0 + (cPv-cCl)*(tW-tTrip)

		// f(tW) ist die thermodynamische Energiebilanz-Gleichung (Enthalpiedifferenz)
		f := (cPd+w*cPv)*(t-tW) - lTw*(wSatTw-w)

		// Erste Ableitung f'(tW) unter exakter Einbeziehung der Clausius-Clapeyron-Steigung
		dpSatDTr := pSatTw * (e0 + (cPv-cCl)*(tW-tTrip)) / (rV * tW * tW)
		dwSatDTr := epsilon * pPa * dpSatDTr / ((pPa - pSatTw) * (pPa - pSatTw))
		dfDTw := -(cPd + w*cPv) - lTw*dwSatDTr - (cPv-cCl)*(wSatTw-w)

		// Newton-Schrittausführung
		tWNew := tW - f/dfDTw

		// Bei erfolgreicher Konvergenz innerhalb der Toleranzgrenze abbrechen
		if math.Abs(tWNew-tW) < tolerance {
			return math.Round((tWNew-273.15)*100) / 100, nil
		}

		tW = tWNew
	}

	// KRITISCH: Keine stillschweigende Rückgabe ungenauer Werte bei mathematischer Divergenz
	return 0.0, fmt.Errorf("algebraischer Fehler: Newton-Raphson-Verfahren konvergiert nicht innerhalb von %d Iterationen", maxIterations)
}

// returns a0, a1, and goodness
// http://elsenaju.info/Rechnen/Ausgleichsgerade.htm
// y = a1*x + a0
func CalculateRegression(x, y []float64) (a1, a0, goodness float64) {
	n := len(x)
	if n == 0 || len(y) != n {
		return 0, 0, 0
	}

	// y = a1*x + a0
	var xavg, yavg float64

	for i := 0; i < n; i++ {
		xavg += x[i]
		yavg += y[i]
	}

	xavg /= float64(n)
	yavg /= float64(n)

	var s1, s2 float64

	for i := 0; i < n; i++ {
		dx := x[i] - xavg
		dy := y[i] - yavg

		s1 += dx * dy
		s2 += dx * dx
	}

	if s2 == 0 {
		return 0, yavg, 0
	}

	a1 = s1 / s2
	a0 = yavg - a1*xavg

	// goodness
	var sum float64

	for i := 0; i < n; i++ {
		if y[i] != 0 {
			delta := (y[i] - (x[i]*a1 + a0)) / y[i]
			sum += delta * delta
		}
	}

	if sum < 1.0 {
		goodness = math.Sqrt(1.0 - sum)
	}
	return
}

func IsRoot() bool {
	if os.Getuid() == 0 {
		return true
	} else if os.Geteuid() == 0 {
		return true
	}
	return false
}

func NewSessionID() (string, error) {
	b := make([]byte, 32)

	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
