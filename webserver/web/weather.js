/*!
 * Copyright (c) 2026 Reiner Pröls
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 *
 * SPDX-License-Identifier: MIT
 *
 * Author: Reiner Pröls
 *
 * Weather station - main script
*/

const version = "v3.3";

let updateDataTimer = null;
let updateDataTimerGraph = null;
let updateIFrameTimer = null;

const debug = 0;

let updateInterval = 5000;
let updateIntervalGraph = 30000;
let updateIntervalIFrame = 30*60*1000;

let data_request = null;
let data_request_graph = null;
let file_request = null;

const pressure_trend_colors = [];
const graphs = [];

const graph_p = new SVGGraph("p_", 0, 25, 940, 980, 24, 980, 1, 4, 1, 5, 70, 40, 20, 20);
graphs.push(graph_p);

const graph_p2 = new SVGGraph("p2_", 0, 25, 940, 980, 24, 980, 1, 4, 1, 5, 70, 40, 20, 20);
graphs.push(graph_p2);

// prefix, minXScale, maxXScale, minYScale, maxYScale, maxXValue, maxYValue, stepX, stepX_bold, stepY, stepY_bold, leftMargin, bottomMargin, rightMargin, topMargin)
const graph_pt3 = new SVGGraph("pt3_", 0, 25, -2.0, 2.0, 24, 2.0, 1, 4, 0.1, 0.5, 70, 40, 20, 20);
graphs.push(graph_pt3);

const graph_t = new SVGGraph("t_", 0, 25, 0, 30, 24, 28, 1, 4, 1, 5, 70, 40, 20, 20);
graphs.push(graph_t);

const graph_t2 = new SVGGraph("t2_", 0, 25, 0, 30, 24, 28, 1, 4, 1, 5, 70, 40, 20, 20);
graphs.push(graph_t2);

const graph_h = new SVGGraph("h_", 0, 25, 0, 102, 24, 100, 1, 4, 2, 10, 70, 40, 20, 20);
graphs.push(graph_h);

const graph_h2 = new SVGGraph("h2_", 0, 25, 0, 102, 24, 100, 1, 4, 2, 10, 70, 40, 20, 20);
graphs.push(graph_h2);

const graph_d = new SVGGraph("d_", 0, 25, 0, 20, 24, 19, 1, 4, 0.25, 1, 70, 40, 20, 20);
graphs.push(graph_d);

const graph_d2 = new SVGGraph("d2_", 0, 25, 0, 20, 24, 19, 1, 4, 0.25, 1, 70, 40, 20, 20);
graphs.push(graph_d2);

const graph_wb = new SVGGraph("wb_", 0, 25, 0, 20, 24, 19, 1, 4, 0.25, 1, 70, 40, 20, 20);
graphs.push(graph_wb);

const graph_wb2 = new SVGGraph("wb2_", 0, 25, 0, 20, 24, 19, 1, 4, 0.25, 1, 70, 40, 20, 20);
graphs.push(graph_wb2);

const graph_l = new SVGGraph("l_", 0, 25, 0, 1000, 24, 980, 1, 4, 50, 250, 70, 40, 20, 20);
graphs.push(graph_l);

const graph_l2 = new SVGGraph("l2_", 0, 25, 0, 1000, 24, 980, 1, 4, 50, 250, 70, 40, 20, 20);
graphs.push(graph_l2);

function getXAxisText(str, param)
{
	let h = parseInt(str);
	if (h >= 0)
	{
		return (param ==1 ? h%24 + ":0": pad(h%24, 2) + ":00");
	}
	else
	{
		while (h < 0)
			h += 24;
		return (param == 1 ? (h != 0 ? "-" : "") + h + ":0" : (h != 0 ? "- " : "") + pad(h, 2) + ":00");
	}
}

function doLoad()
{
	const date = new Date();
	const value = date.getFullYear() + "-" + pad(date.getMonth()+1, 2) + "-" + pad(date.getDate(), 2);
	const tmpStart = document.getElementById("startdate");
	if (tmpStart)
		tmpStart.value = value;
	const tmpEnd = document.getElementById("enddate");
	if (tmpEnd)
		tmpEnd.value = value;

	for (let i=0; i<5; i++){
		pressure_trend_colors[i] = getCustomCssProperty(`--weather-pressure_trend_${i}`)
	}

	doStart();
	doPeriodic();
	doLoadFiles("");

	doLoadGraph();
	
	if ((tmp = document.getElementById("version")))
		tmp.textContent = version;
	
	onClickActualData();
}

