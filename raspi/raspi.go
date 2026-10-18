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

// Raspberry Pi stuff

package raspi

import (
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"weatherd/cfg"
)

var RebbotRequested atomic.Bool

// in Hz
func GetCPUFrequency() (uint64, error) {
	out, err := exec.Command("vcgencmd", "measure_clock", "arm").Output()
	if err != nil {
		return 0, err
	}

	var hz uint64
	_, err = fmt.Sscanf(string(out), "frequency(0)=%d", &hz)
	if err != nil {
		return 0, err
	}

	return hz, nil
}

/*
https://www.raspberrypi.com/news/heating-and-cooling-raspberry-pi-5/
https://www.raspberrypi.com/documentation/computers/raspberry-pi.html#cooling-raspberry-pi-5
It is actively managed by the Raspberry Pi firmware:

[pi5]
# Stufe 0: Ab 50°C einschalten, bei 45°C ausschalten, ~30% Geschwindigkeit
dtparam=fan_temp0=50000
dtparam=fan_temp0_hyst=5000
dtparam=fan_temp0_speed=75

# Stufe 1: Ab 60°C erhöhen, bei 55°C verringern, ~50% Geschwindigkeit
dtparam=fan_temp1=60000
dtparam=fan_temp1_hyst=5000
dtparam=fan_temp1_speed=125

# Stufe 2: Ab 67.5°C erhöhen, bei 62.5°C verringern, ~70% Geschwindigkeit
dtparam=fan_temp2=67500
dtparam=fan_temp2_hyst=5000
dtparam=fan_temp2_speed=175

# Stufe 3: Ab 75°C volle Leistung, bei 70°C verringern, 100% Geschwindigkeit
dtparam=fan_temp3=75000
dtparam=fan_temp3_hyst=5000
dtparam=fan_temp3_speed=255
*/
func GetFanRPM() (int, error) {
	entries, err := os.ReadDir("/sys/class/hwmon")
	if err != nil {
		return -1, err
	}

	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "hwmon") {
			continue
		}

		base := filepath.Join("/sys/class/hwmon", entry.Name())

		nameData, err := os.ReadFile(filepath.Join(base, "name"))
		if err != nil {
			continue
		}

		if strings.TrimSpace(string(nameData)) != "pwmfan" {
			continue
		}

		rpmData, err := os.ReadFile(filepath.Join(base, "fan1_input"))
		if err != nil {
			return -1, err
		}

		rpm, err := strconv.Atoi(strings.TrimSpace(string(rpmData)))
		if err != nil {
			return -1, err
		}

		return rpm, nil
	}

	return -1, fmt.Errorf("pwmfan/fan1_input not found")
}

type cpuUsage struct {
	oldTotal uint64
	oldIdle  uint64
}

var cpuUsageData = cpuUsage{}

func (c *cpuUsage) get() (uint16, error) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return 0, err
	}
	defer f.Close()

	var user, nice, system, idle, iowait, irq, softirq, steal uint64

	_, err = fmt.Fscanf(f, "cpu %d %d %d %d %d %d %d %d",
		&user, &nice, &system, &idle, &iowait, &irq, &softirq, &steal)
	if err != nil {
		return 0, err
	}

	idleAll := idle + iowait
	total := user + nice + system + idle + iowait + irq + softirq + steal

	if c.oldTotal == 0 {
		c.oldTotal = total
		c.oldIdle = idleAll
		return 0, nil
	}

	totalDelta := total - c.oldTotal
	idleDelta := idleAll - c.oldIdle

	c.oldTotal = total
	c.oldIdle = idleAll

	if totalDelta == 0 {
		return 0, nil
	}

	return uint16(1000 * (1.0 - float64(idleDelta)/float64(totalDelta))), nil
}

// in %*10
func GetCpuUsage() (uint16, error) {
	return cpuUsageData.get()
}

// returns temperature*100
func GetTemperature() (int, error) {
	data, err := os.ReadFile("/sys/class/thermal/thermal_zone0/temp")
	if err != nil {
		return 0, err
	}
	var temp int
	n, err := fmt.Sscanf(string(data), "%d", &temp)
	if err != nil {
		return 0, err
	}
	if n != 1 {
		return 0, fmt.Errorf("scanf failed")
	}
	return temp, nil
}

// in mV
func GetVoltages() (v5, v3v3 int, err error) {
	out, err := exec.Command("vcgencmd", "pmic_read_adc", "3V3_SYS_V", "EXT5V_V").Output()
	if err != nil {
		return 0, 0, err
	}

	for _, line := range strings.Split(string(out), "\n") {
		name, rest, ok := strings.Cut(strings.TrimSpace(line), " ")
		if !ok {
			continue
		}

		_, value, ok := strings.Cut(rest, "=")
		if !ok {
			continue
		}

		value = strings.TrimSuffix(strings.TrimSpace(value), "V")
		v, parseErr := strconv.ParseFloat(value, 64)
		if parseErr != nil {
			continue
		}

		switch name {
		case "3V3_SYS_V":
			v3v3 = int(v * 1000)
		case "EXT5V_V":
			v5 = int(v * 1000)
		}
	}

	if v5 == 0 || v3v3 == 0 {
		return 0, 0, fmt.Errorf("Spannungsmessung unvollständig: 5 V=%d mV, 3,3 V=%d mV", v5, v3v3)
	}

	return v5, v3v3, nil
}

