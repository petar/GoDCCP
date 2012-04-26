package dccp

import (
	"fmt"
	"testing"
)

func A(r Runtime) {
	r.Go(func() { B(r) })
	r.Sleep(6000)
	fmt.Printf("A, now=%d\n", r.Now())
}

func B(r Runtime) {
	r.Sleep(3000)
	fmt.Printf("B, now=%d\n", r.Now())
}

func TestGoSynthetic(t *testing.T) {
	GoSynthetic(A)
}

func TestNewSyntheticRuntime(t *testing.T) {
	runtime := NewSyntheticRuntime()
	runtime.Go(func() { A(runtime) })
	runtime.Join()
}
