// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp


// Link{} is an abstract interface to a physical packet layer which
// sends and receives packets between ourselves and a remote node.
type Link interface {

	Healthy() bool
	SourceIP() []byte
	DestIP() []byte
	MTU() uint32				// (Path) Maximum Transmission Unit, PMTU
	Send(p []byte) os.Error			// Send a block of data
	Recv() (p []byte, err os.Error)		// Receive next block of data
}

type Time interface {
	Now()		// returns current time in nanoseconds
}