function doLoadGraph()
{
	window.addEventListener("resize", function(event)
		{
			doPeriodicGraph();
		}
	);

	graph_p.yScaleText = "Pressure [hPa]";
	graph_p2.yScaleText = "Pressure (long) [hPa]";
	graph_pt3.yScaleText = "Pressure Trend 3h [hPa/h]";
	graph_t.yScaleText = "Temperature [°C]";
	graph_t2.yScaleText = "Temperature (long) [°C]";
	graph_h.yScaleText = "Humidity [% rel]";
	graph_h2.yScaleText = "Humidity (long) [% rel]";
	graph_d.yScaleText = "Dew point [°C]";
	graph_d2.yScaleText = "Dew point (long) [°C]";
	graph_wb.yScaleText = "Wet bulb [°C]";
	graph_wb2.yScaleText = "Wet bulb (long) [°C]";
	graph_l.yScaleText = "Luminosity [lux]";
	graph_l2.yScaleText = "Luminosity (long) [lux]";
	
	graphs.forEach(function(graph)
	{
		graph.xScaleText = "Time [hours]";
		graph.getXAxisText = getXAxisText;
		graph.hasBullets = false;
		graph.doInit();
	});
}

function doRestart()
{
	var restart_request = new XMLHttpRequest();
	restart_request.open("GET","restart", true);
	noCacheInRequest(restart_request);
	restart_request.responseType = "text";
	restart_request.onload = function()
	{
		document.body.style.cursor = "default";
		window.location.href = "restart.html"
	}
	document.body.style.cursor = "wait";
	restart_request.send();
}

function doCfg()
{
	window.location.href = "config.html";
}

function doStart()
{
	if (!updateDataTimer)
		updateDataTimer = setInterval(function(){doPeriodic()}, updateInterval);
	if (!updateIFrameTimer)
		updateIFrameTimer = setInterval(function(){doUpdateIFrames()}, updateIntervalIFrame);
}

function doStartGraph()
{
	// log("doStartGraph");
	if (!updateDataTimerGraph)
		updateDataTimerGraph = setInterval(function(){doPeriodicGraph()}, updateIntervalGraph);
}


function onClickActualData()
{
	let ctrl, tmp;
	if ((ctrl = document.getElementById("actualdate")))
	{
		if ((tmp = document.getElementById("startdate")))
			tmp.disabled = ctrl.checked;
		if ((tmp = document.getElementById("enddate")))
			tmp.disabled = ctrl.checked;
		if (ctrl.checked)
			doStartGraph();
		else
		{
			if (updateDataTimer)
			{
				clearInterval(updateDataTimer);
				updateDataTimerGraph = null;
			}
		}
			
		doPeriodicGraph();
		log("onClickActualData()");
	}
}

function onStartDate()
{
	log("onClickStartDate()");
	doPeriodicGraph();
}

function onEndDate()
{
	log("onClickEndDate()");
	doPeriodicGraph();
}

function kB2Str(kbSize)
{
	if (kbSize < 1024)
		return kbSize + " kB";
	
	else if (kbSize < 1024*1024)
		return (kbSize/1024).toFixed(2) + " MB";
	
	else if (kbSize < 1024*1024*1024)
		return (kbSize/(1024*1024)).toFixed(2) + " GB";

	else if (kbSize < 1024*1024*1024*1024)
		return (kbSize/(1024*1024*1024)).toFixed(2) + " TB";
	else if (kbSize < 1024*1024*1024*1024*1024)
		return (kbSize/(1024*1024*1024*1024)).toFixed(2) + " PB";
	else if (kbSize < 1024*1024*1024*1024*1024*1024)
		return (kbSize/(1024*1024*1024*1024*1023)).toFixed(2) + " EB";
	else
		return "???";
}

