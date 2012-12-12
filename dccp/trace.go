// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"bytes"
	"fmt"
	"runtime"
)

func Caller() string {
	pc, file, line, ok := runtime.Caller(2)
	if !ok {
		return ""
	}
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return fmt.Sprintf("(%s:%d)", file, line)
	}
	return fmt.Sprintf("%-61s  (%s:%d)", fn.Name(), file, line)
}

// StackTrace formats the stack trace of the calling goroutine, 
// excluding pointer information and including DCCP runtime-specific information, 
// in a manner convenient for debugging DCCP
func StackTrace(labels []string, skip int, sfile string, sline int) string {
	var w bytes.Buffer
	var stk []uintptr = make([]uintptr, 32)	// DCCP logic stack should not be deeper than that
	n := runtime.Callers(skip+1, stk)
	stk = stk[:n]
	var utf2byte int
	for _, l := range labels {
		fmt.Fprintf(&w, "%s·", l)
		utf2byte++
	}
	for w.Len() < 40 + 4 + utf2byte {
		w.WriteRune(' ')
	}
	fmt.Fprintf(&w, " (%s:%d)\n", sfile, sline)
	var external bool
	for _, pc := range stk {
		f := runtime.FuncForPC(pc)
		if f == nil {
			break
		}
		file, line := f.FileLine(pc)
		fname, isdccp := TrimFuncName(f.Name())
		if !isdccp {
			external = true
		} else {
			if external {
				fmt.Fprintf(&w, "    ···· ···· ···· \n")
			}
			fmt.Fprintf(&w, "    %-40s (%s:%d)\n", fname, TrimSourceFile(file), line)
		}
	}
	return string(w.Bytes())
}

