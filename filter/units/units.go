package units

import "math"

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
