// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"fmt"
	"github.com/petar/GoGauge/gauge"
)

type Logger string

var NoLogging Logger = ""

func (t Logger) GetName() string {
	return string(t)
}

func (t Logger) GetState() string {
	g := gauge.GetAttr([]string{t.GetName()}, "state")
	if g == nil {
		return ""
	}
	return g.(string)
}

func (t Logger) SetState(s int) {
	gauge.SetAttr([]string{t.GetName()}, "state", StateString(s))
}

func (t Logger) Logf(modifier string, typ string, format string, v ...interface{}) {
	if t == "" {
		return
	}
	if !gauge.Selected(t.GetName(), modifier) {
		return
	}
	fmt.Printf("%d  @%-8s  %6s:%-11s  %-5s  ——  %s\n", 
		runtime.Time.Nanoseconds(), t.GetState(), t.GetName(), modifier,
		typ, fmt.Sprintf(format, v...),
	)
}
