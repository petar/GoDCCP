package main

import (
	"container/list"
)

type SparseSeries struct {
	sparse map[string]*list.List
}

func (x *SparseSeries) Add(series string, value float64) {
}
