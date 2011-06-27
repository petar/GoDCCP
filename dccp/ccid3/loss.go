// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import (
	"os"
	"time"
	"github.com/petar/GoDCCP/dccp"
)

type lossEvents struct {
}

func (t *lossIntervalTracker) OnRead(htype byte, x bool, seqno int64, ccval byte, options []*dccp.Option) os.Error {
}

func (t *lossIntervalTracker) Option() *Option {
	?
}

