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
 * Weather station - Graph
*/

class SVGGraph {
	static get GRIDLINE_XOFFSET() { return 10; }
	static get GRIDLINE_YOFFSET() { return 10; }
	static get TEXT_XOFFSET() { return 10; }
	static get TEXT_YOFFSET() { return 18; }
	
	static get SVG_NAMESPACE() { return "http://www.w3.org/2000/svg"; }
  	constructor(prefix, minXScale, maxXScale, minYScale, maxYScale, maxXValue, maxYValue, stepX, stepX_bold, stepY, stepY_bold, leftMargin, bottomMargin, rightMargin, topMargin)
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
		this.getXAxisText = null;
		this.getYAxisText = null;
		this.userParam = null;
		this.yScaleTextOffset = "1.3em";
		this.YStepFactor = 1;
		if (this.stepY_bold < 1.0) {
			const nachkommastellen = (this.stepY_bold.toString().split('.')[1] || '').length;
			this.YStepFactor = Math.pow(10, nachkommastellen);
		}
		this.svg = null;
	}
	getTransformParam() {
		let w = this.svg.getBoundingClientRect().width;
		let h = this.svg.getBoundingClientRect().height;
		const yOffset = h - this.bottomMargin;
		w -= (this.leftMargin + this.rightMargin);
		h -= (this.bottomMargin + this.topMargin);
		const scaleH = h / (this.maxYScale - this.minYScale);
		const scaleW = w / (this.maxXScale - this.minXScale);
		return new TransformParam(w, h, this.leftMargin, yOffset, scaleW, -scaleH);
	}

	doResize() {
		this.doInit();
		this.createData();
	}

	doInit() {
		this.svg = document.getElementById(this.prefix + "svg");

		const frame = document.getElementById(this.prefix + "frame");
		const axis = document.getElementById(this.prefix + "axis");
		const data = document.getElementById(this.prefix + "data");
		const gridX = document.getElementById(this.prefix + "gridx");
		const gridY = document.getElementById(this.prefix + "gridy");
		const w = this.svg.getBoundingClientRect().width;
		const h = this.svg.getBoundingClientRect().height;

		removeAllChilds(frame);
		removeAllChilds(axis);
		removeAllChilds(data);
		removeAllChilds(gridX);
		removeAllChilds(gridY);

		if (this.svg.style.vectorEffect === undefined)
		{
			this.vectorEffectSupport = false;
			this.sw_gridLine = parseFloat(getStyleRuleValue("strokeWidth", ".gridline"));
			this.sw_gridLineBold = parseFloat(getStyleRuleValue("strokeWidth", ".gridline_bold"));
			let tmp = getStyleRuleValue("strokeDasharray", ".gridline_dash");
			if (tmp != null && tmp != undefined)
				tmp = tmp.split(",");
			else
				tmp = null;
			if (tmp != null && tmp.length >= 2)
			{
				this.sw_gridLineDash = new Array();
				this.sw_gridLineDash[0] = parseFloat(tmp[0]);
				this.sw_gridLineDash[1] = parseFloat(tmp[1]);
			}
			for (let i=0; i < this.markers.length; i++)
				this.markers[i].swMarker = parseFloat(getStyleRuleValue("strokeWidth", this.markers[i].cssClass));

			if (this.sw_graphData == null)
				this.sw_graphData = parseFloat(getStyleRuleValue("strokeWidth", "." + this.prefix + "graphdata"))/2.0;
		}

		const rect = document.createElementNS(SVGGraph.SVG_NAMESPACE, "rect");
		rect.setAttribute("width", w);
		rect.setAttribute("height", h);
		rect.setAttribute("class", "graphrect " + this.prefix + "graphrect");
		frame.appendChild(rect);

		const lineX = document.createElementNS(SVGGraph.SVG_NAMESPACE, "line");
		lineX.setAttribute("id", this.prefix + "lineX");
		lineX.setAttribute("x1", this.leftMargin);
		lineX.setAttribute("y1", h - this.bottomMargin);
		lineX.setAttribute("x2", w - this.rightMargin);
		lineX.setAttribute("y2", h - this.bottomMargin);
		lineX.setAttribute("class", "line axis axis_x");
		lineX.setAttribute("marker-end", "url(#" + this.prefix + "arrow)");
		axis.appendChild(lineX);

		const lineY = document.createElementNS(SVGGraph.SVG_NAMESPACE, "line");
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

	createXGrid () {
		const gridX = document.getElementById(this.prefix + "gridx");
		removeAllChilds(gridX);

		const g1 = document.createElementNS(SVGGraph.SVG_NAMESPACE, "g");
		const g2 = document.createElementNS(SVGGraph.SVG_NAMESPACE, "g");
		g1.setAttribute("transform", "translate(" + this.transform.transX + ", " + this.transform.transY + ")");
		g2.setAttribute("transform", "scale(" + this.transform.scaleX + " " + this.transform.scaleY + ")");

		let text, line, tmp;
		line = document.createElementNS(SVGGraph.SVG_NAMESPACE, "line");
		line.setAttribute("x1", 0);
		line.setAttribute("x2", 0);
		line.setAttribute("y2", 0);

	//	line.setAttribute("y1", ((this.minXScale ? 1 : 2 ) * SVGGraph.GRIDLINE_YOFFSET) / this.transform.scaleY);
		line.setAttribute("y1", SVGGraph.GRIDLINE_YOFFSET / this.transform.scaleY);

		line.setAttribute("class", "line gridline gridline_bold");
		if (!this.vectorEffectSupport && this.sw_gridLineBold != null)
			line.setAttribute("style", "stroke-width:" + (this.sw_gridLineBold / this.transform.scaleX) + "px;");
		g2.appendChild(line);

		for (let i=this.stepX; i<=(this.maxXValue - this.minXScale); i+=this.stepX)
		{
			line = document.createElementNS(SVGGraph.SVG_NAMESPACE, "line");
			line.setAttribute("id", this.prefix + "gridline_x_" + i);
			line.setAttribute("x1", i);
			line.setAttribute("x2", i);
			line.setAttribute("y2", (this.maxYValue - this.minYScale));
			if (i % this.stepX_bold == 0)
			{
				line.setAttribute("y1", SVGGraph.GRIDLINE_YOFFSET / this.transform.scaleY);
				line.setAttribute("class", "line gridline gridline_bold");
				if (!this.vectorEffectSupport && this.sw_gridLineBold != null)
					line.setAttribute("style", "stroke-width:" + (this.sw_gridLineBold / this.transform.scaleX) + "px;");

				text = document.createElementNS(SVGGraph.SVG_NAMESPACE, "text");
				text.setAttribute("x", i*this.transform.scaleX + this.transform.transX);
				text.setAttribute("y", this.transform.transY + SVGGraph.GRIDLINE_YOFFSET + SVGGraph.TEXT_YOFFSET);
				text.setAttribute("class", "gridtext gridtext_x");
				text.textContent = (this.getXAxisText ? this.getXAxisText(i + this.minXScale, this.userParam) : i + this.minXScale);
				gridX.appendChild(text);
			}
			else
			{
				line.setAttribute("y1", 0);
				line.setAttribute("class", "line gridline gridline_dash");
				let tmp = "";
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

		text = document.createElementNS(SVGGraph.SVG_NAMESPACE, "text");
		text.setAttribute("x", this.transform.transX);
		text.setAttribute("y", this.transform.transY + SVGGraph.GRIDLINE_YOFFSET + SVGGraph.TEXT_YOFFSET);
		text.setAttribute("class", "gridtext gridtext_x");
		text.textContent = (this.getXAxisText ? this.getXAxisText(this.minXScale, this.userParam) : this.minXScale);
		gridX.appendChild(text);

		text = document.createElementNS(SVGGraph.SVG_NAMESPACE, "text");

		text.setAttribute("x", (this.maxXValue - this.minXScale) * 0.5 * this.transform.scaleX + this.transform.transX);
		text.setAttribute("y", ((this.maxYValue - this.minYScale) * this.transform.scaleY + this.transform.transY)/2);
		text.setAttribute("dy", "0.3em");
		text.setAttribute("class", "axistext axistext_x");
		text.textContent = this.xScaleText;
		gridX.appendChild(text);

		gridX.appendChild(g1);
		this.xGridCreated = true;
	}

	createYGrid() {
		const gridY = document.getElementById(this.prefix + "gridy");
		removeAllChilds(gridY);

		const g1 = document.createElementNS(SVGGraph.SVG_NAMESPACE, "g");
		const g2 = document.createElementNS(SVGGraph.SVG_NAMESPACE, "g");
		g1.setAttribute("transform", "translate(" + this.transform.transX + ", " + this.transform.transY + ")");
		g2.setAttribute("transform", "scale(" + this.transform.scaleX + " " + this.transform.scaleY + ")");

		let line, text, tmp;
		line = document.createElementNS(SVGGraph.SVG_NAMESPACE, "line");
		line.setAttribute("x1", -SVGGraph.GRIDLINE_XOFFSET/this.transform.scaleX);
		line.setAttribute("x2", 0);
		line.setAttribute("y1", 0);
		line.setAttribute("y2", 0);
		line.setAttribute("class", "line gridline gridline_bold");
		if (!this.vectorEffectSupport && this.sw_gridLineBold != null)
			line.setAttribute("style", "stroke-width:" + (-this.sw_gridLineBold / this.transform.scaleY) + "px;");
		g2.appendChild(line);

		for (let i=0; i<=(this.maxYValue - this.minYScale); i+=this.stepY)
		{
			line = document.createElementNS(SVGGraph.SVG_NAMESPACE, "line");
			line.setAttribute("id", this.prefix + "gridline_y_" + i);
			line.setAttribute("y1", i);
			line.setAttribute("y2", i);
			line.setAttribute("x2", (this.maxXValue - this.minXScale));
			let isBold = false
			if (this.YStepFactor == 1)  {
				isBold = (i % this.stepY_bold == 0)
			} else {
				const iInt = Math.round(i * this.YStepFactor);
				const stepInt = Math.round(this.stepY_bold * this.YStepFactor);
				if (iInt % stepInt === 0) {
					isBold = true;
				}
			}
			if (isBold)
			{
				line.setAttribute("x1", -SVGGraph.GRIDLINE_XOFFSET / this.transform.scaleX);
				line.setAttribute("class", "line gridline gridline_bold");
				if (!this.vectorEffectSupport && this.sw_gridLineBold != null)
					line.setAttribute("style", "stroke-width:" + (-this.sw_gridLineBold / this.transform.scaleY) + "px;");

				text = document.createElementNS(SVGGraph.SVG_NAMESPACE, "text");
				text.setAttribute("x", -SVGGraph.GRIDLINE_XOFFSET - SVGGraph.TEXT_XOFFSET + this.transform.transX);
				text.setAttribute("y", i * this.transform.scaleY + this.transform.transY);
				text.setAttribute("dy", "0.3em");
				text.setAttribute("class", "gridtext gridtext_y");
				let txt = i + this.minYScale
				if (this.stepY_bold < 1.0) {
					txt = (Math.round(txt*this.YStepFactor)/this.YStepFactor).toFixed(1);
				}
				text.textContent = (this.getYAxisText ? this.getYAxisText(txt, this.userParam) : txt);
				gridY.appendChild(text);
			}
			else
			{
				line.setAttribute("x1", 0);
				line.setAttribute("class", "line gridline gridline_dash");
				let tmp = "";
				if (!this.vectorEffectSupport && this.sw_gridLine != null)
					tmp += "stroke-width:" + (-this.sw_gridLine / this.transform.scaleY) + "px;";
				if (!this.vectorEffectSupport && this.sw_gridLineDash != null)
					tmp += "stroke-dasharray:" + (this.sw_gridLineDash[0] / this.transform.scaleX) + "px," + (this.sw_gridLineDash[1] / this.transform.scaleX) + "px;";
				if (tmp.length > 0)
					line.setAttribute("style", tmp);
			}
			g2.appendChild(line);
		}

		text = document.createElementNS(SVGGraph.SVG_NAMESPACE, "text");
		text.setAttribute("x", -SVGGraph.GRIDLINE_XOFFSET - SVGGraph.TEXT_XOFFSET + this.transform.transX);
		text.setAttribute("y", this.transform.transY);
		text.setAttribute("dy", "0.3em");
		text.setAttribute("class", "gridtext gridtext_y");
		text.textContent = (this.getYAxisText ? this.getYAxisText(this.minYValue, this.userParam) : this.minYValue);
		gridY.appendChild(text);

		text = document.createElementNS(SVGGraph.SVG_NAMESPACE, "text");
		const y = 0.5*(this.maxYValue - this.minYScale) * this.transform.scaleY + this.transform.transY;
		const x = (this.maxXValue - this.minXScale) * this.transform.scaleX + this.transform.transX + 2;
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

	addData (xArray, yArray) {
		this.xArray = xArray;
		this.yArray = yArray;
		this.createData();
	}

	createData () {
		let tmp, i, line;

		if (this.xGridCreated == false)
			this.createXGrid();
		if (this.yGridCreated == false)
			this.createYGrid();

		const data = document.getElementById(this.prefix + "data");
		removeAllChilds(data);

		const g1 = document.createElementNS(SVGGraph.SVG_NAMESPACE, "g");
		const g2 = document.createElementNS(SVGGraph.SVG_NAMESPACE, "g");
		g1.setAttribute("transform", "translate(" + this.transform.transX + ", " + this.transform.transY + ")");
		g2.setAttribute("transform", "scale(" + this.transform.scaleX + " " + this.transform.scaleY + ")");
		g2.setAttribute("id", this.prefix + "g2");

		const g3 = document.createElementNS(SVGGraph.SVG_NAMESPACE, "g");
		g3.setAttribute("id", this.prefix + "g3");
		
		if (this.xArray !== undefined && this.yArray !== undefined)
		{
			let l = this.xArray.length;
			if (this.yArray.length < l)
				l = this.yArray.length;
			
			if (this.hasBullets)
			{
				for (let i = 0; i<l; i++)
				{
					let y = this.yArray[i]
					if (y > this.maxYScale - this.stepY)
						y = this.maxYScale - this.stepY;
					const ellipse = document.createElementNS(SVGGraph.SVG_NAMESPACE, "ellipse");
					ellipse.setAttribute("class", "bullet");
					ellipse.setAttribute("id", this.prefix + "graphdata_bullet_" + i);
					ellipse.setAttribute("cx", this.xArray[i] - this.minXScale);
					ellipse.setAttribute("cy", y - this.minYScale);
					ellipse.setAttribute("rx", this.radius_x_bullet / this.transform.scaleX);
					ellipse.setAttribute("ry", this.radius_y_bullet / (-this.transform.scaleY));
					g3.appendChild(ellipse);
				}
			}
			else
			{
				const strBuilder = new StringBuilder();

				for (let i = 0; i<l; i++)
				{
					let y = this.yArray[i]
					if (y > this.maxYScale - this.stepY)
						y = this.maxYScale- this.stepY;
					strBuilder.append((this.xArray[i] - this.minXScale) + "," + (y - this.minYScale));
				}
				
				const line = document.createElementNS(SVGGraph.SVG_NAMESPACE, "polyline");
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
}

class TransformParam {
	constructor(w, h, transX, transY, scaleX, scaleY) {
		this.w = Number(w);
		this.h = Number(h);
		this.transX = Number(transX);
		this.transY = Number(transY);
		this.scaleX = Number(scaleX);
		this.scaleY = Number(scaleY);
	}
}

// StringBuilder helper object - BEGIN
class StringBuilder {
	constructor() {
		this.stringArray = [];
	}

	append(value) {
		if (value)
			this.stringArray.push(value);
	}

	clear() {
		this.stringArray = [];
	}

	toString(sep) {
		return this.stringArray.join(sep ? sep : "");
	}

	toArray(sep) {
		return this.stringArray.slice(1);
	}

	isEmpty() {
		return this.stringArray.length <= 1;
	}
}
// StringBuilder helper object - END

function removeAllChilds(node)
{
	let last;
	while (last = node.lastChild)
		node.removeChild(last);
}
