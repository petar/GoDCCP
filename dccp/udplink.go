// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"net"
	"os"
)

// UDPLink{} binds to a UDP port and acts as a Link{} type.
type UDPLink struct {
	c *net.UDPConn
}

func BindUDPLink(netw string, laddr *net.UDPAddr) (link *UDPLink, err os.Error) {
	c, err := net.ListenUDP(netw, laddr)
	if err != nil {
		return nil, err
	}
	return &UDPLink{c}, nil
}

func (u *UDPLink) FragmentLen() int { return 1500 }

func (u *UDPLink) ReadFrom(buf []byte) (n int, addr net.Addr, err os.Error) {
	return u.c.ReadFrom(buf)
}

func (u *UDPLink) WriteTo(buf []byte, addr net.Addr) (n int, err os.Error) {
	return u.c.WriteTo(buf, addr)
}

func (u *UDPLink) Close() os.Error {
	return u.c.Close()
}
