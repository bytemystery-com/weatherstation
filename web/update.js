
function doLoad()
{
	setInterval(function(){doPeriodic()}, 6000);
}

function doPeriodic()
{
	window.location.replace("index.html");
}
