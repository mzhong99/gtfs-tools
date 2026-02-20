package common

import (
	"fmt"
	"time"
)

type Benchmarker struct {
	start time.Time
	label string
}

func RuntimeBenchmark[T any](label string, functionUnderTest func() (T, error)) (T, error) {
	start := time.Now()
	result, err := functionUnderTest()
	elapsed := time.Since(start)
	fmt.Printf("[BENCH] %s took %s\n", label, elapsed)
	return result, err
}

func NewBenchmarker(label string) *Benchmarker {
	return &Benchmarker{time.Now(), label}
}

func (benchmarker *Benchmarker) Close() {
	elapsed := time.Since(benchmarker.start)
	fmt.Printf("[BENCH] %s took %s\n", benchmarker.label, elapsed)
}
