// Copyright 2011 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package gauge

// TODO: Don't count drops after one side closes
func CalcRates(trips []*Trip) (sendRate float64, receiveRate float64) {
	var earliest, latest int64
	var sent int64
	var received int64
	for _, t := range trips {
		sent++
		if t.Round {
			received++
		}
		if earliest == 0 {
			earliest = t.Forward[0].Time
		} else {
			earliest = min64(earliest, t.Forward[0].Time)
		}
		if latest == 0 {
			latest = t.Forward[len(t.Forward)-1].Time
		} else {
			latest = max64(latest, t.Forward[len(t.Forward)-1].Time)
		}
	}
	d := float64(latest - earliest) / 1e9

	return float64(sent)/d, float64(received)/d
}

func max64(x, y int64) int64 {
	if x > y {
		return x
	}
	return y
}

func min64(x, y int64) int64 {
	if x < y {
		return x
	}
	return y
}