function doPeriodic()
{
	if (data_request == null)
	{
		data_request = new XMLHttpRequest();
		data_request.open("GET","update", true);
		noCacheInRequest(data_request);
		data_request.responseType = "text";
		data_request.onerror = function()
		{
			log("Error in data request");
			data_request = null;
		}
		data_request.onabort = function()
		{
			log("Abort data request");
			data_request = null;
		}

		data_request.onload = function()
		{
			if (data_request.response)
			{
				const json = JSON.parse(data_request.response);
				// log(data_request.response);
				if (json.config.update_interval !== undefined)
				{
					const interval = parseInt(json.config.update_interval);
					if (interval != updateInterval)
					{
						updateInterval = interval;
						clearInterval(updateDataTimer);
						updateDataTimer = setInterval(function(){doPeriodic()}, updateInterval);
					}
				}

				if (json.status.time !== undefined)
				{
					const wday = ["So", "Mo", "Di", "Mi", "Do", "Fr", "Sa"];
					const date_str = wday[json.status.time.wday] + ", " + (json.status.time.day < 10 ? "0" : "") + json.status.time.day + "." + (json.status.time.month < 10 ? "0" : "") + json.status.time.month + "." + json.status.time.year;
					const time_str = (json.status.time.hour < 10 ? "0" : "") + json.status.time.hour + ":" + (json.status.time.min < 10 ? "0" : "") + json.status.time.min + ":" + (json.status.time.sec < 10 ? "0" : "") + json.status.time.sec;
					if ((tmp = document.getElementById("status_date")))
						tmp.textContent = date_str;
					if ((tmp = document.getElementById("sysstat_date")))
						tmp.textContent = date_str;
					if ((tmp = document.getElementById("status_time")))
						tmp.textContent = time_str;
					if ((tmp = document.getElementById("sysstat_time")))
						tmp.textContent = time_str;
				}
				else
				{
					if ((tmp = document.getElementById("status_tr_date")))
						tmp.style.display = 'none';
					if ((tmp = document.getElementById("status_tr_time")))
						tmp.style.display = 'none';
				}
				
				if (json.status.version !== undefined)
				{
					if ((tmp = document.getElementById("summary_version")))
						tmp.textContent = json.status.version;
				}
				else
				{
					if ((tmp = document.getElementById("summary_tr_version")))
						tmp.style.display = 'none';
				}
				
				if (json.status.uptime !== undefined)
				{
					if ((tmp = document.getElementById("sysstat_uptime")))
						tmp.textContent = json.status.uptime;
				}
				else
				{
					if ((tmp = document.getElementById("sysstat_tr_uptime")))
						tmp.style.display = 'none';
				}
				
				if (tmp = document.getElementById("summary_temperature"))
					tmp.textContent = json.sensors[1].temperature.toFixed(1) + " (" + json.sensors[0].temperature.toFixed(1) + ") °C";
				if (tmp = document.getElementById("summary_temperature2"))
					tmp.textContent = json.sensors[1].temperature2.toFixed(1) + " (" + json.sensors[0].temperature2.toFixed(1) + ") °C";
				if (tmp = document.getElementById("summary_humidity"))
					tmp.textContent = json.sensors[1].humidity.toFixed(1) + " % rel";
				if (tmp = document.getElementById("summary_humidity2"))
					tmp.textContent = json.sensors[1].humidity2.toFixed(1) + " % rel";
				if (tmp = document.getElementById("summary_dewpoint"))
					tmp.textContent = json.sensors[1].dewpoint.toFixed(1) + " °C";
				if (tmp = document.getElementById("summary_dewpoint2"))
					tmp.textContent = json.sensors[1].dewpoint2.toFixed(1) + " °C";
				if (tmp = document.getElementById("summary_pressure"))
					tmp.textContent = json.sensors[0].pressure.toFixed(1) + " / " + json.sensors[0].pressure_normal.toFixed(1) + " hPa";
				if (tmp = document.getElementById("summary_pressure2"))
					tmp.textContent = json.sensors[0].pressure2.toFixed(1) + " / " + json.sensors[0].pressure2_normal.toFixed(1) + " hPa";
				if (tmp = document.getElementById("summary_pressure_trend_31")) {
					const val = json.status.pressure_trend_3.toFixed(2);
					tmp.textContent = val + " / " + json.status.pressure_trend_1.toFixed(2) + " hPa/h";
					const diff = Math.abs(val);
					let colIndex = 0;
					if (diff < 0.5)
						colIndex = 0;
					else if (diff < 1.0)
						colIndex = 1;
					else if (diff < 1.5)
						colIndex = 2;
					else if (diff < 2.0)
						colIndex = 3;
					else
						colIndex = 4;
					tmp.style.backgroundColor = pressure_trend_colors[Math.min(colIndex, pressure_trend_colors.length - 1)];
				}
				if (tmp = document.getElementById("summary_wetbulb"))
					tmp.textContent = json.status.wet_bulb.toFixed(1) + " °C";
				if (tmp = document.getElementById("summary_wetbulb2"))
					tmp.textContent = json.status.wet_bulb2.toFixed(1) + " °C";
				if (tmp = document.getElementById("summary_lux"))
					tmp.textContent = json.sensors[2].lux.toFixed(1) + " lx";
				if (tmp = document.getElementById("summary_lux2"))
					tmp.textContent = json.sensors[2].lux2.toFixed(1) + " lx";
				if (tmp = document.getElementById("summary_errors"))
					tmp.textContent = json.status.errors + " / " + json.status.resets;
				if (tmp = document.getElementById("summary_sensors"))
					tmp.textContent = json.sensors.map(sensor => sensor.name).join(", ").toLowerCase();
				
				if (json.status.raspi !== undefined)
				{
					if (tmp = document.getElementById("sysstat_fan"))
						tmp.textContent = json.status.raspi >= 0 ? json.status.raspi.fan + " rpm" : "---";
					if (tmp = document.getElementById("sysstat_cpu"))
						tmp.textContent = (json.status.raspi.cpu/10.0).toFixed(1) + " %";
					if (tmp = document.getElementById("sysstat_cpu_freq"))
						tmp.textContent = (json.status.raspi.cpu_freq/1000.0).toFixed(2) + " GHz";
					if (tmp = document.getElementById("sysstat_temperature"))
						tmp.textContent = (json.status.raspi.temp/10.0).toFixed(1) + " °C";
					if (tmp = document.getElementById("sysstat_5volt"))
						tmp.textContent = (json.status.raspi.voltage_5/1000.0).toFixed(2) + " V";
					if (tmp = document.getElementById("sysstat_3volt"))
						tmp.textContent = (json.status.raspi.voltage_3/1000.0).toFixed(2) + " V";
					if (tmp = document.getElementById("sysstat_max_current"))
						tmp.textContent = (json.status.raspi.max_current/1000.0).toFixed(1) + " A";
					if (tmp = document.getElementById("sysstat_serial"))
						tmp.textContent = json.status.raspi.serial;
					if (tmp = document.getElementById("sysstat_throttle"))
						tmp.textContent = "0x" + (json.status.raspi.throttle.toString(16).padStart(8, "0"));
					if (tmp = document.getElementById("sysstat_version"))
					{
						const major = (json.status.raspi.version >> 8) & 0xFF;
						const minor = json.status.raspi.version & 0xFF;
						tmp.textContent = major + "." + minor;
					}
					if (tmp = document.getElementById("sysstat_ram"))
					{
						tmp.textContent = kB2Str(json.status.raspi.ram_u) + " / " + kB2Str(json.status.raspi.ram_t);
					}
				}

				if (json.status.sd !== undefined)
				{
					if ((tmp = document.getElementById("sysstat_sd")))
						tmp.textContent = kB2Str(json.status.sd.free) + " / " + kB2Str(json.status.sd.total);
				}
				else
				{
					if ((tmp = document.getElementById("sysstat_tr_sd")))
						tmp.style.display = 'none';
				}
				
				if (json.status.raspi.wifi_strength !== undefined && json.status.raspi.wifi_rx !== undefined && json.status.raspi.wifi_tx !== undefined)
				{
					if ((tmp = document.getElementById("sysstat_wifi")))
						tmp.textContent = json.status.raspi.wifi_strength + " dBm / " + kB2Str(json.status.raspi.wifi_rx + json.status.raspi.wifi_tx);
				}
				else
				{
					if ((tmp = document.getElementById("sysstat_tr_wifi")))
						tmp.style.display = 'none';
				}

				/*
				if ((tmp = document.getElementById("summary_inttemp")))
					tmp.textContent = json.sensors[json.sensors.length - 1].temperature + " °C";
				*/
				
			} // response != NULL

			data_request = null;
		}
		data_request.send();
	}
}

