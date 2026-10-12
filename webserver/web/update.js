
function doLoad()
{
	setInterval(function(){doPeriodic()}, 120*1000);
}

function doPeriodic()
{
	window.location.replace("index.html");
}
