package shamir

// Some comments in this file were written by @autrilla
// The code was written by HashiCorp as part of Vault.

// This implementation of Shamir's Secret Sharing matches the definition
// of the scheme. Other tools used, such as GF(2^8) arithmetic, Lagrange
// interpolation and Horner's method also match their definitions and should
// therefore be correct.
// More information about Shamir's Secret Sharing and Lagrange interpolation
// can be found in README.md

import (
	"crypto/rand"
	"crypto/subtle"
	"fmt"
	mathrand "math/rand"
	"time"
)

// Tables taken from http://www.samiam.org/galois.html
// They use 0xe5 (229) as the generator

var (
	// logTable provides the log(X)/log(g) at each index X
	logTable = [256]uint8{
		0x00, 0xff, 0xc8, 0x08, 0x91, 0x10, 0xd0, 0x36,
		0x5a, 0x3e, 0xd8, 0x43, 0x99, 0x77, 0xfe, 0x18,
		0x23, 0x20, 0x07, 0x70, 0xa1, 0x6c, 0x0c, 0x7f,
		0x62, 0x8b, 0x40, 0x46, 0xc7, 0x4b, 0xe0, 0x0e,
		0xeb, 0x16, 0xe8, 0xad, 0xcf, 0xcd, 0x39, 0x53,
		0x6a, 0x27, 0x35, 0x93, 0xd4, 0x4e, 0x48, 0xc3,
		0x2b, 0x79, 0x54, 0x28, 0x09, 0x78, 0x0f, 0x21,
		0x90, 0x87, 0x14, 0x2a, 0xa9, 0x9c, 0xd6, 0x74,
		0xb4, 0x7c, 0xde, 0xed, 0xb1, 0x86, 0x76, 0xa4,
		0x98, 0xe2, 0x96, 0x8f, 0x02, 0x32, 0x1c, 0xc1,
		0x33, 0xee, 0xef, 0x81, 0xfd, 0x30, 0x5c, 0x13,
		0x9d, 0x29, 0x17, 0xc4, 0x11, 0x44, 0x8c, 0x80,
		0xf3, 0x73, 0x42, 0x1e, 0x1d, 0xb5, 0xf0, 0x12,
		0xd1, 0x5b, 0x41, 0xa2, 0xd7, 0x2c, 0xe9, 0xd5,
		0x59, 0xcb, 0x50, 0xa8, 0xdc, 0xfc, 0xf2, 0x56,
		0x72, 0xa6, 0x65, 0x2f, 0x9f, 0x9b, 0x3d, 0xba,
		0x7d, 0xc2, 0x45, 0x82, 0xa7, 0x57, 0xb6, 0xa3,
		0x7a, 0x75, 0x4f, 0xae, 0x3f, 0x37, 0x6d, 0x47,
		0x61, 0xbe, 0xab, 0xd3, 0x5f, 0xb0, 0x58, 0xaf,
		0xca, 0x5e, 0xfa, 0x85, 0xe4, 0x4d, 0x8a, 0x05,
		0xfb, 0x60, 0xb7, 0x7b, 0xb8, 0x26, 0x4a, 0x67,
		0xc6, 0x1a, 0xf8, 0x69, 0x25, 0xb3, 0xdb, 0xbd,
		0x66, 0xdd, 0xf1, 0xd2, 0xdf, 0x03, 0x8d, 0x34,
		0xd9, 0x92, 0x0d, 0x63, 0x55, 0xaa, 0x49, 0xec,
		0xbc, 0x95, 0x3c, 0x84, 0x0b, 0xf5, 0xe6, 0xe7,
		0xe5, 0xac, 0x7e, 0x6e, 0xb9, 0xf9, 0xda, 0x8e,
		0x9a, 0xc9, 0x24, 0xe1, 0x0a, 0x15, 0x6b, 0x3a,
		0xa0, 0x51, 0xf4, 0xea, 0xb2, 0x97, 0x9e, 0x5d,
		0x22, 0x88, 0x94, 0xce, 0x19, 0x01, 0x71, 0x4c,
		0xa5, 0xe3, 0xc5, 0x31, 0xbb, 0xcc, 0x1f, 0x2d,
		0x3b, 0x52, 0x6f, 0xf6, 0x2e, 0x89, 0xf7, 0xc0,
		0x68, 0x1b, 0x64, 0x04, 0x06, 0xbf, 0x83, 0x38}

	// expTable provides the anti-log or exponentiation value
	// for the equivalent index
	expTable = [256]uint8{
		0x01, 0xe5, 0x4c, 0xb5, 0xfb, 0x9f, 0xfc, 0x12,
		0x03, 0x34, 0xd4, 0xc4, 0x16, 0xba, 0x1f, 0x36,
		0x05, 0x5c, 0x67, 0x57, 0x3a, 0xd5, 0x21, 0x5a,
		0x0f, 0xe4, 0xa9, 0xf9, 0x4e, 0x64, 0x63, 0xee,
		0x11, 0x37, 0xe0, 0x10, 0xd2, 0xac, 0xa5, 0x29,
		0x33, 0x59, 0x3b, 0x30, 0x6d, 0xef, 0xf4, 0x7b,
		0x55, 0xeb, 0x4d, 0x50, 0xb7, 0x2a, 0x07, 0x8d,
		0xff, 0x26, 0xd7, 0xf0, 0xc2, 0x7e, 0x09, 0x8c,
		0x1a, 0x6a, 0x62, 0x0b, 0x5d, 0x82, 0x1b, 0x8f,
		0x2e, 0xbe, 0xa6, 0x1d, 0xe7, 0x9d, 0x2d, 0x8a,
		0x72, 0xd9, 0xf1, 0x27, 0x32, 0xbc, 0x77, 0x85,
		0x96, 0x70, 0x08, 0x69, 0x56, 0xdf, 0x99, 0x94,
		0xa1, 0x90, 0x18, 0xbb, 0xfa, 0x7a, 0xb0, 0xa7,
		0xf8, 0xab, 0x28, 0xd6, 0x15, 0x8e, 0xcb, 0xf2,
		0x13, 0xe6, 0x78, 0x61, 0x3f, 0x89, 0x46, 0x0d,
		0x35, 0x31, 0x88, 0xa3, 0x41, 0x80, 0xca, 0x17,
		0x5f, 0x53, 0x83, 0xfe, 0xc3, 0x9b, 0x45, 0x39,
		0xe1, 0xf5, 0x9e, 0x19, 0x5e, 0xb6, 0xcf, 0x4b,
		0x38, 0x04, 0xb9, 0x2b, 0xe2, 0xc1, 0x4a, 0xdd,
		0x48, 0x0c, 0xd0, 0x7d, 0x3d, 0x58, 0xde, 0x7c,
		0xd8, 0x14, 0x6b, 0x87, 0x47, 0xe8, 0x79, 0x84,
		0x73, 0x3c, 0xbd, 0x92, 0xc9, 0x23, 0x8b, 0x97,
		0x95, 0x44, 0xdc, 0xad, 0x40, 0x65, 0x86, 0xa2,
		0xa4, 0xcc, 0x7f, 0xec, 0xc0, 0xaf, 0x91, 0xfd,
		0xf7, 0x4f, 0x81, 0x2f, 0x5b, 0xea, 0xa8, 0x1c,
		0x02, 0xd1, 0x98, 0x71, 0xed, 0x25, 0xe3, 0x24,
		0x06, 0x68, 0xb3, 0x93, 0x2c, 0x6f, 0x3e, 0x6c,
		0x0a, 0xb8, 0xce, 0xae, 0x74, 0xb1, 0x42, 0xb4,
		0x1e, 0xd3, 0x49, 0xe9, 0x9c, 0xc8, 0xc6, 0xc7,
		0x22, 0x6e, 0xdb, 0x20, 0xbf, 0x43, 0x51, 0x52,
		0x66, 0xb2, 0x76, 0x60, 0xda, 0xc5, 0xf3, 0xf6,
		0xaa, 0xcd, 0x9a, 0xa0, 0x75, 0x54, 0x0e, 0x01}
)

