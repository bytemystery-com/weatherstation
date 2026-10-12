
function doLoad(duration)
{
	setInterval(function(){doPeriodic()}, duration*1000);
}

function doPeriodic()
{
	window.location.replace("index.html");
}
