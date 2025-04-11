package picalc

import (
	"fmt"
	"math"
	"math/big"
	"os"
	"sync"
	"sync/atomic"
)

// VERSION is the current version of the picalc package
const VERSION = "0.1.0"

// Pi represents a structure for storing and synchronizing
// computed digits of Pi
type Pi struct {
	digits    []int
	mutex     sync.RWMutex
	computed  atomic.Int64
	precision int64
}

// NewPi creates a new Pi calculator with specified precision
func NewPi(precision int64) *Pi {
	return &Pi{
		digits:    make([]int, precision+1), // +1 for the '3' digit
		precision: precision,
	}
}

// CalculatePi calculates decimal digits of Pi using Chudnovsky algorithm
func CalculatePi(precision int64, pi *Pi) {
	// For very small precisions, use hardcoded values
	if precision <= 10 {
		hardcodedPi := []int{3, 1, 4, 1, 5, 9, 2, 6, 5, 3}
		pi.mutex.Lock()
		for i := 0; i < len(hardcodedPi) && i < len(pi.digits); i++ {
			pi.digits[i] = hardcodedPi[i]
		}
		pi.mutex.Unlock()
		pi.computed.Store(pi.precision)
		return
	}

	// Calculate Pi using fixed precision algorithm
	decimalStr := calculatePiChudnovsky(precision)

	// Extract the digits
	pi.mutex.Lock()
	// First digit is 3
	pi.digits[0] = 3

	// Extract the decimal part (skip the "3." at the beginning)
	start := 2 // Skip "3."
	for i := 1; i <= int(precision) && i < len(pi.digits) && start < len(decimalStr); i++ {
		if decimalStr[start] >= '0' && decimalStr[start] <= '9' {
			pi.digits[i] = int(decimalStr[start] - '0')
		}
		start++
	}
	pi.mutex.Unlock()

	// Mark as completed
	pi.computed.Store(pi.precision)
}

// calculatePiChudnovsky calculates pi to specified precision using Chudnovsky algorithm
func calculatePiChudnovsky(precision int64) string {
	// Calculate number of terms needed (each term gives ~14.18 digits)
	terms := int64(float64(precision)/14.18) + 2

	// Set up constants for Chudnovsky algorithm
	A := big.NewInt(13591409)
	B := big.NewInt(545140134)
	C3_24 := big.NewInt(640320 * 640320 * 640320 / 24)

	// Set precision for big.Float operations
	floatPrec := uint(int(math.Ceil(math.Log2(10)*float64(precision))) + 100)

	// Use binary splitting to calculate the sum
	// P, Q, R are as defined in the Chudnovsky paper
	var Q, R *big.Int

	// For small calculations, use direct approach
	if precision < 100 {
		_, Q, R = binarySplitSerial(0, terms, A, B, C3_24)
	} else {
		// For larger calculations, use parallel approach
		_, Q, R = binarySplitParallel(0, terms, A, B, C3_24)
	}

	// Final calculation Pi = (426880 * sqrt(10005)) / (R/Q)
	// Convert to big.Float for division and square root
	sqrtArg := new(big.Float).SetPrec(floatPrec)
	sqrtArg.SetInt64(10005)

	sqrt10005 := new(big.Float).SetPrec(floatPrec)
	sqrt10005.Sqrt(sqrtArg)

	C := new(big.Float).SetPrec(floatPrec)
	C.SetInt64(426880)
	C.Mul(C, sqrt10005)

	// R/Q
	sum := new(big.Float).SetPrec(floatPrec)
	sumQ := new(big.Float).SetPrec(floatPrec)
	sumQ.SetInt(Q)

	sumR := new(big.Float).SetPrec(floatPrec)
	sumR.SetInt(R)

	sum.Quo(sumR, sumQ)

	// Pi = C / sum
	pi := new(big.Float).SetPrec(floatPrec)
	pi.Quo(C, sum)

	// Return as string with enough precision
	return pi.Text('f', int(precision)+10)
}

