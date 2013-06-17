package sandbox

import (
	"github.com/petar/GoDCCP/dccp"
)

type SyncTraceWriter struct {
	wrapped dccp.TraceWriter
	queue chan dccp.Trace
	closed chan bool
}

func NewSyncTraceWriter(t dccp.TraceWriter) dccp.TraceWriter {
	w :=  &SyncTraceWriter{}
	w.wrapped = t
	w.queue = make(chan dccp.Trace)
	w.closed = make(chan bool)
	go w.writeLoop()
	return w
}

func (w *SyncTraceWriter) writeLoop() {
	for {
		t, ok := <-w.queue
		if !ok {
			w.closed <-true
			return
		}
		w.wrapped.Write(&t)
	}
}

func (w *SyncTraceWriter) Write(t *dccp.Trace) {
	w.queue <- *t
}

func (w *SyncTraceWriter) Sync() error { 
	return w.wrapped.Sync()
}

func (w *SyncTraceWriter) Close() error {
	close(w.queue)
	<-w.closed
	return w.wrapped.Close()
}
