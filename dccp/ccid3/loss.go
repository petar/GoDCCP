// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package ccid3

import (
	"math"
)

// lossRateCalculator calculates the inverse of the loss event rate as
// specified in Section 5.4, RFC 5348. One instantiation can perform repeated
// calculations using a fixed nInterval parameter.
type lossRateCalculator struct {
	nInterval int
	w         []float64
	h         []float64
}

// Init resets the calculator for new use with the given nInterval parameter
func (t *lossRateCalculator) Init(nInterval int) {
	t.nInterval = nInterval
	t.w = make([]float64, nInterval)
	for i, _ := range t.w {
		t.w[i] = intervalWeight(i, nInterval)
	}
	t.h = make([]float64, nInterval)
}

func intervalWeight(i, nInterval int) float64 {
	if i < nInterval/2 {
		return 1.0
	}
	return 2.0 * float64(nInterval-i) / float64(nInterval+2)
}

// CalcLossEventRateInv computes the inverse of the loss event rate, RFC 5348, Section 5.4.
// NOTE: We currently don't use the alternative algorithm, called History Discounting,
// discussed in RFC 5348, Section 5.5
// TODO: This calculation should be replaced with an entirely integral one.
// TODO: Remove the most recent unfinished interval from the calculation, if too small. Not crucial.
func (t *lossRateCalculator) CalcLossEventRateInv(history []*LossIntervalDetail) uint32 {

	// Prepare a slice with interval lengths
	k := min(len(history), t.nInterval)
	if k < 2 {
		// Too few loss events are reported as UnknownLossEventRateInv which signifies 'no loss'
		return UnknownLossEventRateInv
	}
	h := t.h[:k]
	for i := 0; i < k; i++ {
		h[i] = float64(history[i].LossInterval.SeqLen())
	}

	// Directly from the RFC
	var I_tot0 float64 = 0
	var I_tot1 float64 = 0
	var W_tot float64 = 0
	for i := 0; i < k-1; i++ {
		I_tot0 += h[i] * t.w[i]
		W_tot += t.w[i]
	}
	for i := 1; i < k; i++ {
		I_tot1 += h[i] * t.w[i-1]
	}
	I_tot := math.Max(I_tot0, I_tot1)
	I_mean := I_tot / W_tot

	if I_mean < 1.0 {
		panic("invalid inverse")
	}
	return uint32(I_mean)
}
