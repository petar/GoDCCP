// Copyright 2011-2013 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package sandbox

import (
	"github.com/petar/GoDCCP/dccp"
)

// GuzzlePlex is a dccp.Guzzle that replicates Guzzle method invocations to a set of Guzzles
type GuzzlePlex struct {
	guzzles   []dccp.Guzzle
	highlight []string
}

func NewGuzzlePlex(guzzles ...dccp.Guzzle) *GuzzlePlex {
	return &GuzzlePlex{
		guzzles: guzzles,
	}
}

// HighlightSamples instructs the guzzle to highlight any records carrying samples of the given names
func (t *GuzzlePlex) HighlightSamples(samples ...string) {
	t.highlight = samples
}

// Add adds an additional guzzle to the plex
func (t *GuzzlePlex) Add(g dccp.Guzzle) {
	t.guzzles = append(t.guzzles, g)
}

func (t *GuzzlePlex) Write(r *dccp.Trace) {
	sample, ok := r.Sample()
	if ok {
		for _, hi := range t.highlight {
			if sample.Series == hi {
				r.SetHighlight()
				break
			}
		}
	}
	for _, g := range t.guzzles {
		g.Write(r)
	}
}

// Sync syncs all the guzzles in the plex
func (t *GuzzlePlex) Sync() error {
	var err error
	for _, g := range t.guzzles {
		e := g.Sync()
		if err != nil {
			err = e
		}
	}
	return err
}

// Close closes all the guzzles in the plex
func (t *GuzzlePlex) Close() error {
	var err error
	for _, g := range t.guzzles {
		e := g.Close()
		if err != nil {
			err = e
		}
	}
	return err
}
