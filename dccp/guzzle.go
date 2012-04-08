// Copyright 2011 GoDCCP Authors. All rights reserved.
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

// Guzzle is a type that consumes log entries.
type Guzzle interface {
	Write(*LogRecord)
	Sync() error
	Close() error
}

// FileGuzzle saves all log entries to a file in JSON format
type FileGuzzle struct {
	sync.Mutex
	f   *os.File
	enc *json.Encoder
	dup Guzzle
}

// NewFileGuzzleDup creates a Guzzle that saves logs in a file and also passes them to dup.
func NewFileGuzzleDup(filename string, dup Guzzle) *FileGuzzle {
	os.Remove(filename)
	f, err := os.Create(filename)
	if err != nil {
		panic(fmt.Sprintf("cannot create log file '%s'", filename))
	}
	w := &FileGuzzle{ f:f, enc:json.NewEncoder(f), dup:dup }
	runtime.SetFinalizer(w, func(w *FileGuzzle) { 
		w.f.Close() 
	})
	return w
}

func NewFileGuzzle(filename string) *FileGuzzle {
	return NewFileGuzzleDup(filename, nil)
}

func (t *FileGuzzle) Write(r *LogRecord) {
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

func (t *FileGuzzle) Sync() error {
	if t.dup != nil {
		t.dup.Sync()
	}
	return t.f.Sync()
}

func (t *FileGuzzle) Close() error {
	if t.dup != nil {
		t.dup.Close()
	}
	return t.f.Close()
}
