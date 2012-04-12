#!/bin/sh
go test -test.run=RoundtripEstimation && dccp-inspector -emits=false var/rtt.emit > var/rtt.html
