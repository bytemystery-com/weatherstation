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

package sd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"weatherd/cfg"
	"weatherd/helper"
)

const (
	// DT;T;T2;H;H2;DP;DP2;P;PN;P2;PN2;T;T2;L;L2;ESHT;EBMH;EVEML
	DATA_HEADERLINE = "# Date Time;Temp [°C] (SHT);Temp2 [°C] (SHT);Humidity [rel %] (SHT);Humidity2 [rel %] (SHT);Dew point [°C] (SHT);Dew point2 [°C] (SHT);Pressure [hPa] (BMH);Pressure normal height [hPa] (BMH);Pressure2 [hPa] (BMH);Pressure normal height 2 [hPa] (BMH);Temp [°C] (BMH);Temp2 [°C] (BMH);Light [Lx] (VEML);Light2 [Lx] (VEML);Errors (SHT);Errors (BMH);Errors (VEML)\r\n"
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
	return os.WriteFile(file, []byte(DATA_HEADERLINE), 0o644)
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
