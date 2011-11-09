// TODO: use scales

var w = 1200,
    h = 20000;

// Create SVG element
var svg = d3.select("body")
	.append("svg:svg")
		.attr("width", w)
		.attr("height", h);

var places = {
	"client": 0,
	"line":   300,
	"server": 600
};

function XOfPlace(place) {
	var x = places[place];
	if (_.isUndefined(x)) {
		x = 200*_.size(places);
		places[place] = x;
	}
	return x;
}

var lastt = 0, lasty = 0;
var timey = {};
function YOfTime(t) {
	var y = timey[t];
	if (_.isUndefined(y)) {
		var dy = Math.log(t-lastt)*3;
		y = lasty + dy;
		timey[t] = y;
		lastt = t;
		lasty = y;
	}
	return y;
}
for (i in data["check_ins"]) {
	y = YOfTime(data["check_ins"][i].time);
}

var colors = {
	"LISTEN":   "#0ff",
	"REQUEST":  "#f0f",
	"RESPOND":  "#ff0",
	"PARTOPEN": "#0f0",
	"OPEN":     "#080",
	"CLOSEREQ": "#f00",
	"CLOSING":  "#800",
	"CLOSED":   "#400",
	"TIMEWAIT": "#f8f"
};
function ColorOfState(state) {
	var c = colors[state];
	if (_.isUndefined(c)) {
		return "#ccc";
	}
	return c;
}

// Make master group
var g = svg.append("svg:g")
	.attr("transform", "translate(200, 100)");

// Places

gplaces = g.append("svg:g");
d3places = gplaces.selectAll("XXX")
		.data(data["places"])
	.enter().append("svg:g")
		.attr("transform", function(d) { return "translate("+XOfPlace(d.name)+",0)" } );

d3places.selectAll("XXX")
		.data(function(d) { return d.intervals; })
	.enter().append("svg:rect")
		.attr("x", -6)
		.attr("y", function(d) { return YOfTime(d.start) })
		.attr("width", 13)
		.attr("height", function(d) { return YOfTime(d.end) - YOfTime(d.start); })
		.attr("fill", function(d) { return ColorOfState(d.state) })
		//.attr("stroke", "#000")
		.style("opacity", "0.3");

// Trips

var line = d3.svg.line()
	.x(function(d) { return XOfPlace(d.place); })
	.y(function(d) { return YOfTime(d.time); })
	.interpolate("linear");

gtrips = g.append("svg:g");
gtrips.selectAll("XXX")
		.data(data["trips"])
	.enter().append("svg:path")
		.attr("d", function(d) { return line(d.path) })
		.attr("stroke", "#000")
		.attr("stroke-width", "5")
		.attr("stroke-linecap", "round")
		.attr("stroke-linejoin", "round")
		.attr("fill", "none")
		.style("opacity", "0.2");

// Nodes

gnodes = g.append("svg:g");
d3nodes = gnodes.selectAll("XXX")
		.data(data["check_ins"])
	.enter().append("svg:g")
		.attr("transform", function(d) { return "translate("+XOfPlace(d.place)+","+YOfTime(d.time)+")" } );

d3nodes.append("svg:circle")
		.attr("r", 9)
		.attr("stroke", "#fff")
		.attr("stroke-width", "2")
		.attr("fill", "#000");

d3nodes.append("svg:text")
		.attr("transform", function(d) { return "translate(-12,4)" } )
		.style("font-family", "Verdana")
		.style("font-size", "10px")
		.style("text-anchor", "end")
		.attr("fill", "#444")
		.text(function(d) { return d.time / 1000000000.0 });

d3nodes.append("svg:text")
		.attr("transform", function(d) { return "translate(12,-4)" } )
		.style("font-family", "Verdana")
		.style("font-size", "12px")
		.style("text-anchor", "start")
		.attr("fill", "#bbb")
		.text(function(d) { return d.sub + "/" + d.type + ": " + d.comment });

d3nodes.append("svg:text")
		.attr("transform", function(d) { return "translate(12,11)" } )
		.style("font-family", "Verdana")
		.style("font-size", "12px")
		.style("text-anchor", "start")
		.attr("fill", "#c55")
		.text(function(d) { return "SeqNo: " + d.seqno + ", AckNo:" + d.ackno });
