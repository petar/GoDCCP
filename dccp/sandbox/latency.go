package sandbox

import (
	"sort"
	"github.com/petar/GoDCCP/dccp"
)

type latencyQueue struct {
	run         *dccp.Runtime
	amb         *dccp.Amb
	queue       []*pipeHeader
}

func (x *latencyQueue) Init(run *dccp.Runtime, amb *dccp.Amb) {
	x.run = run
	x.amb = amb
	x.queue = make([]*pipeHeader, 0)
}

func (x *latencyQueue) Add(ph *pipeHeader) {
	x.queue = append(x.queue, ph)
	sort.Sort(pipeHeaderTimeSort(x.queue))
}

func (x *latencyQueue) DeleteMin() *pipeHeader {
	if len(x.queue) == 0 {
		return nil
	}
	ph := x.queue[0]
	x.queue = x.queue[1:]
	return ph
}

func (x *latencyQueue) TimeToNext() int64 {
	if len(x.queue) == 0 {
		return -1
	}
	now := x.run.Nanoseconds()
	return max64(0, x.queue[0].DeliverTime - now)
}

func max64(x, y int64) int64 {
	if x > y {
		return x
	}
	return y
}

// pipeHeaderTimeSort sorts a slice of *pipeHeader by timestamp
type pipeHeaderTimeSort []*pipeHeader

func (t pipeHeaderTimeSort) Len() int {
	return len(t)
}

func (t pipeHeaderTimeSort) Less(i, j int) bool {
	return t[i].DeliverTime < t[j].DeliverTime
}

func (t pipeHeaderTimeSort) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}
