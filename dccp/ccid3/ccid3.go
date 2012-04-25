// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import (
	"github.com/petar/GoDCCP/dccp"
)

type CCID3 struct {}

func (CCID3) NewSender(env *dccp.Env, amb *dccp.Amb) dccp.SenderCongestionControl { 
	return newSender(env, amb)
}

func (CCID3) NewReceiver(env *dccp.Env, amb *dccp.Amb) dccp.ReceiverCongestionControl { 
	return newReceiver(env, amb)
}