function pad(num, size)
{
    let s = num + "";
    while (s.length < size)
		s = "0" + s;
    return s;
}

function doPeriodicGraph()
{
	if (data_request_graph == null)
	{
		data_request_graph = new XMLHttpRequest();
		const now = new Date();
		let timeRaster = 4;
		let timeRaster_fine = 1;
		let userParam = 0;
		let yScaleTextOffset = "1.4em";
		if (tmp = document.getElementById("p_svg"))
		{
			const svgWidth = tmp.getBoundingClientRect().width;
			// log("SVG-Size: " + svgWidth);
			if (svgWidth < 350)
			{
				timeRaster = 8;
				timeRaster_fine = 2;
				userParam = 1;
				yScaleTextOffset = "0.9em"
			}
			else if (svgWidth < 400)
			{
				timeRaster = 8;
				timeRaster_fine = 2;
				yScaleTextOffset = "0.9em"
			}
			else if (svgWidth < 590)
			{
				timeRaster = 6;
				yScaleTextOffset = "1.0em"
			}
		}

		graphs.forEach(function(graph)
		{
			graph.userParam = userParam;
			graph.yScaleTextOffset = yScaleTextOffset;
		});
		
		let useActualData = true;
		let ctrl, startDateCtrl = null, endDateCtrl = null;
		let startDateArray, endDateArray;
		if ((ctrl = document.getElementById("actualdate")))
		{
			startDateCtrl = document.getElementById("startdate");
			endDateCtrl = document.getElementById("enddate");
			if (startDateCtrl && endDateCtrl)
				useActualData = ctrl.checked;
		}
		
		if (useActualData)
		{
			now.setHours(now.getHours() - 36);
			data_request_graph.open("GET", "getdata?start=" + now.getFullYear() + "-" + pad(now.getMonth() + 1, 2) + "-" + pad(now.getDate(), 2) + "_" + pad(now.getHours(), 2) + "-00-00");
		}
		else
		{
			startDateArray = startDateCtrl.value.split("-");
			endDateArray = endDateCtrl.value.split("-");
			data_request_graph.open("GET", "getdata?start=" + startDateArray[0] + "-" + startDateArray[1] + "-" + startDateArray[2] + "_00-00-00&end=" + endDateArray[0] + "-" + endDateArray[1] + "-" + endDateArray[2] + "_23-59-59");
		}
		
		noCacheInRequest(data_request_graph);
		data_request_graph.responseType = "text";
		data_request_graph.onerror = function()
		{
			log("Error in data request _graph");
			data_request_graph = null;
		}
		data_request_graph.onabort = function()
		{
			log("Abort data request");
			data_request_graph = null;
		}

		data_request_graph.onload = function()
		{
			if (data_request_graph.response)
			{
				var json = JSON.parse(data_request_graph.response);
				// log(data_request_graph.response);
				if (updateDataTimerGraph)
				{
					if (json.config.sd_interval !== undefined)
					{
						const interval = parseInt(json.config.sd_interval);
						if (interval != updateIntervalGraph)
						{
							updateIntervalGraph = interval;
							clearInterval(updateDataTimerGraph);
							updateDataTimerGraph = setInterval(function(){doPeriodicGraph()}, updateIntervalGraph);
						}
					}
				}
				if (json.data.length > 1)
				{
					var startDate, endDate, tmpDate;
					
					if (useActualData)
					{
						startDate = new Date(json.data[0].ti_y, json.data[0].ti_m - 1, json.data[0].ti_d, json.data[0].ti_h, 0, 0);
						endDate = new Date(json.data[json.data.length-1].ti_y, json.data[json.data.length-1].ti_m - 1, json.data[json.data.length-1].ti_d, json.data[json.data.length-1].ti_h, 0, 0);
						tmpDate = new Date(json.data[json.data.length-1].ti_y, json.data[json.data.length-1].ti_m - 1, json.data[json.data.length-1].ti_d, 0, 0, 0);
					}
					else
					{
						startDate = new Date(startDateArray[0], startDateArray[1] - 1, startDateArray[2], 0, 0, 0);
						endDate = new Date(endDateArray[0], endDateArray[1] - 1, endDateArray[2], 0, 0, 0);
						tmpDate = endDate;
					
						const diff = (endDate - startDate) / 3600000;
						if (diff > 36)
						{
							timeRaster = (parseInt(parseInt(diff / 10) / 4) + 1)*4;
							if (timeRaster == 16 || timeRaster == 20)
								timeRaster = 24;
							if (timeRaster > 24)
								timeRaster = parseInt(timeRaster/24)*24;
							if (timeRaster >= 48)
								timeRaster_fine = parseInt(timeRaster/8);
							else
								timeRaster_fine = parseInt(timeRaster/4);
						}
					}
					
					graphs.forEach(function(graph)
					{
						graph.minXScale = (tmpDate - startDate) / 3600000;
							
						if (parseInt(graph.minXScale/timeRaster)*timeRaster == parseInt(graph.minXScale))
							graph.minXScale = -(parseInt(graph.minXScale/timeRaster))*timeRaster;
						else
							graph.minXScale = -(parseInt(graph.minXScale/timeRaster) + 1)*timeRaster;
						
						graph.maxXScale = (useActualData ? json.data[json.data.length-1].ti_h : 24);
						graph.maxXScale = (parseInt(graph.maxXScale/timeRaster) + (useActualData ? 1 : 0))*timeRaster + 2*timeRaster_fine;
						graph.stepX_bold = timeRaster;
						graph.stepX = timeRaster_fine;
						
						graph.maxXValue = graph.maxXScale - timeRaster_fine;
						
//						log("minXScale=" + graph.minXScale + " maxXScale=" + graph.maxXScale);
					});
					
					// log("Start time: " + startDate);
					// log("End time: " + endDate);
					// log("Duration: " + duration);
					graphs.forEach(function(graph)
					{
						const xArray = [];
						const yArray = [];
						let minY = null;
						let maxY = null;
						json.data.forEach(function(item)
						{
							// log(item);
							const date = new Date(item.ti_y, item.ti_m - 1, item.ti_d, item.ti_h, item.ti_mi, item.ti_s);
							const diff = (date - tmpDate)  / 3600000.0;
							
							let y = null;
							if (graph == graph_p)
							{
								y = parseFloat(item.pn);
								if (y < 960)
									y = null;
							}
							else if (graph == graph_p2)
							{
								y = parseFloat(item.pn2);
								if (y < 960)
									y = null;
							}
							else if (graph == graph_pt3)
							{
								if (item.pt3 !== undefined)
									y = parseFloat(item.pt3);
							}
							else if (graph == graph_t)
							{
								y = parseFloat(item.t);
							}
							else if (graph == graph_t2)
							{
								if (item.t2 !== undefined)
									y = parseFloat(item.t2);
							}
							else if (graph == graph_h)
							{
								y = parseFloat(item.h);
								if (y < 5)
									y = null;
							}
							else if (graph == graph_h2)
							{
								if (item.h2 !== undefined)
								{
									y = parseFloat(item.h2);
									if (y < 5)
										y = null;
								}
							}
							else if (graph == graph_d)
								y = parseFloat(item.d);
							else if (graph == graph_d2)
							{
								if (item.d2 !== undefined)
									y = parseFloat(item.d2);
							}
							else if (graph == graph_wb)
								y = parseFloat(item.wb);
							else if (graph == graph_wb2)
							{
								if (item.wb2 !== undefined)
									y = parseFloat(item.wb2);
							}
							else if (graph == graph_l)
								y = parseFloat(item.l);
							else if (graph == graph_l2)
							{
								if (item.l2 !== undefined)
									y = parseFloat(item.l2);
							}
							
							if (y !== null)
							{
								xArray.push(diff);
								yArray.push(y);
							
								if (minY === null || minY > y)
									minY = y;
								if (maxY === null || maxY < y)
									maxY = y;
							}
						});
						if (graph == graph_l && maxY !== null && maxY > 3000)
							maxY = 3000;
						
						if (graph == graph_l2 && maxY !== null)
						{
							if (maxY > 6000)
							{
								graph.stepY_bold = 1000;
								graph.stepY = 250;
							}
							else if (maxY > 3000)
							{
								graph.stepY_bold = 500;
								graph.stepY = 100;
							}
							else
							{
								graph.stepY_bold = 250;
								graph.stepY = 50;
							}
						}
						/*
						if (graph == graph_pt3)
						{
							maxY = 2.0;
							minY = -2.0;
						}
							*/
						// log("Graph=" + graph + " MinY=" + minY + " MaxY=" + maxY); 
						graph.maxYScale = (parseInt(maxY / graph.stepY_bold) + 1) * graph.stepY_bold + 2.0 * graph.stepY;
						graph.minYScale = parseInt(minY / graph.stepY_bold) * graph.stepY_bold;
						if (minY < 0)
							graph.minYScale -= graph.stepY_bold;
						graph.maxYValue = graph.maxYScale - graph.stepY;
						
						graph.doInit();
						
						graph.addData(xArray, yArray);
					});
				}
				
			} // response != NULL

			data_request_graph = null;
		}
		data_request_graph.send();
	}
}

