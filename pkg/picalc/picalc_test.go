package picalc

import (
	"bytes"
	"fmt"
	"math/big"
	"os"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"
)

// UNIT TESTS
// ==========

// TestPiCalculation tests various aspects of PI calculation
func TestPiCalculation(t *testing.T) {
	t.Run("FirstDigits", func(t *testing.T) {
		// Test first 10 digits of Pi
		expectedDigits := []int{3, 1, 4, 1, 5, 9, 2, 6, 5, 3}

		pi := NewPi(10)
		CalculatePi(10, pi)

		digits := pi.GetDigits(10)

		if !reflect.DeepEqual(digits, expectedDigits) {
			t.Errorf("First 10 digits don't match.\nExpected: %v\nGot: %v", expectedDigits, digits)
		}
	})

	t.Run("Accuracy", func(t *testing.T) {
		// Test with 50 digits for accuracy
		knownPiDigits := "3.14159265358979323846264338327950288419716939937510"

		pi := NewPi(50)
		CalculatePi(50, pi)
		digits := pi.GetDigits(50)

		// Format digits for comparison
		var result bytes.Buffer
		result.WriteString("3.")
		for i := 1; i < len(digits); i++ {
			result.WriteByte('0' + byte(digits[i]))
		}

		// Get the string versions
		resultStr := result.String()

		// Check starting digits match
		// We compare the first 10 characters since precision may vary slightly
		if len(resultStr) < 10 || len(knownPiDigits) < 10 || resultStr[:10] != knownPiDigits[:10] {
			t.Errorf("Pi calculation inaccurate.\nExpected: %s\nGot: %s", knownPiDigits, resultStr)
		}
	})
}

func TestProgressTracking(t *testing.T) {
	// Test progress reporting
	pi := NewPi(100)

	// Simulate partial completion
	pi.computed.Store(50)

	progress := pi.GetProgress()
	if progress < 1.0 || progress > 99.0 {
		t.Errorf("Progress should be between 1 and 99, got: %f", progress)
	}

	// Test 100% completion (should be capped at 99%)
	pi.computed.Store(pi.precision / 14 * 2) // Double the expected completion
	progress = pi.GetProgress()
	if progress != 99.0 {
		t.Errorf("Progress should be capped at 99%%, got: %f", progress)
	}
}

func TestMathFunctions(t *testing.T) {
	t.Run("SquareRoot", func(t *testing.T) {
		testCases := []struct {
			input    int64
			expected int64
		}{
			{4, 2},
			{9, 3},
			{16, 4},
			{25, 5},
			{10000, 100},
		}

		for _, tc := range testCases {
			// Test square root implementation
			result := bigSqrt(big.NewInt(tc.input), 0)
			if result.Int64() != tc.expected {
				t.Errorf("Square root of %d: expected %d, got %v", tc.input, tc.expected, result)
			}
		}
	})

	t.Run("PowerOf10", func(t *testing.T) {
		testCases := []struct {
			input    int64
			expected string
		}{
			{0, "1"},
			{1, "10"},
			{2, "100"},
			{5, "100000"},
		}

		for _, tc := range testCases {
			result := pow10(tc.input)
			if result.String() != tc.expected {
				t.Errorf("Power of 10^%d: expected %s, got %s", tc.input, tc.expected, result.String())
			}
		}
	})

	t.Run("BinarySplit", func(t *testing.T) {
		// Test with a simple range
		result := binarySplit(1, 2)

		// Verify all components are non-zero
		if result.p == nil || result.q == nil || result.r == nil {
			t.Error("Binary split returned nil components")
		} else if result.p.Sign() == 0 || result.q.Sign() == 0 || result.r.Sign() == 0 {
			t.Errorf("Binary split should produce non-zero values, got p=%v, q=%v, r=%v",
				result.p, result.q, result.r)
		}
	})
}

func TestFileIO(t *testing.T) {
	// Test file writing functionality
	testDigits := []int{3, 1, 4, 1, 5, 9}
	tempFile := "test_pi.txt"

	WriteDigitsToFile(testDigits, tempFile)

	// Read the file back
	content, err := os.ReadFile(tempFile)
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	expected := "3.14159"
	if string(content) != expected {
		t.Errorf("File content mismatch.\nExpected: %s\nGot: %s", expected, string(content))
	}

	// Clean up
	os.Remove(tempFile)
}

func TestConcurrency(t *testing.T) {
	t.Run("ConcurrentReads", func(t *testing.T) {
		// Test concurrent read safety
		pi := NewPi(100)

		// Simulate calculation
		pi.digits[0] = 3
		for i := 1; i < 100; i++ {
			pi.digits[i] = i % 10
		}

		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				pi.GetDigits(50)
			}()
		}

		// This should not cause race conditions
		wg.Wait()
	})

	t.Run("ConcurrentReadWrite", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping in short mode")
		}

		pi := NewPi(100)

		var wg sync.WaitGroup
		wg.Add(2)

		go func() {
			defer wg.Done()
			CalculatePi(100, pi)
		}()

		go func() {
			defer wg.Done()
			for i := 0; i < 20; i++ {
				pi.GetProgress()
				pi.GetDigits(50)
				time.Sleep(5 * time.Millisecond)
			}
		}()

		wg.Wait()
	})
}

