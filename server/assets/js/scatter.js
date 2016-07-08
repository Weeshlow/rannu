function scatter(chartId, loadingId, filename) {
  'use strict';

  var margin = {top: 20, right: 20, bottom: 65, left: 65};
  var width = 600 - margin.left - margin.right;
  var height = 600 - margin.top - margin.bottom;

  var xValue = function(d) { return d.pc1; };
  var xScale = d3.scale
    .linear()
    .range([0, width]);
  var xMap = function(d) { return xScale(xValue(d));};
  var xAxis = d3.svg.axis()
    .ticks(5)
    .scale(xScale)
    .orient('bottom');

  var yValue = function(d) { return d.pc2; };
  var yScale = d3.scale.linear()
    .range([height, 0]);
  var yMap = function(d) { return yScale(yValue(d)); };
  var yAxis = d3.svg.axis()
    .ticks(5)
    .scale(yScale)
    .orient('left');

  var pointClass = function(d) {
    if (Math.round(d.value) === 1) {
      return 'point default';
    }
    return 'point';
  };
  var legendClass = function(d) {
    if (d === 'Default') {
      return 'legend default';
    }
    return 'legend';
  }

  var svg = d3.select(chartId)
    .append('svg')
    .attr('width', width + margin.left + margin.right)
    .attr('height', height + margin.top + margin.bottom)
    .append('g')
    .attr('transform', 'translate(' + margin.left + ',' + margin.top + ')');

  d3.csv(filename, function(error, data) {
    data.forEach(function(d) {
      d.pc1 = +d.pc1;
      d.pc2 = +d.pc2;
      d.value = +d.value;
    });

    d3.select(loadingId)
      .style('display', 'none');

    xScale.domain([d3.min(data, xValue) - 1, d3.max(data, xValue) + 1]);
    yScale.domain([d3.min(data, yValue) - 1, d3.max(data, yValue) + 1]);

    svg.append('g')
      .attr('class', 'x axis')
      .attr('transform', 'translate(0,' + height + ')')
      .call(xAxis)
      .append('text')
      .attr('class', 'label')
      .attr('x', width)
      .attr('y', -6)
      .style('text-anchor', 'end')
      .text('Principal Component 1');

    svg.append('g')
      .attr('class', 'y axis')
      .call(yAxis)
      .append('text')
      .attr('class', 'label')
      .attr('transform', 'rotate(-90)')
      .attr('y', 6)
      .attr('dy', '.71em')
      .style('text-anchor', 'end')
      .text('Principal Component 2');

    svg.selectAll('.dot')
      .data(data)
      .enter()
      .append('circle')
      .attr('class', pointClass)
      .attr('r', 3.5)
      .attr('cx', xMap)
      .attr('cy', yMap);

    var legend = svg.selectAll('.legend')
      .data(['Default', 'No Default'])
      .enter()
      .append('g')
      .attr('class', legendClass)
      .attr('transform', function(d, i) { return 'translate(0,' + i * 20 + ')'; });

    legend.append('rect')
      .attr('x', width - 18)
      .attr('width', 18)
      .attr('height', 18);

    legend.append('text')
      .attr('x', width - 24)
      .attr('y', 9)
      .attr('dy', '.35em')
      .style('text-anchor', 'end')
      .text(function(d) { return d; })
  });
}
