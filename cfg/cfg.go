package cfg

import (
	"strconv"

	"gopkg.in/ini.v1"
)

const (
	DEF_TEMPERATURE_FC  = 0.2   // nach ~ 6 Werten 75%,
	DEF_TEMPERATURE_FC2 = 0.002 // nach ~ 693 Werten 75%,
	DEF_HUMIDITY_FC     = 0.1   // nach 13 Werten 75%
	DEF_HUMIDITY_FC2    = 0.002
	DEF_DEWPOINT_FC     = 0.1
	DEF_DEWPOINT_FC2    = 0.002
	DEF_PRESSURE_FC     = 0.1
	DEF_LUX_FC          = 0.3 // nach 4 Werten 75%
	DEF_LUX_FC2         = 0.004

	SECTION_FILTERDATA   = "filterdata"
	KEY_FILTER_CONST_T   = "filter_constant_t"
	KEY_FILTER_CONST_T2  = "filter_constant_t2"
	KEY_FILTER_CONST_H   = "filter_constant_h"
	KEY_FILTER_CONST_H2  = "filter_constant_h2"
	KEY_FILTER_CONST_DP  = "filter_constant_dp"
	KEY_FILTER_CONST_DP2 = "filter_constant_dp2"
	KEY_FILTER_CONST_P   = "filter_constant_p"
	KEY_FILTER_CONST_L   = "filter_constant_l"
	KEY_FILTER_CONST_L2  = "filter_constant_l2"
)

type Cfg struct {
	ini     *ini.File
	cfgFile string
}

func NewCfg(file string) (*Cfg, error) {
	cfg, err := ini.Load(file)
	if err != nil {
		cfg = ini.Empty()
	}
	return &Cfg{
		ini:     cfg,
		cfgFile: file,
	}, nil
}

func (c *Cfg) Save() error {
	return c.ini.SaveTo(c.cfgFile)
}

func (c *Cfg) GetIntValue(section, key string, def int) int {
	return c.ini.Section(section).Key(key).MustInt(def)
}

func (c *Cfg) GetStringValue(section, key string, def string) string {
	return c.ini.Section(section).Key(key).MustString(def)
}

func (c *Cfg) GetFloatValue(section, key string, def float32) float32 {
	return float32(c.ini.Section(section).Key(key).MustFloat64(float64(def)))
}

func (c *Cfg) GetBoolValue(section, key string, def bool) bool {
	return c.ini.Section(section).Key(key).MustBool(def)
}

func (c *Cfg) SetStringValue(section, key, value string) {
	c.ini.Section(section).Key(key).SetValue(value)
}

func (c *Cfg) SetIntValue(section, key string, value int) {
	c.ini.Section(section).Key(key).SetValue(strconv.Itoa(value))
}

func (c *Cfg) SetFloatValue(section, key string, value float32, comment string) {
	c.ini.Section(section).Key(key).SetValue(strconv.FormatFloat(float64(value), 'f', -1, 32))
	if comment != "" {
		c.ini.Section(section).Key(key).Comment = comment
	}
}

func (c *Cfg) SetBoolValue(section, key string, value bool) {
	c.ini.Section(section).Key(key).SetValue(strconv.FormatBool(value))
}

func (c *Cfg) SetDeaultCfg() {
	c.ini = ini.Empty()
	c.SetFloatValue(SECTION_FILTERDATA, KEY_FILTER_CONST_T, DEF_TEMPERATURE_FC, "Temperatur filter")
	c.SetFloatValue(SECTION_FILTERDATA, KEY_FILTER_CONST_T2, DEF_TEMPERATURE_FC2, "Temperatur filter 2")
	c.SetFloatValue(SECTION_FILTERDATA, KEY_FILTER_CONST_H, DEF_TEMPERATURE_FC, "Humidity filter")
	c.SetFloatValue(SECTION_FILTERDATA, KEY_FILTER_CONST_H2, DEF_TEMPERATURE_FC2, "Humidity filter 2")
	c.SetFloatValue(SECTION_FILTERDATA, KEY_FILTER_CONST_DP, DEF_TEMPERATURE_FC, "Dew point filter")
	c.SetFloatValue(SECTION_FILTERDATA, KEY_FILTER_CONST_DP2, DEF_TEMPERATURE_FC2, "Dew point filter 2")
	c.SetFloatValue(SECTION_FILTERDATA, KEY_FILTER_CONST_L, DEF_TEMPERATURE_FC, "Lux filter")
	c.SetFloatValue(SECTION_FILTERDATA, KEY_FILTER_CONST_L2, DEF_TEMPERATURE_FC2, "Lux filter 2")
	c.SetFloatValue(SECTION_FILTERDATA, KEY_FILTER_CONST_P, DEF_TEMPERATURE_FC2, "Pressure filter")
}