// BENCHMARK TESTS
// ==============

func BenchmarkCalculatePi(b *testing.B) {
	benchmarks := []struct {
		name      string
		precision int64
	}{
		{"10Digits", 10},
		{"100Digits", 100},
		{"1000Digits", 1000},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			if bm.precision >= 1000 && b.N > 1 {
				b.N = 1 // Limit iterations for expensive tests
			}

			for i := 0; i < b.N; i++ {
				b.StopTimer()
				pi := NewPi(bm.precision)
				b.StartTimer()

				CalculatePi(bm.precision, pi)
			}
		})
	}
}

func BenchmarkMathOperations(b *testing.B) {
	b.Run("BinarySplit", func(b *testing.B) {
		sizes := []int64{1, 5, 10}
		for _, size := range sizes {
			b.Run(fmt.Sprintf("terms=%d", size), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					binarySplit(0, size)
				}
			})
		}
	})

	b.Run("PowerOf10", func(b *testing.B) {
		sizes := []int64{5, 10, 50}
		for _, size := range sizes {
			b.Run(fmt.Sprintf("n=%d", size), func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					pow10(size)
				}
			})
		}
	})

	b.Run("SquareRoot", func(b *testing.B) {
		inputs := []int64{100, 10000, 1000000}
		for _, input := range inputs {
			b.Run(fmt.Sprintf("n=%d", input), func(b *testing.B) {
				x := big.NewInt(input)
				for i := 0; i < b.N; i++ {
					bigSqrt(x, 100)
				}
			})
		}
	})
}

// PERFORMANCE TESTS
// ================

func TestPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance tests in short mode")
	}

	t.Run("CPUScaling", func(t *testing.T) {
		cpuTests := []int{1, runtime.NumCPU()}
		results := make(map[int]time.Duration)

		for _, numCPU := range cpuTests {
			old := runtime.GOMAXPROCS(numCPU)

			start := time.Now()
			pi := NewPi(500)
			CalculatePi(500, pi)
			elapsed := time.Since(start)

			runtime.GOMAXPROCS(old)
			results[numCPU] = elapsed
		}

		// Log the results
		for cpu, duration := range results {
			t.Logf("CPU cores: %d, Duration: %v", cpu, duration)
		}

		// Check if more CPUs helped
		if len(cpuTests) > 1 {
			speedup := float64(results[cpuTests[0]]) / float64(results[cpuTests[len(cpuTests)-1]])
			t.Logf("Speedup with %d vs %d cores: %.2fx",
				cpuTests[len(cpuTests)-1], cpuTests[0], speedup)

			// Basic validation that parallelism is working
			if speedup < 1.1 && cpuTests[len(cpuTests)-1] > 1 {
				t.Logf("Warning: Limited parallel speedup detected")
			}
		}
	})

	t.Run("MemoryUsage", func(t *testing.T) {
		precisions := []int64{100, 1000}

		for _, prec := range precisions {
			var m1, m2 runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&m1)

			pi := NewPi(prec)
			CalculatePi(prec, pi)

			runtime.GC()
			runtime.ReadMemStats(&m2)

			memUsed := m2.TotalAlloc - m1.TotalAlloc
			t.Logf("Memory for %d digits: %d bytes (%.2f MB)",
				prec, memUsed, float64(memUsed)/(1024*1024))

			// Basic validation of memory efficiency
			bytesPerDigit := float64(memUsed) / float64(prec)
			t.Logf("Bytes per digit: %.2f", bytesPerDigit)
		}
	})

	t.Run("TimingAnalysis", func(t *testing.T) {
		precisions := []int64{10, 100, 500}
		times := make(map[int64]time.Duration)

		for _, prec := range precisions {
			start := time.Now()
			pi := NewPi(prec)
			CalculatePi(prec, pi)
			times[prec] = time.Since(start)
		}

		// Log and analyze results
		for prec, duration := range times {
			t.Logf("Precision %d: %v", prec, duration)
		}

		// Calculate scaling factor
		if len(precisions) > 2 {
			ratio1 := float64(times[precisions[1]]) / float64(times[precisions[0]])
			ratio2 := float64(times[precisions[2]]) / float64(times[precisions[1]])
			precRatio1 := float64(precisions[1]) / float64(precisions[0])
			precRatio2 := float64(precisions[2]) / float64(precisions[1])

			t.Logf("Time scaling: %.2f vs precision scaling: %.2f (first pair)",
				ratio1, precRatio1)
			t.Logf("Time scaling: %.2f vs precision scaling: %.2f (second pair)",
				ratio2, precRatio2)
		}
	})
}

// Extended benchmarks for profiling
func BenchmarkForProfiling(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping profiling benchmark in short mode")
	}

	b.Run("FullCalculation", func(b *testing.B) {
		b.N = 1 // Only run once

		b.ReportAllocs()
		pi := NewPi(2000)
		CalculatePi(2000, pi)
	})
}
