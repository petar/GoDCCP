// Copyright 2011-2013 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"fmt"
	"runtime"
	"encoding/json"
	"os"
	"sync"
)

// FileTraceWriter saves all log entries to a file in JSON format
type FileTraceWriter struct {
	sync.Mutex
	f   *os.File
	enc *json.Encoder
	dup TraceWriter
}

// NewFileTraceWriterDup creates a TraceWriter that saves logs in a file and also passes them to dup.
func NewFileTraceWriterDup(filename string, dup TraceWriter) *FileTraceWriter {
	os.Remove(filename)
	f, err := os.Create(filename)
	if err != nil {
		panic(fmt.Sprintf("cannot create log file '%s'", filename))
	}
	w := &FileTraceWriter{ f:f, enc:json.NewEncoder(f), dup:dup }
	runtime.SetFinalizer(w, func(w *FileTraceWriter) { 
		w.f.Close() 
	})
	return w
}

func NewFileTraceWriter(filename string) *FileTraceWriter {
	return NewFileTraceWriterDup(filename, nil)
}

func (t *FileTraceWriter) Write(r *Trace) {
	t.Lock()
	err := t.enc.Encode(r)
	t.Unlock()
	if err != nil {
		panic(fmt.Sprintf("error encoding log entry (%s)", err))
	}
	if t.dup != nil {
		t.dup.Write(r)
	}
}

func (t *FileTraceWriter) Sync() error {
	if t.dup != nil {
		t.dup.Sync()
	}
	return t.f.Sync()
}

func (t *FileTraceWriter) Close() error {
	if t.dup != nil {
		t.dup.Close()
	}
	return t.f.Close()
}
