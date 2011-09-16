// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import (
	//"os"
	"github.com/petar/GoDCCP/dccp"
)

type CCID3 struct {}

func (CCID3) NewSender(name string) dccp.SenderCongestionControl { 
	return newSender(name) 
}

func (CCID3) NewReceiver(name string) dccp.ReceiverCongestionControl { 
	return newReceiver(name) 
}
