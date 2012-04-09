#!/bin/sh
go test -test.run=RoundtripEstimation && dccp-inspector var/rtt.emit > var/rtt.html
