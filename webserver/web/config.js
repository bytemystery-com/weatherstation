const version = "v1.6";

var debug = 0;

var data_request = null;

function doLoad()
{
	doGetValues();
	
	if ((tmp = document.getElementById("version")))
		tmp.textContent = version;
}

function doPass()
{
	var ctrl, tmp;
	if ((ctrl = document.getElementById("use_pass")))
	{
		if ((tmp = document.getElementById("pass")))
			tmp.disabled = !ctrl.checked;
	}
}

function doGetValues()
{
	if (data_request == null)
	{
		data_request = new XMLHttpRequest();
		data_request.open("GET","getcfg", true);
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
				log(data_request.response);
				setValue("t_fc", Number(json.filterdata.t_fc).toFixed(5));
				setValue("t_fc2", Number(json.filterdata.t_fc2).toFixed(5));
				setValue("h_fc", Number(json.filterdata.h_fc).toFixed(5));
				setValue("h_fc2", Number(json.filterdata.h_fc2).toFixed(5));
				setValue("d_fc", Number(json.filterdata.d_fc).toFixed(5));
				setValue("d_fc2", Number(json.filterdata.d_fc2).toFixed(5));
				setValue("p_fc", Number(json.filterdata.p_fc).toFixed(5));
				setValue("p_fc2", Number(json.filterdata.p_fc2).toFixed(5));
				setValue("l_fc", Number(json.filterdata.l_fc).toFixed(5));
				setValue("l_fc2", Number(json.filterdata.l_fc2).toFixed(5));
				setValue("wb_fc", Number(json.filterdata.wb_fc).toFixed(5));
				setValue("wb_fc2", Number(json.filterdata.wb_fc2).toFixed(5));
				setValue("height", Number(json.general.height).toFixed(0));
				setCheck("use_pass", json.general.use_pass);
				setValue("name", json.general.name);
				document.title = json.general.name;
			
				doPass();
			} // response != NULL

			data_request = null;
		}
		data_request.send();
	}
}

function doSetValues(form)
{
	if (data_request)
		data_request.abort();

	var request = new XMLHttpRequest();
	request.open(form.method, form.action, true);
	noCacheInRequest(request);
	var refreshTime = 1000;
	request.onload = function()
	{
		log(request.responseText);
		setTimeout(function()
		{
			doGetValues();
		}, refreshTime);
	}
	request.onerror = function()
	{
		log("Error send setup:" + request.responseText);
		setTimeout(function()
		{
			doGetValues();
		}, refreshTime);
	}
	var data = new FormData();

	prepareData(data, "t_fc", null);
	prepareData(data, "t_fc2", null);
	prepareData(data, "h_fc", null);
	prepareData(data, "h_fc2", null);
	prepareData(data, "d_fc", null);
	prepareData(data, "d_fc2", null);
	prepareData(data, "p_fc", null);
	prepareData(data, "p_fc2", null);
	prepareData(data, "l_fc", null);
	prepareData(data, "l_fc2", null);
	prepareData(data, "wb_fc", null);
	prepareData(data, "wb_fc2", null);
	prepareData(data, "height", null);

	prepareDataFromCheckBox(data, "use_pass", "use_pass");
	prepareDataFromString(data, "pass", "pass");
	prepareDataFromString(data, "name", "name");
	
	request.send(data);
	
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

function setValue(id, value)
{
	var tmp;
	if ((tmp = document.getElementById(id)))
		tmp.value = value;
}

function setCheck(id, value)
{
	var tmp;
	var ret = false;
	if ((tmp = document.getElementById(id)))
	{
		if (tmp.checked != value)
			ret = true;
		tmp.checked = (value ? true : false);
	}
	return ret;
}

function setOption(id, value)
{
	var tmp;
	var ret = false;
	if ((tmp = document.getElementById(id)))
	{
		var oldIndex = tmp.selectedIndex;
		for (var i=0; i<tmp.options.length; i++)
		{
			if (tmp.options[i].value == value)
			{
				tmp.selectedIndex = i;
				return;
			}
		}
		tmp.selectedIndex = -1;
		ret = (tmp.selectedIndex != oldIndex);
	}
	return ret;
}

function emptyOptions(ctrl)
{
	while(ctrl.length > 0)
		ctrl.remove(ctrl.length-1);
}

function addOptions(ctrl, text, value)
{
	var op = document.createElement("option");
	op.text = text;
	op.value = value;
	ctrl.add(op);
}

function prepareData(data, id, scale, name)
{
	var ctrl;
	if (ctrl = document.getElementById(id))
	{
		var value = parseFloat(ctrl.value);
		var tmp = ctrl.getAttribute("min");
		if (tmp)
		{
			var min = parseFloat(tmp);
			if (value < min)
				value = min;
		}
		tmp = ctrl.getAttribute("max");
		if (tmp)
		{
			var max = parseFloat(tmp);
			if (value > max)
				value = max;
		}
		if (scale != null)
			value = Math.round(value*scale);

		data.append(name != null && name != undefined ? name : id, value);
	}
}

function prepareDataFromComboBox(data, id, name)
{
	var ctrl;
	if (ctrl = document.getElementById(id))
	{
		if (ctrl.selectedIndex >= 0)
			data.append(name != null && name != undefined ? name : id, ctrl.options[ctrl.selectedIndex].value);
	}
}

function prepareDataFromCheckBox(data, id, name)
{
	var ctrl;
	if (ctrl = document.getElementById(id))
		data.append(name != null && name != undefined ? name : id, ctrl.checked ? 1 : 0);
}

function prepareDataFromString(data, id, name)
{
	var ctrl;
	if (ctrl = document.getElementById(id))
		data.append(name != null && name != undefined ? name : id, ctrl.value);
}

function doMain()
{
	window.location.href = "index.html";
}