// binarySplitSerial computes the Chudnovsky series using binary splitting (serial version)
func binarySplitSerial(a, b int64, A, B, C3_24 *big.Int) (*big.Int, *big.Int, *big.Int) {
	// Base case: compute a single term
	if b-a == 1 {
		var P, Q, R *big.Int

		if a == 0 {
			// First term
			P = big.NewInt(1)
			Q = big.NewInt(1)
			R = new(big.Int).Set(A) // 13591409
		} else {
			// P(a) = (6a-5)(2a-1)(6a-1)
			P = big.NewInt(6*a - 5)
			P = P.Mul(P, big.NewInt(2*a-1))
			P = P.Mul(P, big.NewInt(6*a-1))

			// Q(a) = a^3 * C3_24
			Q = big.NewInt(a)
			Q = Q.Mul(Q, big.NewInt(a))
			Q = Q.Mul(Q, big.NewInt(a))
			Q = Q.Mul(Q, C3_24)

			// R(a) = P(a) * (A + B*a)
			term := new(big.Int).Mul(B, big.NewInt(a))
			term = term.Add(term, A)
			R = new(big.Int).Mul(P, term)

			// Alternate sign: (-1)^a
			if a%2 == 1 {
				R = R.Neg(R)
			}
		}

		return P, Q, R
	}

	// Recursive case: split the range
	m := (a + b) / 2
	P1, Q1, R1 := binarySplitSerial(a, m, A, B, C3_24)
	P2, Q2, R2 := binarySplitSerial(m, b, A, B, C3_24)

	// Combine the results
	// P = P1 * P2
	P := new(big.Int).Mul(P1, P2)

	// Q = Q1 * Q2
	Q := new(big.Int).Mul(Q1, Q2)

	// R = R1 * Q2 + P1 * R2
	R1Q2 := new(big.Int).Mul(R1, Q2)
	P1R2 := new(big.Int).Mul(P1, R2)
	R := new(big.Int).Add(R1Q2, P1R2)

	return P, Q, R
}

// binarySplitParallel computes the Chudnovsky series using binary splitting (parallel version)
func binarySplitParallel(a, b int64, A, B, C3_24 *big.Int) (*big.Int, *big.Int, *big.Int) {
	// For small ranges, use serial version
	if b-a <= 100 {
		return binarySplitSerial(a, b, A, B, C3_24)
	}

	// Split the range
	m := (a + b) / 2

	// Use goroutines for parallel computation
	var P1, Q1, R1, P2, Q2, R2 *big.Int
	var wg sync.WaitGroup

	// Calculate left half in parallel
	wg.Add(1)
	go func() {
		defer wg.Done()
		P1, Q1, R1 = binarySplitParallel(a, m, A, B, C3_24)
	}()

	// Calculate right half in this goroutine
	P2, Q2, R2 = binarySplitParallel(m, b, A, B, C3_24)

	// Wait for left half to complete
	wg.Wait()

	// Combine the results
	// P = P1 * P2
	P := new(big.Int).Mul(P1, P2)

	// Q = Q1 * Q2
	Q := new(big.Int).Mul(Q1, Q2)

	// R = R1 * Q2 + P1 * R2
	R1Q2 := new(big.Int).Mul(R1, Q2)
	P1R2 := new(big.Int).Mul(P1, R2)
	R := new(big.Int).Add(R1Q2, P1R2)

	return P, Q, R
}

// GetDigits returns the first n decimal digits of Pi
func (p *Pi) GetDigits(n int) []int {
	if n > len(p.digits) {
		n = len(p.digits)
	}

	p.mutex.RLock()
	result := make([]int, n)
	copy(result, p.digits[:n])
	p.mutex.RUnlock()

	return result
}

// GetProgress returns the percentage of computation completed
func (p *Pi) GetProgress() float64 {
	computed := p.computed.Load()
	divisor := float64(p.precision) / 14.0
	if divisor <= 0 {
		divisor = 1
	}
	progress := float64(computed) / divisor * 100.0

	// Ensure progress is between 0 and 99
	if progress > 99.0 {
		progress = 99.0
	}
	if progress < 0.0 {
		progress = 0.0
	}

	return progress
}

// WriteDigitsToFile writes Pi digits to a file
func WriteDigitsToFile(digits []int, filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating file: %v", err)
	}
	defer f.Close()

	// Write the initial 3.
	f.WriteString("3.")

	// Write digits in batches to avoid memory spikes
	const batchSize = 1000
	for i := 1; i < len(digits); i += batchSize {
		end := i + batchSize
		if end > len(digits) {
			end = len(digits)
		}

		for j := i; j < end; j++ {
			fmt.Fprint(f, digits[j])
		}
	}

	return nil
}
