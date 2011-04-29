// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import "os"

// muxMsg{} contains the source and destination labels of a flow.
type muxMsg struct {
	Source, Sink *Label
}

// Len() returns the length of the flow pair's footprint in wire format
func (msg *muxMsg) Len() int { return msg.Source.Len() + msg.Sink.Len() }

// readMuxMsg() decodes a muxMsg{} from wire format
func readMuxMsg(p []byte) (msg *muxMsg, n int, err os.Error) {
	source, n0, err := ReadLabel(p)
	if err != nil {
		return nil, 0, err
	}
	dest, n1, err := ReadLabel(p[n0:])
	if err != nil {
		return nil, 0, err
	}
	return &muxMsg{source, dest}, n0+n1, nil
}

// Write() encodes the muxMsg{} to p@ in wire format
func (msg *muxMsg) Write(p []byte) (n int, err os.Error) {
	n0, err := msg.Source.Write(p)
	if err != nil {
		return 0, err
	}
	n1, err := msg.Sink.Write(p[n0:])
	if err != nil {
		return 0, err
	}
	return n0+n1, nil
}
