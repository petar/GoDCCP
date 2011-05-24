// Copyright 2010 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package dccp

import (
	"rand"
	"testing"
)

const (
	K                         = 10
	NanoTestRange             = 9223372036854775807 / 10
	TimestampDeltaRangeInNano = 210010010 * TenMicroInNano
)

func TestTimestampOption(t *testing.T) {
	for i := 0; i < K; i++ {
		t0 := rand.Int63n(NanoTestRange)
		delta := rand.Int63n(TimestampDeltaRangeInNano)
		t1 := t0 + delta
		opt0 := NewTimestampOption(t0)
		opt1 := NewTimestampOption(t1)
		opt0_ := ValidateTimestampOption(opt0)
		opt1_ := ValidateTimestampOption(opt1)
		t0_ := opt0_.GetTimestamp()
		t1_ := opt1_.GetTimestamp()
		delta_ := GetTimestampDiff(t0_, t1_)
		if delta != delta_ {
			t.Errorf("Expecting %d, got %d", delta, delta_)
		}
	}
}

func TestElapsedTimeOption(t *testing.T) {
	for i := 0; i < K; i++ {
		e := rand.Int63n(NanoTestRange)
		opt := NewElapsedTimeOption(e)
		opt_ := ValidateElapsedTimeOption(opt)
		e_ := opt_.GetElapsed()
		if e != e_ {
			t.Errorf("Expecting %d, got %d", e, e_)
		}
	}
}

func TestTimestampEchoOption(t *testing.T) {
	for i := 0; i < K; i++ {
		t0 := rand.Int63n(NanoTestRange)
		delta := rand.Int63n(TimestampDeltaRangeInNano)
		t1 := t0 + delta
		opt0 := NewTimestampOption(t0)
		opt1 := NewTimestampOption(t1)

		e0 := rand.Int63n(NanoTestRange)
		e1 := rand.Int63n(NanoTestRange)
		ech0 := NewTimestampEchoOption(opt0, e0)
		ech1 := NewTimestampEchoOption(opt1, e1)

		ech0_ := ValidateTimestampOption(ech0)
		ech1_ := ValidateTimestampOption(ech1)
		t0_ := ech0_.GetTimestamp()
		t1_ := ech1_.GetTimestamp()

		delta_ := GetTimestampDiff(t0_, t1_)
		if delta != delta_ {
			t.Errorf("delta: expecting %d, got %d", delta, delta_)
		}

		e0_ := ech0_.GetElapsed()
		e1_ := ech1_.GetElapsed()
		if e0 != e0_ {
			t.Errorf("e0: expecting %d, got %d", e0, e0_)
		}
		if e1 != e1_ {
			t.Errorf("e1: expecting %d, got %d", e1, e1_)
		}
	}
}
