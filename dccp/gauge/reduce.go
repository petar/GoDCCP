// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package gauge

import (
	"github.com/petar/GoDCCP/dccp"
)

// LogReducer is a dccp.LogEmitter which processes the logs to a form
// that is convenient to illustrate via tools like D3 (Data-Driven Design).
type LogReducer struct {
}

func NewLogReducer() *LogReducer {
	return &LogReducer{}
}

func (t *LogReducer) Emit(r *dccp.LogRecord) {
}