function doLoadFiles(start_dir)
{
	if (file_request == null)
	{
		file_request = new XMLHttpRequest();
		file_request.open("GET","listsd?dir=" + start_dir, true);
		noCacheInRequest(file_request);
		file_request.responseType = "text";
		file_request.onerror = function()
		{
			log("Error in file request");
			file_request = null;
		}
		file_request.onabort = function()
		{
			log("Abort file request");
			file_request = null;
		}

		file_request.onload = function()
		{
			if (file_request.response)
			{
				const json = JSON.parse(file_request.response);
				log(file_request.response);
				if (tmp = document.getElementById("files"))
				{
					while (tmp.firstChild)
						tmp.removeChild(tmp.firstChild);
					for (let index=0; index < json.dirs.length; index++)
					{
						const item = document.createElement("a");
						item.setAttribute("class", "dir");
						// item.setAttribute("href", "");
						item.setAttribute("onclick", "doLoadFiles(\"" + json.path + "/" + json.dirs[index].name + "\")");
						item.innerHTML = json.dirs[index].name;
						tmp.appendChild(item);
						tmp.appendChild(document.createElement("br"));
					}
					for (let index=0; index < json.files.length; index++)
					{
						const item = document.createElement("a");
						item.setAttribute("class", "file");
						item.setAttribute("href", "getsd?file=" + json.path + "/" + json.files[index].name);
						if (!json.files[index].name.toLowerCase().endsWith(".png"))
    						item.setAttribute("download", json.files[index].name);						
						item.innerHTML = json.files[index].name;
						tmp.appendChild(item);
						tmp.appendChild(document.createElement("br"));
					}
				}
			} // response != NULL

			file_request = null;
		}
		file_request.send();
	}
}

function doUpdate()
{
	const tmp = document.getElementById("update_file_upload");
	if (tmp)
	{
		const request = new XMLHttpRequest();
		request.open("POST", "firmwareupdate", true);
		noCacheInRequest(request);
		request.responseType = "text";
		request.onload = function()
		{
			document.body.style.cursor = "default";
			window.location.href = "update.html"
		}
			
		const data = new FormData();
		data.append("firmwarefile", tmp.files[0])
		document.body.style.cursor = "wait";
		request.send(data);
	}
	return false;
}

function noCacheInRequest(request)
{
	request.setRequestHeader("Cache-Control", "no-cache")
	request.setRequestHeader("Pragma", "no-cache")
}

function log(msg)
{
	if (debug && window.console)
		console.log(msg);
}

function doUpdateIFrames()
{
	const iframe = document.querySelector('.weather_iframe');
		if (iframe) {
		iframe.src = iframe.src;
	}
	iframe = document.querySelector('.weather_iframe_last');
		if (iframe) {
			iframe.src = iframe.src;
	}
}

function getCustomCssProperty(name) {
	return getComputedStyle(document.documentElement).getPropertyValue(name).trim();
}
