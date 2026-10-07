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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"weatherd/cfg"
	"weatherd/helper"
)

const (
	// DT;T;T2;H;H2;DP;DP2;P;PN;P2;PN2;T;T2;L;L2;ESHT;EBMH;EVEML
	DATA_HEADERLINE = "# Weather station\r\n# Version: %s\r\n# Date Time;Temp [°C] (SHT);Temp2 [°C] (SHT);Humidity [rel %%] (SHT);Humidity2 [rel %%] (SHT);Dew point [°C] (SHT);Dew point2 [°C] (SHT);Pressure [hPa] (BMH);Pressure normal height [hPa] (BMH);Pressure2 [hPa] (BMH);Pressure normal height 2 [hPa] (BMH);Temp [°C] (BMH);Temp2 [°C] (BMH);Light [Lx] (VEML);Light2 [Lx] (VEML);Errors (SHT);Errors (BMH);Errors (VEML)\r\n"
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
	return os.WriteFile(file, []byte(fmt.Sprintf(DATA_HEADERLINE, cfg.WEATHER_STATION_VERSION)), 0o644)
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

	line := fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d;%.1f;%.1f;%.1f;%.1f;%.1f;%.1f;%.1f;%.1f;%.1f;%.1f;%.1f;%.1f;%.1f;%.1f;%d;%d;%d\r\n",
		now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(),
		data.T1_1, data.T1_2, data.H_1, data.H_2, data.Dp_1, data.Dp_2, data.P_1, data.Pn_1, data.P_2, data.Pn_2,
		data.T2_1, data.T2_2, data.L_1, data.L_2, data.E_1, data.E_2, data.E_3)

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
	Ti_y  int `json:"ti_y"`
	Ti_m  int `json:"ti_m"`
	Ti_d  int `json:"ti_d"`
	Ti_h  int `json:"ti_h"`
	Ti_mi int `json:"ti_mi"`
	Ti_s  int `json:"ti_s"`
	T     int `json:"t"`
	T2    int `json:"t2"`
	H     int `json:"h"`
	H2    int `json:"h2"`
	DP    int `json:"d"`
	DP2   int `json:"d2"`
	P     int `json:"p"`
	P2    int `json:"p2"`
	Pn    int `json:"pn"`
	Pn2   int `json:"pn2"`
	L     int `json:"l"`
	L2    int `json:"i2"`
}

type csvData struct {
	CsvConfig csvConfig  `json:"config"`
	Entries   []csvEntry `json:"data"`
}

func (sd *SdData) GetFilePath(t time.Time) string {
	path := filepath.Join(sd.folder, fmt.Sprintf("%04d", t.Year()))
	path = filepath.Join(path, fmt.Sprintf("%02d", t.Month()))
	path = filepath.Join(path, fmt.Sprintf("%04d-%02d-%04d.csv", t.Year(), t.Month(), t.Day()))
	return path
}

func (sd *SdData) GetStartFile(t, endTime time.Time) (string, error) {
	path := sd.GetFilePath(t)
	ok, err := helper.ExistFile(path)
	if err != nil {
		return "", err
	}
	if ok {
		return path, nil
	} else {
		for {
			t = t.AddDate(0, 0, 1)
			if !t.Before(endTime) {
				return "", fmt.Errorf("No data")
			}
			path = sd.GetFilePath(t)
			ok, err = helper.ExistFile(path)
			if err != nil {
				return "", err
			}
			if ok {
				return path, nil
			}
		}
	}
}

func (sd *SdData) GetEndFile(t, startTime time.Time) (string, error) {
	path := sd.GetFilePath(t)
	ok, err := helper.ExistFile(path)
	if err != nil {
		return "", err
	}
	if ok {
		return path, nil
	} else {
		for {
			t = t.AddDate(0, 0, -1)
			if !t.After(startTime) {
				return "", fmt.Errorf("No data")
			}
			path = sd.GetFilePath(t)
			ok, err = helper.ExistFile(path)
			if err != nil {
				return "", err
			}
			if ok {
				return path, nil
			}
		}
	}
}

func (sd *SdData) GetData(startTime, endTime time.Time) ([]byte, error) {
	data := csvData{
		CsvConfig: csvConfig{
			Interval: sd.config.GetIntValue(cfg.SECTION_DATA, cfg.KEY_DATA_INTERVAL, cfg.DEF_DATA_INTERVAL),
		},
		Entries: make([]csvEntry, 0, 500),
	}
	startFile, err := sd.GetStartFile(startTime, endTime)
	if err != nil {
		return nil, err
	}
	
	endFile, err := sd.GetEndFile(startTime, endTime)
	if err != nil {
		return nil, err
	}
	startTime, err = time.Parse("2006-01-02_15-04-05", startFile)
	

	d, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return d, nil
}
