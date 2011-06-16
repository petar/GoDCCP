# Copyright 2010 GoDCCP Authors. All rights reserved.
# Use of this source code is governed by a 
# license that can be found in the LICENSE file.

include $(GOROOT)/src/Make.inc

all:	install

install:
	cd dccp && make install && \
	cd retransmit && make install

clean:
	cd dccp && make clean && \
	cd retransmit && make clean

nuke:
	cd dccp && make nuke && \
	cd retransmit && make nuke
