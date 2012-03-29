#!/bin/sh
go test -test.run=Idle && dccp-inspector var/idle.emit > var/idle.html
