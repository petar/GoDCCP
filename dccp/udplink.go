// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import "net"

// UDPLink binds to a UDP port and acts as a Link type.
type UDPLink struct {
	c *net.UDPConn
}

func BindUDPLink(netw string, laddr *net.UDPAddr) (link *UDPLink, err error) {
	c, err := net.ListenUDP(netw, laddr)
	if err != nil {
		return nil, err
	}
	return &UDPLink{c}, nil
}

func (u *UDPLink) GetMTU() int { return 1500 }

func (u *UDPLink) SetReadTimeout(nsec int64) error {
	return u.c.SetReadTimeout(nsec)
}

func (u *UDPLink) ReadFrom(buf []byte) (n int, addr net.Addr, err error) {
	return u.c.ReadFrom(buf)
}

func (u *UDPLink) WriteTo(buf []byte, addr net.Addr) (n int, err error) {
	return u.c.WriteTo(buf, addr)
}

func (u *UDPLink) Close() error {
	return u.c.Close()
}
