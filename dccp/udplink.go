// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"net"
)

// UDPLink{} binds to a UDP port and acts as a Link{} type.
type UDPLink struct {
}

func BindUDPLink() (link *UDPLink, err os.Error) {
	?
}

func (u *UDPLink) Read() (buf []byte, addr *Addr, err os.Error) {
}

func (u *UDPLink) Write(buf []byte, addr *Addr) (n int, err os.Error) {
}

func (u *UDPLink) Close() os.Error {
}