// in mv
func Get3V3() (int, error) {
	out, err := exec.Command("vcgencmd", "pmic_read_adc", "3V3_SYS_V").Output()
	if err != nil {
		return 0, err
	}

	// Beispiel: 3V3_SYS_V volt(9)=3.32580900V
	s := strings.TrimSpace(string(out))
	_, value, ok := strings.Cut(s, "=")
	if !ok {
		return 0, fmt.Errorf("unerwartete Ausgabe: %s", s)
	}

	value = strings.TrimSuffix(value, "V")
	v, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, err
	}

	return int(v * 1000), nil
}

// in mv
func Get5VVoltage() (int, error) {
	cmd := exec.Command("vcgencmd", "pmic_read_adc", "EXT5V_V")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	line := string(output)
	pos := strings.IndexByte(line, '=')
	if pos < 0 {
		return 0, fmt.Errorf("invalid vcgencmd output: %q", line)
	}

	var voltage float64
	n, err := fmt.Sscanf(line[pos+1:], "%fV", &voltage)
	if err != nil {
		return 0, err
	}
	if n != 1 {
		return 0, fmt.Errorf("invalid voltage: %q", line)
	}

	return int(voltage*1000.0 + 0.5), nil
}

// 5A PD ok
// od -A n -d -j 2 --endian=big /proc/device-tree/chosen/power/max_current
func GetMaxCurrent() (uint16, error) {
	data, err := os.ReadFile("/proc/device-tree/chosen/power/max_current")
	if err != nil {
		return 0, err
	}
	if len(data) < 4 {
		return 0, fmt.Errorf("file too short (%d Bytes)", len(data))
	}
	return binary.BigEndian.Uint16(data[2:4]), nil
}

/*
https://www.raspberrypi.com/documentation/computers/os.html
now:
0x1			Undervoltage detected
0x2			Arm frequency capped
0x4			Currently throttled
0x8			Soft temperature limit active

since last boot:
0x10000		Undervoltage has occurred
0x20000		Arm frequency capping has occurred
0x40000		Throttling has occurred
0x80000		Soft temperature limit has occurred
*/

func GetThrottled() (uint32, error) {
	cmd := exec.Command("vcgencmd", "get_throttled")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	var value uint32
	_, err = fmt.Sscanf(string(output), "throttled=0x%x", &value)
	if err != nil {
		return 0, err
	}

	return value, nil
}

func GetSysVersion() (uint16, error) {
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return 0, err
	}

	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "DEBIAN_VERSION_FULL=") {
			continue
		}

		var major, minor uint
		n, err := fmt.Sscanf(line[len("DEBIAN_VERSION_FULL="):], "%d.%d", &major, &minor)
		if err != nil {
			return 0, err
		}

		if n < 1 {
			return 0, fmt.Errorf("invalid VERSION_ID: %q", line)
		}
		return uint16((major << 8) | minor), nil
	}

	return 0, fmt.Errorf("VERSION_ID not found")
}

func GetSerial() (string, error) {
	data, err := os.ReadFile("/sys/firmware/devicetree/base/serial-number")
	if err != nil {
		return "", err
	}
	return strings.TrimLeft(strings.TrimRight(string(data), "\x00\r\n "), "\x00\r\n "), nil
}

// in kB
func GetRam() (used, total uint32, err error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0, err
	}

	var memTotal, memAvailable uint32

	for _, line := range strings.Split(string(data), "\n") {
		var value uint32

		if strings.HasPrefix(line, "MemTotal:") {
			fmt.Sscanf(line, "MemTotal: %d kB", &value)
			memTotal = value
		} else if strings.HasPrefix(line, "MemAvailable:") {
			fmt.Sscanf(line, "MemAvailable: %d kB", &value)
			memAvailable = value
		}
	}

	if memTotal == 0 {
		return 0, 0, fmt.Errorf("MemTotal not found")
	}

	return memTotal - memAvailable, memTotal, nil
}

// sudo -E rpi-eeprom-config --edit
// POWER_OFF_ON_HALT=1
// WAKE_ON_GPIO=0
func Restart(config *cfg.Cfg) {
	go func() {
		RebbotRequested.Store(true)

		time.Sleep(3 * time.Second)
		cmd := exec.Command("sh", "-c", fmt.Sprintf("echo +%d > /sys/class/rtc/rtc0/wakealarm", config.GetIntValue(cfg.SECTION_SYSTEM, cfg.KEY_SYSTEM_RESTART_POFF_TIME, cfg.DEF_RESTART_POFF_TIME)))
		err := cmd.Run()
		if err == nil {
			config.SetIntValue(cfg.SECTION_RUN, cfg.KEY_RUN_REBOOT,
				config.GetIntValue(cfg.SECTION_RUN, cfg.KEY_RUN_REBOOT, 0)+1, "")
			config.Save()
			exec.Command("systemctl", "poweroff").Run()
		}
	}()
}

// kB
func GetDiskSpace() (free, total uint64, err error) {
	var stat syscall.Statfs_t

	if err := syscall.Statfs("/", &stat); err != nil {
		return 0, 0, err
	}

	total = stat.Blocks * uint64(stat.Bsize) / 1024
	free = stat.Bavail * uint64(stat.Bsize) / 1024

	return free, total, nil
}

// sec
func GetUptime() (uint32, error) {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, err
	}

	var seconds float64
	_, err = fmt.Sscanf(string(data), "%f", &seconds)
	if err != nil {
		return 0, err
	}
	return uint32(seconds), nil
}
