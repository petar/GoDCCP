#!/bin/sh
go test -test.run=Loss; dccp-inspector -emits=true var/loss.emit > var/loss.html
