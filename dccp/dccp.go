// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"net"
	"os"
)

type Stack struct {
	mux   *Mux
	link  Link
	ccid  CCID
}

// NewStack creates a new connection-handling object.
func NewStack(link Link, ccid CCID) *Stack {
	return &Stack{
		mux:  NewMux(link),
		link: link,
		ccid: ccid,
	}
}

// Dial initiates a new connection to the specified Link-layer address.
func (s *Stack) Dial(addr net.Addr, serviceCode uint32) (c BlockConn, err os.Error) {
	bc, err := s.mux.Dial(addr)
	if err != nil {
		return nil, err
	}
	hc := NewHeaderOverBlockConn(bc)
	c = newConnClient(hc, s.ccid.NewSender(), s.ccid.NewReceiver(), serviceCode)
	return c, nil
}

// Accept blocks until a new connecion is established. It then
// returns the connection.
func (s *Stack) Accept() (c BlockConn, err os.Error) {
	bc, err := s.mux.Accept()
	if err != nil {
		return nil, err
	}
	hc := NewHeaderOverBlockConn(bc)
	c = newConnServer(hc, s.ccid.NewSender(), s.ccid.NewReceiver())
	return c, nil
}
