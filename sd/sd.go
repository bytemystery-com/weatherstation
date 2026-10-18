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
//
// Write data in csv file on SD card

package sd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"weatherd/cfg"
	"weatherd/helper"
)

const (
	// DT;T;T2;H;H2;DP;DP2;P;PN;P2;PN2;T;T2;L;L2;WB;WB2;ESHT;EBMH;EVEML
	DATA_HEADERLINE = "# Weather station - %s\r\n# Version: %s\r\nDate Time;Temp [°C] (SHT);Temp2 [°C] (SHT);Humidity [rel %%] (SHT);Humidity2 [rel %%] (SHT);Dew point [°C] (SHT);Dew point2 [°C] (SHT);Pressure [hPa] (BMH);Pressure normal height [hPa] (BMH);Pressure2 [hPa] (BMH);Pressure normal height 2 [hPa] (BMH);Temp [°C] (BMH);Temp2 [°C] (BMH);Light [Lx] (VEML);Light2 [Lx] (VEML);Wet bulb [°C];Wet bulb2 [°C];Errors (SHT);Errors (BMH);Errors (VEML)\r\n"
)

type SdData struct {
	config *cfg.Cfg
	folder string
}

type SdDataEntry struct {
	T1_1 float32
	T1_2 float32
	H_1  float32
	H_2  float32
	Dp_1 float32
	Dp_2 float32
	P_1  float32
	Pn_1 float32
	P_2  float32
	Pn_2 float32
	T2_1 float32
	T2_2 float32
	L_1  float32
	L_2  float32
	WB_1 float32
	WB_2 float32
	E_1  uint32
	E_2  uint32
	E_3  uint32
}

func NewSdData(config *cfg.Cfg) (*SdData, error) {
	folder := config.GetStringValue(cfg.SECTION_DATA, cfg.KEY_DATA_FOLDER, cfg.DEF_DATA_FOLDER)
	err := os.MkdirAll(folder, 0o755)
	if err != nil {
		return nil, err
	}
	sd := SdData{
		config: config,
		folder: folder,
	}
	return &sd, nil
}

func (sd *SdData) buildFileName(t time.Time) string {
	return filepath.Join(sd.folder, fmt.Sprintf("%04d", t.Year()), fmt.Sprintf("%02d", t.Month()),
		fmt.Sprintf("%04d-%02d-%02d.csv", t.Year(), t.Month(), t.Day()))
}

func (sd *SdData) writeHeader(file string) error {
	err := os.MkdirAll(filepath.Dir(file), 0o755)
	if err != nil {
		return err
	}
	return os.WriteFile(file, []byte(fmt.Sprintf(DATA_HEADERLINE,
		sd.config.GetStringValue(cfg.SECTION_GENERAL, cfg.KEY_GENERAL_NAME, cfg.DEF_NAME),
		cfg.WEATHER_STATION_VERSION)), 0o644)
}

func (sd *SdData) WriteEntry(data *SdDataEntry) error {
	now := time.Now()
	return sd.writeEntryInternal(sd.buildFileName(now), data, &now)
}

func (sd *SdData) writeEntryInternal(file string, data *SdDataEntry, now *time.Time) error {
	ok, err := helper.ExistFile(file)
	if err != nil {
		return err
	}
	if !ok {
		sd.writeHeader(file)
	}
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	line := fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d;%.1f;%.1f;%.1f;%.1f;%.1f;%.1f;%.1f;%.1f;%.1f;%.1f;%.1f;%.1f;%.1f;%.1f;%.1f;%.1f;%d;%d;%d\r\n",
		now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(),
		data.T1_1, data.T1_2, data.H_1, data.H_2, data.Dp_1, data.Dp_2, data.P_1, data.Pn_1, data.P_2, data.Pn_2,
		data.T2_1, data.T2_2, data.L_1, data.L_2, data.WB_1, data.WB_2, data.E_1, data.E_2, data.E_3)

	_, err = f.WriteString(line)
	return err
}

type listEntry struct {
	Name string `json:"name"`
}

type listData struct {
	Path  string      `json:"path"`
	Dirs  []listEntry `json:"dirs"`
	Files []listEntry `json:"files"`
}

