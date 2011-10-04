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
	sinceZero, sinceLast := SnapLog()
	fmt.Printf("%15s %15s  @%-8s  %6s:%-11s  %-5s  ——  %s\n", 
		nstoa(sinceZero), nstoa(sinceLast), t.GetState(), t.GetName(), modifier,
		typ, fmt.Sprintf(format, v...),
	)
}

const nsAlpha = "0123456789"

func nstoa(ns int64) string {
	if ns < 0 {
		panic("negative time")
	}
	if ns == 0 {
		return "0"
	}
	b := make([]byte, 32)
	z := len(b) - 1
	i := 0
	j := 0
	for ns != 0 {
		if j % 3 == 0 && j > 0 {
			b[z-i] = ','
			i++
		}
		b[z-i] = nsAlpha[ns % 10]
		j++
		i++
		ns /= 10
	}
	return string(b[z-i+1:])
}
