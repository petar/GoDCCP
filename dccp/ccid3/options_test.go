// Copyright 2011-2013 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import (
	"reflect"
	"testing"
)

var lossEventRateOptions = []*LossEventRateOption{
	&LossEventRateOption{
		RateInv: 0,
	},
	&LossEventRateOption{
		RateInv: UnknownLossEventRateInv,
	},
}

func TestLossEventRateOption(t *testing.T) {
	for _, original := range lossEventRateOptions {
		encoded, err := original.Encode()
		if err != nil {
			t.Fatalf("encoding option (%s)", err)
		}
		decoded := DecodeLossEventRateOption(encoded)
		if decoded == nil {
			t.Fatalf("decoding option")
		}
		if !reflect.DeepEqual(original, decoded) {
			t.Fatalf("expecting %v, encountered %v", original, decoded)
		}
	}
}

var lossIntervalsOptions = []*LossIntervalsOption{
	&LossIntervalsOption{
		SkipLength:    3,
		LossIntervals: nil,
	},
	&LossIntervalsOption{
		SkipLength:    0,
		LossIntervals: []*LossInterval{},
	},
	&LossIntervalsOption{
		SkipLength:    0,
		LossIntervals: []*LossInterval{
			&LossInterval{
				LosslessLength: 10,
				LossLength:     5,
				DataLength:     15,
				ECNNonceEcho:   true,
			},
			&LossInterval{
				LosslessLength: 0,
				LossLength:     1,
				DataLength:     1,
				ECNNonceEcho:   false,
			},
		},
	},
}

func TestLostIntervalsOption(t *testing.T) {
	for _, original := range lossIntervalsOptions {
		encoded, err := original.Encode()
		if err != nil {
			t.Fatalf("encoding option (%s)", err)
		}
		decoded := DecodeLossIntervalsOption(encoded)
		if decoded == nil {
			t.Fatalf("decoding option")
		}
		if !reflect.DeepEqual(original, decoded) {
			t.Fatalf("expecting %v, encountered %v", original, decoded)
		}
	}
}

var receiveRateOptions = []*ReceiveRateOption{
	&ReceiveRateOption{
		Rate: 0,
	},
	&ReceiveRateOption{
		Rate: 15,
	},
}

func TestReceiveRateOption(t *testing.T) {
	for _, original := range receiveRateOptions {
		encoded, err := original.Encode()
		if err != nil {
			t.Fatalf("encoding option (%s)", err)
		}
		decoded := DecodeReceiveRateOption(encoded)
		if decoded == nil {
			t.Fatalf("decoding option")
		}
		if !reflect.DeepEqual(original, decoded) {
			t.Fatalf("expecting %v, encountered %v", original, decoded)
		}
	}
}
