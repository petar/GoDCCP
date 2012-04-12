#!/bin/sh
go test -test.run=RoundtripEstimation && dccp-inspector -emits=true var/rtt.emit > var/rtt.html
