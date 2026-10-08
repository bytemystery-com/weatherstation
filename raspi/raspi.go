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

func GetCpuUsage() (uint16, error) {
	return cpuUsageData.get()
}

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
		cmd := exec.Command("sh", "-c", fmt.Sprintf("echo +%d > /sys/class/rtc/rtc0/wakealarm", config.GetIntValue(cfg.SECTION_RUN, cfg.KEY_RUN_REBOOT, 0)))
		err := cmd.Run()
		if err == nil {
			config.SetIntValue(cfg.SECTION_RUN, cfg.KEY_RUN_REBOOT, config.GetIntValue(cfg.SECTION_RUN, cfg.KEY_RUN_REBOOT, 0)+1, "")
			exec.Command("systemctl", "poweroff").Run()
		}
	}()
}

func GetDiskSpace() (free, total uint64, err error) {
	var stat syscall.Statfs_t

	if err := syscall.Statfs("/", &stat); err != nil {
		return 0, 0, err
	}

	total = stat.Blocks * uint64(stat.Bsize) / 1024
	free = stat.Bavail * uint64(stat.Bsize) / 1024

	return free, total, nil
}
