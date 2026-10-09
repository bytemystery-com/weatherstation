const version = "v3.1";

var updateDataTimer = null;
var updateDataTimerGraph = null;

var debug = 0;

var updateInterval = 5000;
var updateIntervalGraph = 30000;

var data_request = null;
var data_request_graph = null;
var file_request = null;

var graphs = new Array();

var graph_p = new SVGGraph("p_", 0, 25, 940, 980, 24, 980, 1, 4, 1, 5, 70, 40, 20, 20);
graphs.push(graph_p);

var graph_p2 = new SVGGraph("p2_", 0, 25, 940, 980, 24, 980, 1, 4, 1, 5, 70, 40, 20, 20);
graphs.push(graph_p2);

var graph_t = new SVGGraph("t_", 0, 25, 0, 30, 24, 28, 1, 4, 1, 5, 70, 40, 20, 20);
graphs.push(graph_t);

var graph_t2 = new SVGGraph("t2_", 0, 25, 0, 30, 24, 28, 1, 4, 1, 5, 70, 40, 20, 20);
graphs.push(graph_t2);

var graph_h = new SVGGraph("h_", 0, 25, 0, 102, 24, 100, 1, 4, 2, 10, 70, 40, 20, 20);
graphs.push(graph_h);

var graph_h2 = new SVGGraph("h2_", 0, 25, 0, 102, 24, 100, 1, 4, 2, 10, 70, 40, 20, 20);
graphs.push(graph_h2);

var graph_d = new SVGGraph("d_", 0, 25, 0, 20, 24, 19, 1, 4, 0.25, 1, 70, 40, 20, 20);
graphs.push(graph_d);

var graph_d2 = new SVGGraph("d2_", 0, 25, 0, 20, 24, 19, 1, 4, 0.25, 1, 70, 40, 20, 20);
graphs.push(graph_d2);

var graph_l = new SVGGraph("l_", 0, 25, 0, 1000, 24, 980, 1, 4, 50, 250, 70, 40, 20, 20);
graphs.push(graph_l);

var graph_l2 = new SVGGraph("l2_", 0, 25, 0, 1000, 24, 980, 1, 4, 50, 250, 70, 40, 20, 20);
graphs.push(graph_l2);

function getXAxisText(str, param)
{
	var h = parseInt(str);
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
	var date = new Date();
	var value = date.getFullYear() + "-" + pad(date.getMonth()+1, 2) + "-" + pad(date.getDate(), 2);
	var tmp = document.getElementById("startdate");
	if (tmp)
		tmp.value = value;
	tmp = document.getElementById("enddate");
	if (tmp)
		tmp.value = value;
	
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
	graph_t.yScaleText = "Temperature [°C]";
	graph_t2.yScaleText = "Temperature (long) [°C]";
	graph_h.yScaleText = "Humidity [% rel]";
	graph_h2.yScaleText = "Humidity (long) [% rel]";
	graph_d.yScaleText = "Dew point [°C]";
	graph_d2.yScaleText = "Dew point (long) [°C]";
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
}

function doStartGraph()
{
	// log("doStartGraph");
	if (!updateDataTimerGraph)
		updateDataTimerGraph = setInterval(function(){doPeriodicGraph()}, updateIntervalGraph);
}

function onClickActualData()
{
	var ctrl, tmp;
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
		return (kbSize/1024).toFixed(1) + " MB";
	
	else if (kbSize < 1024*1024*1024)
		return (kbSize/(1024*1024)).toFixed(2) + " GB";

	else if (kbSize < 1024*1024*1024*1024)
		return (kbSize/(1024*1024*1024)).toFixed(2) + " TB";
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
				var json = JSON.parse(data_request.response);
				// log(data_request.response);
				if (json.config.update_interval !== undefined)
				{
					var interval = parseInt(json.config.update_interval);
					if (interval != updateInterval)
					{
						updateInterval = interval;
						clearInterval(updateDataTimer);
						updateDataTimer = setInterval(function(){doPeriodic()}, updateInterval);
					}
				}

				if (json.status.time !== undefined)
				{
					var wday = ["So", "Mo", "Di", "Mi", "Do", "Fr", "Sa"];
					var date_str = wday[json.status.time.wday] + ", " + (json.status.time.day < 10 ? "0" : "") + json.status.time.day + "." + (json.status.time.month < 10 ? "0" : "") + json.status.time.month + "." + json.status.time.year;
					var time_str = (json.status.time.hour < 10 ? "0" : "") + json.status.time.hour + ":" + (json.status.time.min < 10 ? "0" : "") + json.status.time.min + ":" + (json.status.time.sec < 10 ? "0" : "") + json.status.time.sec;
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
				if (tmp = document.getElementById("summary_pressure_trend"))
					tmp.textContent = json.status.pressure_trend_3.toFixed(2) + " / " + json.status.pressure_trend_1.toFixed(2) + " hPa/h";
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
						tmp.textContent = json.status.raspi.fan + " rpm";
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
    var s = num + "";
    while (s.length < size)
		s = "0" + s;
    return s;
}

function doPeriodicGraph()
{
	if (data_request_graph == null)
	{
		data_request_graph = new XMLHttpRequest();
		var now = new Date();
		var svgWidth = 0;
		var timeRaster = 4;
		var timeRaster_fine = 1;
		var userParam = 0;
		var yScaleTextOffset = "1.4em";
		if (tmp = document.getElementById("p_svg"))
		{
			svgWidth = tmp.getBoundingClientRect().width;
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
		
		var useActualData = true;
		var ctrl, startDateCtrl = null, endDateCtrl = null;
		var startDateArray, endDateArray;
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
						var interval = parseInt(json.config.sd_interval);
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
					
						var diff = (endDate - startDate) / 3600000;
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
						var xArray = new Array();
						var yArray = new Array();
						var minY = null;
						var maxY = null;
						json.data.forEach(function(item)
						{
							// log(item);
							var date = new Date(item.ti_y, item.ti_m - 1, item.ti_d, item.ti_h, item.ti_mi, item.ti_s);
							var diff = (date - tmpDate)  / 3600000.0;
							
							var y = null;
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
							else if (graph == graph_t)
								y = parseFloat(item.t);
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
				var json = JSON.parse(file_request.response);
				log(file_request.response);
				if (tmp = document.getElementById("files"))
				{
					while (tmp.firstChild)
						tmp.removeChild(tmp.firstChild);
					var index;
					var item;
					for (index=0; index < json.dirs.length; index++)
					{
						item = document.createElement("a");
						item.setAttribute("class", "dir");
						// item.setAttribute("href", "");
						item.setAttribute("onclick", "doLoadFiles(\"" + json.path + "/" + json.dirs[index].name + "\")");
						item.innerHTML = json.dirs[index].name;
						tmp.appendChild(item);
						tmp.appendChild(document.createElement("br"));
					}
					for (index=0; index < json.files.length; index++)
					{
						item = document.createElement("a");
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
	var tmp = document.getElementById("update_file_upload");
	if (tmp)
	{
		var request = new XMLHttpRequest();
		request.open("POST", "firmwareupdate", true);
		noCacheInRequest(request);
		request.responseType = "text";
		request.onload = function()
		{
			document.body.style.cursor = "default";
			window.location.href = "update.html"
		}
			
		var data = new FormData();
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
