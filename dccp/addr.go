// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import ()

type Addr struct{}

func (a *Addr) Network() string { return "GoDCCP" }

func (a *Addr) String() string { return "godccp.default" }

var DefaultAddr = Addr{}
