// Copyright 2011-2013 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
)

var testHeaders = []*Header{
	&Header{
		SourcePort:  33,
		DestPort:    77,
		CCVal:       1,
		CsCov:       CsCov8,
		Type:        Ack,
		X:           true,
		SeqNo:       0x0000334455667788,
		AckNo:       0x0000112233445566,
		ServiceCode: 0,
		ResetCode:   0,
		ResetData:   nil,
		Options: []*Option{
			&Option{OptionSlowReceiver, make([]byte, 0), true},
		},
		Data: []byte{1, 2, 3, 0, 4, 5, 6, 7, 8, 9},
	},
}
// Expected wire representation:
//
// Source Port  00·21·
// Dest Port    00·4d·
// Data Offset  07·
// CCVal/CsCov  13·
// Checksum     38·13·
// Res/Type/X   07·
// Reserved     00·
// SeqNo        33·44·55·66·77·88·
// AckNo        00·00·11·22·33·44·55·66·
// Options      01·02·00·00·
// Data         01·02·03·00·04·05·06·07·08·09

func TestReadWrite(t *testing.T) {
	for _, gh := range testHeaders {
		hd, err := gh.Write([]byte{1, 2, 3, 4}, []byte{5, 6, 7, 8}, 34, false)
		if err != nil {
			t.Errorf("write error: %s", err)
		}
		fmt.Println(dumpBytes(hd))
		gh2, err := ReadHeader(hd, []byte{1, 2, 3, 4}, []byte{5, 6, 7, 8}, 34, false)
		if err != nil {
			t.Errorf("read error: %s", err)
		} else {
			diff(t, "** ", gh2, gh)
		}
	}
}

func dumpBytes(bb []byte) string {
	var w bytes.Buffer
	for _, b := range bb {
		fmt.Fprintf(&w, "%02x·", b)
	}
	return string(w.Bytes())
}

func diff(t *testing.T, prefix string, have, want interface{}) {
	hv := reflect.ValueOf(have).Elem()
	wv := reflect.ValueOf(want).Elem()
	if hv.Type() != wv.Type() {
		t.Errorf("%s: type mismatch %v vs %v", prefix, hv.Type(), wv.Type())
	}
	for i := 0; i < hv.NumField(); i++ {
		hf := hv.Field(i).Interface()
		wf := wv.Field(i).Interface()
		if !reflect.DeepEqual(hf, wf) {
			t.Errorf("%s: %s = %v want %v", prefix, hv.Type().Field(i).Name, hf, wf)
		}
	}
}
