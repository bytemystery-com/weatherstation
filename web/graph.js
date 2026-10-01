const gridLineXOffset = 10;
const gridLineYOffset = 10;
const textYOffset = 18;
const textXOffset = 10;

// SVGGraph
function SVGGraph(prefix, minXScale, maxXScale, minYScale, maxYScale, maxXValue, maxYValue, stepX, stepX_bold, stepY, stepY_bold, leftMargin, bottomMargin, rightMargin, topMargin)
{
	this.transform = null;
	this.prefix = prefix;
	this.minXScale = minXScale;
	this.maxXScale = maxXScale;
	this.minYScale = minYScale;
	this.maxYScale = maxYScale;
	this.maxXValue = maxXValue;
	this.maxYValue = maxYValue;
	this.stepX = stepX;
	this.stepX_bold = stepX_bold;
	this.stepY = stepY;
	this.stepY_bold = stepY_bold;
	this.xGridCreated = false;
	this.yGridCreated = false;
	this.leftMargin = leftMargin;
	this.bottomMargin = bottomMargin;
	this.rightMargin = rightMargin;
	this.topMargin = topMargin;
	this.xScaleText = "x";
	this.yScaleText = "y";
	this.vectorEffectSupport = true;
	this.sw_graphData = null;
	this.sw_gridLine = null;
	this.sw_gridLineBold = null;
	this.sw_gridDash = null;
	this.radius_x_bullet = 2;
	this.radius_y_bullet = 2;
	this.hasBullets = false;
	this.svgNameSpace = "http://www.w3.org/2000/svg";
	this.getXAxisText = null;
	this.getYAxisText = null;
	this.userParam = null;
	this.yScaleTextOffset = "1.3em";
}

SVGGraph.prototype.getTransformParam = function()
{
	var svg = document.getElementById(this.prefix + "svg");
	var lineX = document.getElementById(this.prefix + "lineX");
	var lineY = document.getElementById(this.prefix + "lineY");
	var w = svg.getBoundingClientRect().width;
	var h = svg.getBoundingClientRect().height;
	var yOffset = h - this.bottomMargin;
	w -= (this.leftMargin + this.rightMargin);
	h -= (this.bottomMargin + this.topMargin);
	var scaleH = h / (this.maxYScale - this.minYScale);
	var scaleW = w / (this.maxXScale - this.minXScale);
	return new TransformParam(w, h, this.leftMargin, yOffset, scaleW, -scaleH);
}

SVGGraph.prototype.doResize = function()
{
	this.doInit();
	this.createData();
}

SVGGraph.prototype.doInit = function()
{
	var svg = document.getElementById(this.prefix + "svg");
	var frame = document.getElementById(this.prefix + "frame");
	var axis = document.getElementById(this.prefix + "axis");
	var data = document.getElementById(this.prefix + "data");
	var gridX = document.getElementById(this.prefix + "gridx");
	var gridY = document.getElementById(this.prefix + "gridy");
	var w = svg.getBoundingClientRect().width;
	var h = svg.getBoundingClientRect().height;

	removeAllChilds(frame);
	removeAllChilds(axis);
	removeAllChilds(data);
	removeAllChilds(gridX);
	removeAllChilds(gridY);

	if (svg.style.vectorEffect === undefined)
	{
		this.vectorEffectSupport = false;
		this.sw_gridLine = parseFloat(getStyleRuleValue("strokeWidth", ".gridline"));
		this.sw_gridLineBold = parseFloat(getStyleRuleValue("strokeWidth", ".gridline_bold"));
		var tmp = getStyleRuleValue("strokeDasharray", ".gridline_dash");
		if (tmp != null && tmp != undefined)
			tmp = tmp.split(",");
		else tmp = null;
		if (tmp != null && tmp.length >= 2)
		{
			this.sw_gridLineDash = new Array();
			this.sw_gridLineDash[0] = parseFloat(tmp[0]);
			this.sw_gridLineDash[1] = parseFloat(tmp[1]);
		}
		for (var i =0; i < this.markers.length; i++)
			this.markers[i].swMarker = parseFloat(getStyleRuleValue("strokeWidth", this.markers[i].cssClass));

		if (this.sw_graphData == null)
			this.sw_graphData = parseFloat(getStyleRuleValue("strokeWidth", "." + this.prefix + "graphdata"))/2.0;
	}


	var rect = document.createElementNS(this.svgNameSpace, "rect");
	rect.setAttribute("width", w);
	rect.setAttribute("height", h);
	rect.setAttribute("class", "graphrect " + this.prefix + "graphrect");
	frame.appendChild(rect);

	var lineX = document.createElementNS(this.svgNameSpace, "line");
	lineX.setAttribute("id", this.prefix + "lineX");
	lineX.setAttribute("x1", this.leftMargin);
	lineX.setAttribute("y1", h - this.bottomMargin);
	lineX.setAttribute("x2", w - this.rightMargin);
	lineX.setAttribute("y2", h - this.bottomMargin);
	lineX.setAttribute("class", "line axis axis_x");
	lineX.setAttribute("marker-end", "url(#" + this.prefix + "arrow)");
	axis.appendChild(lineX);

	var lineY = document.createElementNS(this.svgNameSpace, "line");
	lineY.setAttribute("id", this.prefix + "lineY");
	lineY.setAttribute("x1", this.leftMargin);
	lineY.setAttribute("y1", h - this.bottomMargin);
	lineY.setAttribute("x2", this.leftMargin);
	lineY.setAttribute("y2", this.topMargin);
	lineY.setAttribute("class", "line axis axis_y");
	lineY.setAttribute("marker-end", "url(#" + this.prefix + "arrow)");
	axis.appendChild(lineY);

	this.transform = this.getTransformParam();

	this.createXGrid();
	this.xGridCreated = false; // recreate with first data set
	this.createYGrid()
	this.yGridCreated = false; // recreate with first data set
}

