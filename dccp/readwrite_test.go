// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"reflect"
	"testing"
)

var testHeaders = []*GenericHeader{
	&GenericHeader{
		SourcePort:   33,
		DestPort:     77,
		CCVal:        1,
		CsCov:        3,
		Type:         Ack,
		X:            true,
		SeqNo:        0x0000334455667788,
		AckNo:        0x0000112233445566,
		ServiceCode:  0,
		Reset:        nil,
		Options:      []Option{
			Option{OptionSlowReceiver, nil, true},
		},
		Data:         []byte{1,2,3,0,4,5,6,7,8,9},
	},
}

func TestReadWrite(t *testing.T) {
	for _, gh := range testHeaders {
		h, d, err := gh.Write([]byte{1,2,3,4}, []byte{5,6,7,8}, 34, false)
		if err != nil {
			t.Errorf("write error: %s", err)
		}
		hd := make([]byte, len(h) + len(d))
		copy(hd, h)
		copy(hd[len(h):], d)
		gh2, err := ReadGenericHeader(hd, []byte{1,2,3,4}, []byte{5,6,7,8}, 34, false)
		if err != nil {
			t.Errorf("read error: %s", err)
		}
		diff(t, "** ", gh2, gh)
	}
}

func diff(t *testing.T, prefix string, have, want interface{}) {
	hv := reflect.NewValue(have).(*reflect.PtrValue).Elem().(*reflect.StructValue)
	wv := reflect.NewValue(want).(*reflect.PtrValue).Elem().(*reflect.StructValue)
	if hv.Type() != wv.Type() {
		t.Errorf("%s: type mismatch %v vs %v", prefix, hv.Type(), wv.Type())
	}
	for i := 0; i < hv.NumField(); i++ {
		hf := hv.Field(i).Interface()
		wf := wv.Field(i).Interface()
		if !reflect.DeepEqual(hf, wf) {
			t.Errorf("%s: %s = %v want %v", prefix, hv.Type().(*reflect.StructType).Field(i).Name, hf, wf)
		}
	}
}