// returns true if we are in basefolder
func (sd *SdData) checkSdPath(dir string) (bool, error) {
	// Verhindern, dass wir sd.folder verlassen
	rel, err := filepath.Rel(sd.folder, dir)
	if err != nil {
		return false, err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return false, fmt.Errorf("invalid path")
	}
	return rel == ".", nil
}

func (sd *SdData) GetSd(file string) (*os.File, os.FileInfo, error) {
	file = filepath.Join(sd.folder, filepath.Clean(file))
	_, err := sd.checkSdPath(file)
	if err != nil {
		return nil, nil, err
	}
	f, err := os.Open(file)
	if err != nil {
		return nil, nil, err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, nil, err
	}
	return f, info, nil
}

func (sd *SdData) ListSd(dir string) ([]byte, error) {
	var ld listData
	ld.Path = dir
	ld.Dirs = make([]listEntry, 0, 15)
	ld.Files = make([]listEntry, 0, 40)
	dir = filepath.Join(sd.folder, filepath.Clean(dir))
	inRoot, err := sd.checkSdPath(dir)
	if err != nil {
		return nil, err
	}
	if !inRoot {
		ld.Dirs = append(ld.Dirs, listEntry{Name: ".."})
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			ld.Dirs = append(ld.Dirs, listEntry{Name: entry.Name()})
		} else {
			ld.Files = append(ld.Files, listEntry{Name: entry.Name()})
		}
	}
	slices.SortFunc(ld.Dirs, func(a, b listEntry) int {
		A := a.Name
		B := b.Name
		an := strings.ToLower(A)
		bn := strings.ToLower(B)
		if an == bn {
			if A < B {
				return -1
			}
			if A > B {
				return 1
			}
			return 0
		}
		if an < bn {
			return -1
		}
		return 1
	})
	slices.SortFunc(ld.Files, func(a, b listEntry) int {
		A := a.Name
		B := b.Name
		an := strings.ToLower(A)
		bn := strings.ToLower(B)
		if an == bn {
			if A < B {
				return -1
			}
			if A > B {
				return 1
			}
			return 0
		}
		if an < bn {
			return -1
		}
		return 1
	})
	data, err := json.Marshal(ld)
	if err != nil {
		return nil, err
	}
	return data, nil
}

type csvConfig struct {
	Interval int `json:"sd_interval"`
}

type csvEntry struct {
	Ti_y  int     `json:"ti_y"`
	Ti_m  int     `json:"ti_m"`
	Ti_d  int     `json:"ti_d"`
	Ti_h  int     `json:"ti_h"`
	Ti_mi int     `json:"ti_mi"`
	Ti_s  int     `json:"ti_s"`
	T     float32 `json:"t"`
	T2    float32 `json:"t2"`
	H     float32 `json:"h"`
	H2    float32 `json:"h2"`
	DP    float32 `json:"d"`
	DP2   float32 `json:"d2"`
	P     float32 `json:"p"`
	P2    float32 `json:"p2"`
	Pn    float32 `json:"pn"`
	Pn2   float32 `json:"pn2"`
	L     float32 `json:"l"`
	L2    float32 `json:"l2"`
	WB    float32 `json:"wb"`
	WB2   float32 `json:"wb2"`
}

type csvData struct {
	CsvConfig csvConfig  `json:"config"`
	Entries   []csvEntry `json:"data"`
}

func (sd *SdData) GetFilePath(t time.Time) string {
	path := filepath.Join(sd.folder, fmt.Sprintf("%04d", t.Year()))
	path = filepath.Join(path, fmt.Sprintf("%02d", t.Month()))
	path = filepath.Join(path, fmt.Sprintf("%04d-%02d-%02d.csv", t.Year(), t.Month(), t.Day()))
	return path
}

