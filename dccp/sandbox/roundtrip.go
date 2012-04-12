// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package sandbox

import (
	"testing"
	"github.com/petar/GoDCCP/dccp"
)

// roundtripReducer is a dccp.Guzzle which listens to the logs
// emitted from the RTT test and performs various checks.
type roundtripReducer struct {
	t *testing.T
}

func newRoundtripReducer(t *testing.T) *roundtripReducer {
	return &roundtripReducer{t}
}

func (t *roundtripReducer) Write(r *dccp.LogRecord) {
}

func (t *roundtripReducer) Sync() error { 
	return nil 
}

func (t *roundtripReducer) Close() error { 
	return nil 
}
