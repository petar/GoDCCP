// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"io"
	"os"
)

// flow{} acts as a packet ReadWriteCloser{} for Conn.
type flow struct {
	linkaddr LinkAddr	// Link-layer address of the remote
	pair     FlowPair	// Local and remote flow-layer keys
	swtch    *flowSwitch
	ch       chan switchHeader
}

func (flow *flow) Write(buf []byte) os.Error {
	return flow.swtch.Write(buf, flow.FlowID)
}

func (flow *flow) Read() (buf []byte, err os.Error) {
	buf = <-flow.ch
	if buf == nil {
		err = os.EBADF
	}
	return 
}

func (flow *flow) Close() os.Error {
	flow.swtch.delFlow(flow)
	return nil
}