func (sd *SdData) ReadCsv(path string, start time.Time, end time.Time, entries *[]csvEntry) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	var version uint
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if line[0] == '#' {
			var verMaj uint
			var verMin uint
			n, err := fmt.Sscanf(line, "# Version: %d.%d", &verMaj, &verMin)
			if err == nil && n == 2 {
				version = ((verMaj & 0xFF) << 8) | (verMin & 0xFF)
			}
			continue
		}
		if line[0] < '0' || line[0] > '9' {
			continue
		}
		var year int
		var month int
		var day int
		var hour int
		var min int
		var sec int
		var entry csvEntry
		var dummy float32
		if version == 0x0000 {
			// old
			// Date Time;Temp [°C] (SHT);Humidity [rel %] (SHT);Pressure [hPa] (BMX);Pressure normal height [hPa] (BMX);
			// Dew point [°C] (SHT);Temp [°C] (BMX);Humidity [rel %] (BMX);Dew point [°C] (BMX);Light [Lx];
			// Errors (BMX);Errors (SHT);Errors (BH1750);Temp2 [°C] (SHT);Temp2 [°C] (BMX);Humidity2 [rel %] (SHT);
			// Humidity2 [rel %] (BMX);Dew point2 [°C] (SHT);Dew point2 [°C] (BMX);Light2 [Lx]

			n, err := fmt.Sscanf(line, "%d-%d-%d %d:%d:%d;%f;%f;%f;%f;%f;%f;%f;%f;%f;%f;%f;%f;%f;%f;%f;%f;%f;%f;%f",
				&year, &month, &day, &hour, &min, &sec,
				&entry.T, &entry.H, &entry.P, &entry.Pn, &entry.DP, &dummy, &dummy, &dummy, &entry.L,
				&dummy, &dummy, &dummy, &entry.T2, &dummy, &entry.H2, &dummy, &entry.DP2, &dummy, &entry.L2)
			if err != nil || n != 25 {
				continue
			}
			// Diese Werte gab es im alten csv nicht
			entry.P2 = entry.P
			entry.Pn2 = entry.Pn
		} else {
			if version >= 0x0300 {
				n, err := fmt.Sscanf(line, "%d-%d-%d %d:%d:%d;%f;%f;%f;%f;%f;%f;%f;%f;%f;%f;%f;%f;%f;%f;%f;%f;%f;%f;%f",
					&year, &month, &day, &hour, &min, &sec,
					&entry.T, &entry.T2, &entry.H, &entry.H2, &entry.DP, &entry.DP2,
					&entry.P, &entry.Pn, &entry.P2, &entry.Pn2, &dummy, &dummy, &entry.L, &entry.L2,
					&entry.WB, &entry.WB2, &dummy, &dummy, &dummy)
				if err != nil || n != 25 {
					continue
				}
			} else {
				n, err := fmt.Sscanf(line, "%d-%d-%d %d:%d:%d;%f;%f;%f;%f;%f;%f;%f;%f;%f;%f;%f;%f;%f;%f",
					&year, &month, &day, &hour, &min, &sec,
					&entry.T, &entry.T2, &entry.H, &entry.H2, &entry.DP, &entry.DP2,
					&entry.P, &entry.Pn, &entry.P2, &entry.Pn2, &dummy, &dummy, &entry.L, &entry.L2)
				if err != nil || n != 20 {
					continue
				}
			}
		}
		dateTime := time.Date(year, time.Month(month), day, hour, min, sec, 0, time.Local)
		if !dateTime.Before(start) && !dateTime.After(end) {
			entry.Ti_y = dateTime.Year()
			entry.Ti_m = int(dateTime.Month())
			entry.Ti_d = dateTime.Day()
			entry.Ti_h = dateTime.Hour()
			entry.Ti_mi = dateTime.Minute()
			entry.Ti_s = dateTime.Second()
			*entries = append(*entries, entry)
		}
	}

	err = scanner.Err()
	if err != nil {
		return err
	}
	return nil
}

func (sd *SdData) GetData(startTime, endTime time.Time) ([]byte, error) {
	data := csvData{
		CsvConfig: csvConfig{
			Interval: sd.config.GetIntValue(cfg.SECTION_DATA, cfg.KEY_DATA_INTERVAL, cfg.DEF_DATA_INTERVAL) * 1000,
		},
		Entries: make([]csvEntry, 0, 500),
	}
	start := time.Date(startTime.Year(), startTime.Month(), startTime.Day(), 0, 0, 0, 0, startTime.Location())
	end := time.Date(endTime.Year(), endTime.Month(), endTime.Day(), 23, 59, 59, 0, endTime.Location())
	var path string
	day := start
	for day.Before(end) || day.Equal(end) {
		path = sd.GetFilePath(day)
		sd.ReadCsv(path, startTime, endTime, &data.Entries)
		day = day.AddDate(0, 0, 1)
	}

	d, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return d, nil
}