const (
	// ShareOverhead is the byte size overhead of each share
	// when using Split on a secret. This is caused by appending
	// a one byte tag to the share.
	ShareOverhead = 1
)

// polynomial represents a polynomial of arbitrary degree
type polynomial struct {
	coefficients []uint8
}

// makePolynomial constructs a random polynomial of the given
// degree but with the provided intercept value.
func makePolynomial(intercept, degree uint8) (polynomial, error) {
	// Create a wrapper
	p := polynomial{
		coefficients: make([]byte, degree+1),
	}

	// Ensure the intercept is set
	p.coefficients[0] = intercept

	// Assign random co-efficients to the polynomial
	if _, err := rand.Read(p.coefficients[1:]); err != nil {
		return p, err
	}

	return p, nil
}

// evaluate returns the value of the polynomial for the given x
// Uses Horner's method <https://en.wikipedia.org/wiki/Horner%27s_method> to
// evaluate the polynomial at point x
func (p *polynomial) evaluate(x uint8) uint8 {
	// Special case the origin
	if x == 0 {
		return p.coefficients[0]
	}

	// Compute the polynomial value using Horner's method.
	degree := len(p.coefficients) - 1
	out := p.coefficients[degree]
	for i := degree - 1; i >= 0; i-- {
		coeff := p.coefficients[i]
		out = add(mult(out, x), coeff)
	}
	return out
}