SVGGraph.prototype.createXGrid = function()
{
	var gridX = document.getElementById(this.prefix + "gridx");
	removeAllChilds(gridX);

	var g1 = document.createElementNS(this.svgNameSpace, "g");
	var g2 = document.createElementNS(this.svgNameSpace, "g");
	g1.setAttribute("transform", "translate(" + this.transform.transX + ", " + this.transform.transY + ")");
	g2.setAttribute("transform", "scale(" + this.transform.scaleX + " " + this.transform.scaleY + ")");

	var text, line, tmp;
	line = document.createElementNS(this.svgNameSpace, "line");
	line.setAttribute("x1", 0);
	line.setAttribute("x2", 0);
	line.setAttribute("y2", 0);

//	line.setAttribute("y1", ((this.minXScale ? 1 : 2 ) * gridLineYOffset) / this.transform.scaleY);
	line.setAttribute("y1", gridLineYOffset / this.transform.scaleY);

	line.setAttribute("class", "line gridline gridline_bold");
	if (!this.vectorEffectSupport && this.sw_gridLineBold != null)
		line.setAttribute("style", "stroke-width:" + (this.sw_gridLineBold / this.transform.scaleX) + "px;");
	g2.appendChild(line);

	for (var i=this.stepX; i<=(this.maxXValue - this.minXScale); i+=this.stepX)
	{
		line = document.createElementNS(this.svgNameSpace, "line");
		line.setAttribute("id", this.prefix + "gridline_x_" + i);
		line.setAttribute("x1", i);
		line.setAttribute("x2", i);
		line.setAttribute("y2", (this.maxYValue - this.minYScale));
		if (i % this.stepX_bold == 0)
		{
			line.setAttribute("y1", gridLineYOffset / this.transform.scaleY);
			line.setAttribute("class", "line gridline gridline_bold");
			if (!this.vectorEffectSupport && this.sw_gridLineBold != null)
				line.setAttribute("style", "stroke-width:" + (this.sw_gridLineBold / this.transform.scaleX) + "px;");

			text = document.createElementNS(this.svgNameSpace, "text");
			text.setAttribute("x", i*this.transform.scaleX + this.transform.transX);
			text.setAttribute("y", this.transform.transY + gridLineYOffset + textYOffset);
			text.setAttribute("class", "gridtext gridtext_x");
			text.textContent = (this.getXAxisText ? this.getXAxisText(i + this.minXScale, this.userParam) : i + this.minXScale);
			gridX.appendChild(text);
		}
		else
		{
			line.setAttribute("y1", 0);
			line.setAttribute("class", "line gridline gridline_dash");
			var tmp = "";
			if (!this.vectorEffectSupport && this.sw_gridLine != null)
				tmp += "stroke-width:" + (this.sw_gridLine / this.transform.scaleX) + "px;";
			if (!this.vectorEffectSupport && this.sw_gridLineDash != null)
				tmp += "stroke-dasharray:" + (-this.sw_gridLineDash[0] / this.transform.scaleY) + "px," + (-this.sw_gridLineDash[1] / this.transform.scaleY) + "px;";
			if (tmp.length > 0)
				line.setAttribute("style", tmp);
		}
		g2.appendChild(line);
	}

	g1.appendChild(g2);

	text = document.createElementNS(this.svgNameSpace, "text");
	text.setAttribute("x", this.transform.transX);
	text.setAttribute("y", this.transform.transY + gridLineYOffset + textYOffset);
	text.setAttribute("class", "gridtext gridtext_x");
	text.textContent = (this.getXAxisText ? this.getXAxisText(this.minXScale, this.userParam) : this.minXScale);
	gridX.appendChild(text);

	text = document.createElementNS(this.svgNameSpace, "text");

	text.setAttribute("x", (this.maxXValue - this.minXScale) * 0.5 * this.transform.scaleX + this.transform.transX);
	text.setAttribute("y", ((this.maxYValue - this.minYScale) * this.transform.scaleY + this.transform.transY)/2);
	text.setAttribute("dy", "0.3em");
	text.setAttribute("class", "axistext axistext_x");
	text.textContent = this.xScaleText;
	gridX.appendChild(text);

	gridX.appendChild(g1);
	this.xGridCreated = true;
}

