// Copyright 2011-2013 GoDCCP Authors. All rights reserved.
// Use of this source code is governed by a 
// license that can be found in the LICENSE file.

package sandbox

import (
	"github.com/petar/GoDCCP/dccp"
)

// TraceWriterPlex is a dccp.TraceWriter that replicates TraceWriter method invocations to a set of TraceWriters
type TraceWriterPlex struct {
	guzzles   []dccp.TraceWriter
	highlight []string
}

func NewTraceWriterPlex(guzzles ...dccp.TraceWriter) *TraceWriterPlex {
	return &TraceWriterPlex{
		guzzles: guzzles,
	}
}

// HighlightSamples instructs the guzzle to highlight any records carrying samples of the given names
func (t *TraceWriterPlex) HighlightSamples(samples ...string) {
	t.highlight = samples
}

// Add adds an additional guzzle to the plex
func (t *TraceWriterPlex) Add(g dccp.TraceWriter) {
	t.guzzles = append(t.guzzles, g)
}

func (t *TraceWriterPlex) Write(r *dccp.Trace) {
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
func (t *TraceWriterPlex) Sync() error {
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
func (t *TraceWriterPlex) Close() error {
	var err error
	for _, g := range t.guzzles {
		e := g.Close()
		if err != nil {
			err = e
		}
	}
	return err
}
