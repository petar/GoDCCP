// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import (
	//"os"
	"github.com/petar/GoDCCP/dccp"
)

// NewCCID3SenderFunc returns a function that makes new CCID3 HC-Sender Congestion Control 
func NewCCID3SenderFunc () dccp.NewSenderCongestionControlFunc {
	return func() dccp.SenderCongestionControl { return newSender() }
}

// NewCCID3ReceiverFunc returns a function that makes new CCID3 HC-Receiver Congestion Control 
func NewCCID3ReceiverFunc () dccp.NewReceiverCongestionControlFunc {
	return func() dccp.ReceiverCongestionControl { return newReceiver() }
}
