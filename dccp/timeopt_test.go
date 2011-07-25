// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"rand"
	"testing"
)

const (
	K              = 20
	TimestampRange = 2^31 / 10
	ElapsedRange   = 210010010
)

func TestTimestampOption(t *testing.T) {
	for i := 0; i < K; i++ {
		t0 := uint32(rand.Int31n(TimestampRange))
		delta := uint32(rand.Int31n(ElapsedRange))
		t1 := t0 + delta
		opt0 := &TimestampOption{t0}
		opt1 := &TimestampOption{t1}
		opt0_, err1 := opt0.Encode()
		opt1_, err2 := opt1.Encode()
		if err1 != nil || err2 != nil {
			t.Fatalf("error encoding timestamp option")
		}
		opt0 = DecodeTimestampOption(opt0_)
		opt1 = DecodeTimestampOption(opt1_)
		if opt0 == nil || opt1 == nil {
			t.Fatalf("decoding timestamp error")
		}
		t0_ := opt0.Timestamp
		t1_ := opt1.Timestamp
		delta_ := TenMicroTimeDiff(t0_, t1_)
		if delta != delta_ {
			t.Errorf("Expecting %d, got %d", delta, delta_)
		}
	}
}

func TestElapsedTimeOption(t *testing.T) {
	for i := 0; i < K; i++ {
		e := uint32(rand.Int31n(TimestampRange))
		opt := &ElapsedTimeOption{e}
		opt_, err := opt.Encode()
		if err != nil {
			t.Fatalf("error encoding elapsed time option")
		}
		opt = DecodeElapsedTimeOption(opt_)
		if opt == nil {
			t.Fatalf("error decoding elapsed time option")
		}
		e_ := opt.Elapsed
		if e != e_ {
			t.Errorf("Expecting %d, got %d", e, e_)
		}
	}
}

func TestTimestampEchoOption(t *testing.T) {
	for i := 0; i < K; i++ {
		t0 := uint32(rand.Int31n(TimestampRange))
		delta := uint32(rand.Int31n(ElapsedRange))
		t1 := t0 + delta
		e0 := uint32(rand.Int31n(TimestampRange))
		e1 := uint32(rand.Int31n(TimestampRange))
		ech0 := &TimestampEchoOption{t0, e0}
		ech1 := &TimestampEchoOption{t1, e1}

		ech0_, err1 := ech0.Encode()
		ech1_, err2 := ech1.Encode()
		if err1 != nil || err2 != nil {
			t.Fatalf("encoding timstamp echo option")
		}
		ech0 = DecodeTimestampEchoOption(ech0_)
		ech1 = DecodeTimestampEchoOption(ech1_)
		if ech0 == nil || ech1 == nil {
			t.Fatalf("decoding timestamp echo option")
		}
		t0_ := ech0.Timestamp
		t1_ := ech1.Timestamp

		delta_ := TenMicroTimeDiff(t0_, t1_)
		if delta != delta_ {
			t.Errorf("delta: expecting %d, got %d", delta, delta_)
		}

		e0_ := ech0.Elapsed
		e1_ := ech1.Elapsed
		if e0 != e0_ {
			t.Errorf("e0: expecting %d, got %d", e0, e0_)
		}
		if e1 != e1_ {
			t.Errorf("e1: expecting %d, got %d", e1, e1_)
		}
	}
}
