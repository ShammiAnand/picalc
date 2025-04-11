package picalc

import (
	"fmt"
	"math/big"
	"os"
	"runtime"
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
		hardcodedPi := []int{3, 1, 4, 1, 5, 9, 2, 6, 5, 3, 5}
		pi.mutex.Lock()
		for i := 0; i <= int(precision) && i < len(hardcodedPi); i++ {
			pi.digits[i] = hardcodedPi[i]
		}
		pi.mutex.Unlock()
		pi.computed.Store(pi.precision)
		return
	}

	// Use a more efficient direct algorithm with sufficient decimal places
	computePiChudnovsky(precision, pi)
}

// computePiChudnovsky implements the Chudnovsky algorithm for calculating Pi
func computePiChudnovsky(precision int64, pi *Pi) {
	// Constants from the Chudnovsky formula
	a := big.NewInt(13591409)
	c := big.NewInt(640320)

	// Calculate number of terms needed (each term gives ~14 digits)
	terms := precision / 14
	if terms < 1 {
		terms = 1
	}

	// Setup big number calculations with extra precision
	// Need extra precision to avoid rounding errors
	digits := precision + 20

	// Use the Binary Splitting algorithm for the series calculation
	pqt := binarySplitMultiThread(0, terms, pi)

	// Calculate C * sqrt(10005)
	c3 := new(big.Int).Mul(c, c)
	c3.Mul(c3, c)

	// C = 426880 * sqrt(10005)
	cc := new(big.Int).SetInt64(10005)
	cc.Mul(big.NewInt(426880), bigSqrt(cc, digits))

	// Final Pi value: C * sqrt(10005) / (a + b*sum)
	// where sum is R/Q from binary splitting
	num := new(big.Int).Mul(a, pqt.q)
	num.Add(num, pqt.r)

	// Scale for decimal precision
	scaledC := new(big.Int).Mul(cc, pow10(digits))

	// Pi = C / sum
	piVal := new(big.Int).Div(scaledC, num)
	piStr := piVal.String()

	// Set the digits
	pi.mutex.Lock()
	// First digit is 3
	pi.digits[0] = 3

	// Extract decimal digits
	for i := 1; i <= int(precision) && i < len(pi.digits); i++ {
		if i-1 < len(piStr) {
			pi.digits[i] = int(piStr[i-1] - '0')
		} else {
			pi.digits[i] = 0
		}
	}
	pi.mutex.Unlock()
}

// PQT values for binary splitting
type pqt struct {
	p, q, r *big.Int
}

// binarySplitMultiThread performs binary splitting with multiple threads
func binarySplitMultiThread(a, b int64, pi *Pi) pqt {
	// For small ranges or on single processors, use single-threaded version
	if b-a < 100 || runtime.NumCPU() < 2 {
		return binarySplit(a, b)
	}

	// Divide the work
	mid := (a + b) / 2

	// Create channel for results
	resultChan := make(chan pqt, 2)

	// Calculate first half in a separate goroutine
	go func() {
		resultChan <- binarySplit(a, mid)
	}()

	// Calculate second half in current goroutine
	right := binarySplit(mid, b)

	// Get result from first half
	left := <-resultChan

	// Combine the results
	result := combinePQT(left, right)

	// Update progress
	compRange := b - a
	pi.computed.Add(compRange)

	return result
}

// binarySplit implements the binary splitting algorithm for Chudnovsky formula
func binarySplit(a, b int64) pqt {
	// Base case: single term
	if a+1 == b {
		// P(a,a+1) = (6a-5)*(2a-1)*(6a-1)
		p := big.NewInt(6*a - 5)
		p.Mul(p, big.NewInt(2*a-1))
		p.Mul(p, big.NewInt(6*a-1))

		// Q(a,a+1) = (10939058860032000) * a^3
		q := big.NewInt(a)
		q.Mul(q, big.NewInt(a))
		q.Mul(q, big.NewInt(a))
		q.Mul(q, big.NewInt(10939058860032000)) // 640320^3/24

		// R(a,a+1) = P(a,a+1) * (13591409 + 545140134*a)
		r := big.NewInt(545140134)
		r.Mul(r, big.NewInt(a))
		r.Add(r, big.NewInt(13591409))
		r.Mul(r, p)

		// Apply (-1)^k factor
		if a&1 == 1 { // If a is odd
			r.Neg(r)
		}

		return pqt{p: p, q: q, r: r}
	}

	// Recursive case
	mid := (a + b) / 2
	left := binarySplit(a, mid)
	right := binarySplit(mid, b)

	return combinePQT(left, right)
}

// combinePQT combines two PQT values according to binary splitting rules
func combinePQT(left, right pqt) pqt {
	// P = P1 * P2
	p := new(big.Int).Mul(left.p, right.p)

	// Q = Q1 * Q2
	q := new(big.Int).Mul(left.q, right.q)

	// R = Q2 * R1 + P1 * R2
	tmp1 := new(big.Int).Mul(right.q, left.r)
	tmp2 := new(big.Int).Mul(left.p, right.r)
	r := new(big.Int).Add(tmp1, tmp2)

	return pqt{p: p, q: q, r: r}
}

// bigSqrt calculates square root using big integers with sufficient precision
func bigSqrt(n *big.Int, prec int64) *big.Int {
	// Handle small values directly
	if n.Cmp(big.NewInt(1000000)) < 0 {
		// Use the built-in Sqrt for small values
		return new(big.Int).Sqrt(n)
	}

	// Initial approximation
	x := new(big.Int).Rsh(n, 1) // x = n/2
	if x.Sign() == 0 {
		x.SetInt64(1)
	}

	// Temporary variables
	t := new(big.Int)

	// Newton's method: x = (x + n/x) / 2
	for i := 0; i < 100; i++ {
		// t = n/x
		t.Div(n, x)

		// t = x + n/x
		t.Add(t, x)

		// t = (x + n/x) / 2
		t.Rsh(t, 1)

		// Check for convergence
		if t.Cmp(x) == 0 {
			break
		}

		x, t = t, x
	}

	return x
}

// pow10 returns 10^n as a big.Int
func pow10(n int64) *big.Int {
	result := big.NewInt(1)
	if n <= 0 {
		return result
	}

	ten := big.NewInt(10)
	return result.Exp(ten, big.NewInt(n), nil)
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