SVGGraph.prototype.createYGrid = function()
{
	var gridY = document.getElementById(this.prefix + "gridy");
	removeAllChilds(gridY);

	var g1 = document.createElementNS(this.svgNameSpace, "g");
	var g2 = document.createElementNS(this.svgNameSpace, "g");
	g1.setAttribute("transform", "translate(" + this.transform.transX + ", " + this.transform.transY + ")");
	g2.setAttribute("transform", "scale(" + this.transform.scaleX + " " + this.transform.scaleY + ")");

	var line, text, tmp;
	line = document.createElementNS(this.svgNameSpace, "line");
	line.setAttribute("x1", -gridLineXOffset/this.transform.scaleX);
	line.setAttribute("x2", 0);
	line.setAttribute("y1", 0);
	line.setAttribute("y2", 0);
	line.setAttribute("class", "line gridline gridline_bold");
	if (!this.vectorEffectSupport && this.sw_gridLineBold != null)
		line.setAttribute("style", "stroke-width:" + (-this.sw_gridLineBold / this.transform.scaleY) + "px;");
	g2.appendChild(line);

	for (var i=this.stepY; i<=(this.maxYValue - this.minYScale); i+=this.stepY)
	{
		line = document.createElementNS(this.svgNameSpace, "line");
		line.setAttribute("id", this.prefix + "gridline_y_" + i);
		line.setAttribute("y1", i);
		line.setAttribute("y2", i);
		line.setAttribute("x2", (this.maxXValue - this.minXScale));
		if (i % this.stepY_bold == 0)
		{
			line.setAttribute("x1", -gridLineXOffset / this.transform.scaleX);
			line.setAttribute("class", "line gridline gridline_bold");
			if (!this.vectorEffectSupport && this.sw_gridLineBold != null)
				line.setAttribute("style", "stroke-width:" + (-this.sw_gridLineBold / this.transform.scaleY) + "px;");

			text = document.createElementNS(this.svgNameSpace, "text");
			text.setAttribute("x", -gridLineXOffset - textXOffset + this.transform.transX);
			text.setAttribute("y", i * this.transform.scaleY + this.transform.transY);
			text.setAttribute("dy", "0.3em");
			text.setAttribute("class", "gridtext gridtext_y");
			text.textContent = (this.getYAxisText ? this.getYAxisText(i + this.minYScale, this.userParam) : i + this.minYScale);
			gridY.appendChild(text);
		}
		else
		{
			line.setAttribute("x1", 0);
			line.setAttribute("class", "line gridline gridline_dash");
			var tmp = "";
			if (!this.vectorEffectSupport && this.sw_gridLine != null)
				tmp += "stroke-width:" + (-this.sw_gridLine / this.transform.scaleY) + "px;";
			if (!this.vectorEffectSupport && this.sw_gridLineDash != null)
				tmp += "stroke-dasharray:" + (this.sw_gridLineDash[0] / this.transform.scaleX) + "px," + (this.sw_gridLineDash[1] / this.transform.scaleX) + "px;";
			if (tmp.length > 0)
				line.setAttribute("style", tmp);
		}
		g2.appendChild(line);
	}

	text = document.createElementNS(this.svgNameSpace, "text");
	text.setAttribute("x", -gridLineXOffset - textXOffset + this.transform.transX);
	text.setAttribute("y", this.transform.transY);
	text.setAttribute("dy", "0.3em");
	text.setAttribute("class", "gridtext gridtext_y");
	text.textContent = (this.getYAxisText ? this.getYAxisText(this.minYValue, this.userParam) : this.minYValue);
	gridY.appendChild(text);

	text = document.createElementNS(this.svgNameSpace, "text");
	var y = 0.5*(this.maxYValue - this.minYScale) * this.transform.scaleY + this.transform.transY;
	var x = (this.maxXValue - this.minXScale) * this.transform.scaleX + this.transform.transX + 2;
	text.setAttribute("transform", "rotate(-90 " + x + "," + y + ")");
	text.setAttribute("dy", this.yScaleTextOffset);
	text.setAttribute("x", x);
	text.setAttribute("y", y);
	text.setAttribute("class", "axistext axistext_y");
	text.textContent = this.yScaleText;
	gridY.appendChild(text);

	g1.appendChild(g2);
	gridY.appendChild(g1);

	this.yGridCreated = true;
}

