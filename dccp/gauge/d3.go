// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package gauge

import (
	"json"
	"os"
	"unsafe"
	"github.com/petar/GoDCCP/dccp"
)

// D3 is a dccp.LogEmitter which processes the log stream into a JSON structure convenient
// for input into a D3 visualization.
type D3 struct {
	reducer LogReducer
}

func NewD3() *D3 {
	t := &D3{}
	t.Init()
	return t
}

func (t *D3) Init() {
	t.reducer.Init()
}

func (t *D3) Emit(r *dccp.LogRecord) {
	t.reducer.Emit(r)
}

// D3Data is a JSON structure passed to the D3-based visualizer
type D3Data struct {
	CheckIns  []*D3CheckIn  `json:"check_ins"`
}

type D3CheckIn struct {
	Time      int64   `json:"time"`
	SeqNo     int64   `json:"seqno"`
	AckNo     int64   `json:"ackno"`
	Place     string  `json:"place"`
	Submodule string  `json:"sub"`
	Type      string  `json:"type"`
	State     string  `json:"state"`
	Comment   string  `json:"comment"`
}

func (t *D3) Close() *D3Data {
	d := &D3Data{}
	checkins := t.reducer.CheckIns()
	d.CheckIns = make([]*D3CheckIn, len(checkins))
	for i, rec := range checkins {
		d.CheckIns[i] = (*D3CheckIn)(unsafe.Pointer(rec))
	}
	return d
}

func OutToFile(env string, doc interface{}) error {
	name := os.Getenv(env)
	if name == "" {
		return nil
	}
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	defer f.Close()
	buf, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	if _, err = f.WriteString("data = "); err != nil {
		panic("export")
	}
	_, err = f.Write(buf)
	if err != nil {
		return err
	}
	if _, err = f.WriteString(";"); err != nil {
		panic("export")
	}
	return nil
}
