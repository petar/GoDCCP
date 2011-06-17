// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package retransmit

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
)

var testHeaders = []*header{
	&header{
		Sync:   true,
		SyncNo: 1234,

		Ack:       true,
		AckSyncNo: 5678,
		AckDataNo: 43218765,
		AckMap:    []byte{1, 2, 3, 4},

		Data:      true,
		DataNo:    333355551,
		DataCargo: []byte{1, 1, 2, 2, 3, 3, 4, 4},
	},
}

func TestReadWrite(t *testing.T) {
	for _, h := range testHeaders {
		b, err := h.Write()
		if err != nil {
			t.Errorf("write error: %s", err)
		}
		fmt.Println(dumpBytes(b))
		g, err := readHeader(b)
		if err != nil {
			t.Errorf("read error: %s", err)
		} else {
			diff(t, "** ", g, h)
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