SVGGraph.prototype.addData = function(xArray, yArray)
{
	this.xArray = xArray;
	this.yArray = yArray;
	this.createData();
}

SVGGraph.prototype.createData = function()
{
	var tmp, i, line;

	if (this.xGridCreated == false)
		this.createXGrid();
	if (this.yGridCreated == false)
		this.createYGrid();

	var svg = document.getElementById(this.prefix + "svg");
	var data = document.getElementById(this.prefix + "data");
	removeAllChilds(data);

	var g1 = document.createElementNS(this.svgNameSpace, "g");
	var g2 = document.createElementNS(this.svgNameSpace, "g");
	g1.setAttribute("transform", "translate(" + this.transform.transX + ", " + this.transform.transY + ")");
	g2.setAttribute("transform", "scale(" + this.transform.scaleX + " " + this.transform.scaleY + ")");
	g2.setAttribute("id", this.prefix + "g2");

	var g3 = document.createElementNS(this.svgNameSpace, "g");
	g3.setAttribute("id", this.prefix + "g3");
	
	if (this.xArray !== undefined && this.yArray !== undefined)
	{
		var l = this.xArray.length;
		if (this.yArray.length < l)
			l = this.yArray.length;
		
		if (this.hasBullets)
		{
			var ellipse;
			for (var i = 0; i<l; i++)
			{
				ellipse = document.createElementNS(this.svgNameSpace, "ellipse");
				ellipse.setAttribute("class", "bullet");
				ellipse.setAttribute("id", this.prefix + "graphdata_bullet_" + i);
				ellipse.setAttribute("cx", this.xArray[i] - this.minXScale);
				ellipse.setAttribute("cy", this.yArray[i] - this.minYScale);
				ellipse.setAttribute("rx", this.radius_x_bullet / this.transform.scaleX);
				ellipse.setAttribute("ry", this.radius_y_bullet / (-this.transform.scaleY));
				g3.appendChild(ellipse);
			}
		}
		else
		{
			var strBuilder = new StringBuilder();
			for (var i = 0; i<l; i++)
				strBuilder.append((this.xArray[i] - this.minXScale) + "," + (this.yArray[i] - this.minYScale));
			
			var line = document.createElementNS(this.svgNameSpace, "polyline");
			line.setAttribute("id", this.prefix + "graphdata_line");
			line.setAttribute("class", "line " + this.prefix + "graphdata" + " graphdata");
			line.setAttribute("points", strBuilder.toString(" "));
			if (!this.vectorEffectSupport && this.sw_graphData != null)
				line.setAttribute("style", "stroke-width:" + (this.sw_graphData / (this.transform.scaleX)) + "px;");
			g3.appendChild(line);
		}
	}
	
	g2.appendChild(g3);

	g1.appendChild(g2);
	data.appendChild(g1);
}

function TransformParam(w, h, transX, transY, scaleX, scaleY)
{
	this.w = Number(w);
	this.h = Number(h);
	this.transX = Number(transX);
	this.transY = Number(transY);
	this.scaleX = Number(scaleX);
	this.scaleY = Number(scaleY);
}

function removeAllChilds(node)
{
	var last;
	while (last = node.lastChild)
		node.removeChild(last);
}

// StringBuilder helper object - BEGIN
function StringBuilder()
{
	this.stringArray = new Array("");
}

StringBuilder.prototype.append = function(value)
{
	if (value)
		this.stringArray.push(value);
}

StringBuilder.prototype.clear = function()
{
	this.stringArray = new Array("");
}

StringBuilder.prototype.toString = function(sep)
{
	return this.stringArray.join(sep ? sep : "");
}

StringBuilder.prototype.toArray = function(sep)
{
	return this.stringArray.slice(1);
}

StringBuilder.prototype.isEmpty = function()
{
	return this.stringArray.length <= 1;
}
// StringBuilder helper object - END
