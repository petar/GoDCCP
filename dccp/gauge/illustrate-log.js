// TODO: use scales

var w = 700,
    h = 700;

// Create SVG element
var svg = d3.select("body")
	.append("svg:svg")
		.attr("width", w)
		.attr("height", h);

// Make master group
var g = svg.append("svg:g")
	.attr("transform", "translate(20, 20)");

var data_nodes = [
	{x: 0, y: 10}, 
	{x: 0, y: 30},
	{x: 100, y: 60},
	{x: 100, y: 80},
	{x: 0, y: 170}
	];

// Nodes
g.selectAll("__circle")
		.data(data_nodes)
	.enter().append("svg:circle")
		.attr("transform", function(d) { return "translate("+d.x+","+d.y+")" } )
		.attr("r", 7)
		.attr("stroke", "#777")
		.attr("fill", "#f8f8f8");

// Polygons
var data_polygons = [
	[{x:0, y:30}, {x:100, y:60}, {x:100, y:80}, {x:0, y:170}]
	];

var line = d3.svg.line()
	.x(function(d) { return d.x; })
	.y(function(d) { return d.y; })
	.interpolate("basis");

g.selectAll("__polygon")
		.data(data_polygons)
	.enter().append("svg:path")
		.attr("d", function(d) { return poly(d) + "Z"; })
		.attr("stroke", "#c88")
		.attr("fill", "#fdd");