// interpolatePolynomial takes N sample points and returns
// the value at a given x using a lagrange interpolation.
// An implementation of Lagrange interpolation
// <https://en.wikipedia.org/wiki/Lagrange_polynomial>
// For this particular implementation, x is always 0
func interpolatePolynomial(xSamples, ySamples []uint8, x uint8) uint8 {
	limit := len(xSamples)
	var result, basis uint8
	for i := 0; i < limit; i++ {
		basis = 1
		for j := 0; j < limit; j++ {
			if i == j {
				continue
			}
			num := add(x, xSamples[j])
			denom := add(xSamples[i], xSamples[j])
			term := div(num, denom)
			basis = mult(basis, term)
		}
		group := mult(ySamples[i], basis)
		result = add(result, group)
	}
	return result
}

// div divides two numbers in GF(2^8)
// GF(2^8) division using log/exp tables
func div(a, b uint8) uint8 {
	if b == 0 {
		// leaks some timing information but we don't care anyways as this
		// should never happen, hence the panic
		panic("divide by zero")
	}

	var goodVal, zero uint8
	logA := logTable[a]
	logB := logTable[b]
	diff := (int(logA) - int(logB)) % 255
	if diff < 0 {
		diff += 255
	}

	ret := expTable[diff]

	// Ensure we return zero if a is zero but aren't subject to timing attacks
	goodVal = ret

	if subtle.ConstantTimeByteEq(a, 0) == 1 {
		ret = zero
	} else {
		ret = goodVal
	}

	return ret
}

// mult multiplies two numbers in GF(2^8)
// GF(2^8) multiplication using log/exp tables
func mult(a, b uint8) (out uint8) {
	var goodVal, zero uint8
	log_a := logTable[a]
	log_b := logTable[b]
	sum := (int(log_a) + int(log_b)) % 255

	ret := expTable[sum]

	// Ensure we return zero if either a or b are zero but aren't subject to
	// timing attacks
	goodVal = ret

	if subtle.ConstantTimeByteEq(a, 0) == 1 {
		ret = zero
	} else {
		ret = goodVal
	}

	if subtle.ConstantTimeByteEq(b, 0) == 1 {
		ret = zero
	} else {
		// This operation does not do anything logically useful. It
		// only ensures a constant number of assignments to thwart
		// timing attacks.
		goodVal = zero
	}

	return ret
}

// add combines two numbers in GF(2^8)
// This can also be used for subtraction since it is symmetric.
func add(a, b uint8) uint8 {
	return a ^ b
}

