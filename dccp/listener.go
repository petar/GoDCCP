// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"net"
)

type Listener struct {
	phy Physical
}

func Listen(phy Physical) (net.Listener, os.Error) {
	l := &Listener{
		phy: phy,
	}
	go l.loop();
	return l
}

// XXX: multiplex phy layer between listener and active conns
func (l *Listener) Accept() (c Conn, err os.Error) {
	h,err := e.readPacket()
	if err != nil {
		return nil, err
	}
	
}

func (l *Listener) Close() os.Error {
	return l.phy.Close();
}

func (l *Listener) Addr() net.Addr { return DefaultAddr }
