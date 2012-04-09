// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import (
	"github.com/petar/GoDCCP/dccp"
)

type CCID3 struct {}

func (CCID3) NewSender(run *dccp.Runtime, amb *dccp.Amb) dccp.SenderCongestionControl { 
	return newSender(run, amb)
}

func (CCID3) NewReceiver(run *dccp.Runtime, amb *dccp.Amb) dccp.ReceiverCongestionControl { 
	return newReceiver(run, amb)
}
