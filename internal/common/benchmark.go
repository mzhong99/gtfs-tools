package common

import (
	"fmt"
	"time"
)

func RuntimeBenchmark[T any](label string, functionUnderTest func() (T, error)) (T, error) {
	start := time.Now()
	result, err := functionUnderTest()
	elapsed := time.Since(start)
	fmt.Printf("[BENCH] %s took %s\n", label, elapsed)
	return result, err
}
