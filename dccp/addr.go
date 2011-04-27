// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"os"
)

// FlowPair{} contains the local and remote keys of a flow. It is mutable.
type FlowPair struct {
	Source, Dest *Label
}

// Len() returns the length of the flow pair's footprint in wire format
func (fpair *FlowPair) Len() int { return fpair.Source.Len() + fpair.Dest.Len() }

// Flip() switches the source and destination keys
func (fpair *FlowPair) Flip() { fpair.Source, fpair.Dest = fpair.Dest, fpair.Source }

// ReadFlowPair() reads a flow pair from p@ in wire format
func ReadFlowPair(p []byte) (fpair *FlowPair, n int, err os.Error) {
	source, n0, err := ReadLabel(p)
	if err != nil {
		return nil, 0, err
	}
	dest, n1, err := ReadLabel(p[n0:])
	if err != nil {
		return nil, 0, err
	}
	return &FlowPair{source, dest}, n0+n1, nil
}

// Write() writes the flow pair to p@ in wire format
func (fpair *FlowPair) Write(p []byte) (n int, err os.Error) {
	n0, err := fpair.Source.Write(p)
	if err != nil {
		return 0, err
	}
	n1, err := fpair.Dest.Write(p[n0:])
	if err != nil {
		return 0, err
	}
	return n0+n1, nil
}