// Split takes an arbitrarily long secret and generates a `parts`
// number of shares, `threshold` of which are required to reconstruct
// the secret. The parts and threshold must be at least 2, and less
// than 256. The returned shares are each one byte longer than the secret
// as they attach a tag used to reconstruct the secret.
func Split(secret []byte, parts, threshold int) ([][]byte, error) {
	// Sanity check the input
	if parts < threshold {
		return nil, fmt.Errorf("parts cannot be less than threshold")
	}
	if parts > 255 {
		return nil, fmt.Errorf("parts cannot exceed 255")
	}
	if threshold < 2 {
		return nil, fmt.Errorf("threshold must be at least 2")
	}
	if threshold > 255 {
		return nil, fmt.Errorf("threshold cannot exceed 255")
	}
	if len(secret) == 0 {
		return nil, fmt.Errorf("cannot split an empty secret")
	}

	// Generate random x coordinates for computing points. I don't know
	// why random x coordinates are used, and I also don't know why
	// a non-cryptographically secure source of randomness is used.
	// As far as I know the x coordinates do not need to be random.

	mathrand.Seed(time.Now().UnixNano())
	xCoordinates := mathrand.Perm(255)

	// Allocate the output array, initialize the final byte
	// of the output with the offset. The representation of each
	// output is {y1, y2, .., yN, x}.
	out := make([][]byte, parts)
	for idx := range out {
		// Store the x coordinate for each part as its last byte
		// Add 1 to the xCoordinate because if the x coordinate is 0,
		// then the result of evaluating the polynomial at that point
		// will be our secret
		out[idx] = make([]byte, len(secret)+3)
		out[idx][len(secret)] = uint8(xCoordinates[idx]) + 1
	}

	// Construct a random polynomial for each byte of the secret.
	// Because we are using a field of size 256, we can only represent
	// a single byte as the intercept of the polynomial, so we must
	// use a new polynomial for each byte.
	for idx, val := range secret {
		// Create a random polynomial for each point.
		// This polynomial crosses the y axis at `val`.
		p, err := makePolynomial(val, uint8(threshold-1))
		fmt.Printf("Coef: %d\n", p.coefficients)

		if err != nil {
			return nil, fmt.Errorf("failed to generate polynomial: %v", err)
		}

		// Generate a `parts` number of (x,y) pairs
		// We cheat by encoding the x value once as the final index,
		// so that it only needs to be stored once.
		for i := 0; i < parts; i++ {
			// Add 1 to the xCoordinate because if it's 0,
			// then the result of p.evaluate(x) will be our secret
			x := uint8(xCoordinates[i]) + 1
			// Evaluate the polynomial at x
			y := p.evaluate(x)
			out[i][idx] = y

			out[i][len(secret)+1] = p.coefficients[0]
			out[i][len(secret)+2] = p.coefficients[1]
		}
	}

	// Return the encoded secrets
	return out, nil
}

// Combine is used to reverse a Split and reconstruct a secret
// once a `threshold` number of parts are available.
func Combine(parts [][]byte) ([]byte, error) {
	// Verify enough parts provided
	if len(parts) < 2 {
		return nil, fmt.Errorf("less than two parts cannot be used to reconstruct the secret")
	}

	// Verify the parts are all the same length
	firstPartLen := len(parts[0])
	if firstPartLen < 2 {
		return nil, fmt.Errorf("parts must be at least two bytes")
	}
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) != firstPartLen {
			return nil, fmt.Errorf("all parts must be the same length")
		}
	}

	// Create a buffer to store the reconstructed secret
	secret := make([]byte, firstPartLen-1)

	// Buffer to store the samples
	xSamples := make([]uint8, len(parts))
	ySamples := make([]uint8, len(parts))

	// Set the x value for each sample and ensure no x_sample values are the same,
	// otherwise div() can be unhappy
	// Check that we don't have any duplicate parts, that is, two or
	// more parts with the same x coordinate.
	checkMap := map[byte]bool{}
	for i, part := range parts {
		samp := part[firstPartLen-3]
		if exists := checkMap[samp]; exists {
			return nil, fmt.Errorf("duplicate part detected")
		}
		checkMap[samp] = true
		xSamples[i] = samp
	}

	// Reconstruct each byte
	for idx := range secret {
		// Set the y value for each sample
		for i, part := range parts {
			ySamples[i] = part[idx]
		}

		// Use Lagrange interpolation to retrieve the free term
		// of the original polynomial
		val := interpolatePolynomial(xSamples, ySamples, 0)

		// Evaluate the 0th value to get the intercept
		secret[idx] = val
	}
	return secret, nil
}