package main

import (
	"fmt"
	"io"
	"math/big"
	"sync"
)

var (
	initonce sync.Once
	zero     = big.NewInt(0)
	two      = big.NewInt(2)
	three    = big.NewInt(3)
	nineteen = big.NewInt(19)
)

// SIEC255Params contains the parameters of an elliptic curve and also provides
// a generic, non-constant time implementation of Curve.
type SIEC255Params struct {
	P       *big.Int // the order of the underlying field
	N       *big.Int // the order of the base point
	A       *big.Int // the other constant of the curve equation
	B       *big.Int // the constant of the curve equation
	Gx, Gy  *big.Int // (x,y) of the base point
	BitSize int      // the size of the underlying field
	Name    string   // the canonical name of the curve
}

// Params returns the parameters for the curve.
func (curve *SIEC255Params) Params() *SIEC255Params {
	return curve
}

// IsOnCurve reports whether the given (x,y) lies on the curve.
func (curve *SIEC255Params) IsOnCurve(x, y *big.Int) bool {
	// y² = x³ + 19
	y2 := new(big.Int).Exp(y, two, curve.P)
	x3 := new(big.Int).Exp(x, two, curve.P)
	return y2.Cmp(x3.Add(x3, nineteen)) == 0
}

// Add returns the sum of (x1,y1) and (x2,y2)
func (curve *SIEC255Params) Add(x1, y1, x2, y2 *big.Int) (x, y *big.Int) {
	if x1.BitLen() == 0 && y1.BitLen() == 0 {
		return x2, y2
	}
	if x2.BitLen() == 0 && y2.BitLen() == 0 {
		return x1, y1
	}
	if x1.Cmp(x2) == 0 && y1.Cmp(y2) == 0 {
		return curve.Double(x1, y1)
	}
	// TODO: optimize
	// λ = (y2 - y1)/(x2 - x1)
	lambda := new(big.Int).Sub(y2, y1)
	z := new(big.Int).Sub(x2, x1)
	z.Mod(z, curve.P)
	if z.BitLen() == 0 {
		return z.Set(zero), lambda.Set(zero)
	}
	z.ModInverse(z, curve.P)
	lambda.Mul(lambda, z)
	lambda.Mod(lambda, curve.P)
	// x3 = λ² - x1 - x2
	x3 := new(big.Int).Exp(lambda, two, curve.P)
	x3.Sub(x3, z.Add(x1, x2))
	x3.Mod(x3, curve.P)
	// y3 = λ(x1 - x3) - y1
	y3 := new(big.Int).Mul(lambda, z.Sub(x1, x3))
	y3.Mod(y3, curve.P)
	y3.Sub(y3, y1)
	y3.Mod(y3, curve.P)
	return x3, y3
}

// Double returns 2*(x,y)
func (curve *SIEC255Params) Double(x1, y1 *big.Int) (x, y *big.Int) {
	x = new(big.Int)
	y = new(big.Int)
	// TODO: optimize
	// λ = (3x1^2)/(2y1)
	lambda := new(big.Int).Mul(three, x.Exp(x1, two, curve.P))
	if y1.BitLen() == 0 {
		return x.Set(zero), y.Set(zero)
	}
	x.Mul(two, y1)
	x.ModInverse(x, curve.P)
	lambda.Mul(lambda, x)
	// x3 = λ² - x1 - x2
	x.Exp(lambda, two, curve.P)
	x.Sub(x, y.Add(x1, x1))
	x.Mod(x, curve.P)
	// y = λ(x1 - x) - y1
	y.Mul(lambda, new(big.Int).Sub(x1, x))
	y.Mod(y, curve.P)
	y.Sub(y, y1)
	y.Mod(y, curve.P)
	return
}

// ScalarMult returns k*(Bx,By) where k is a number in big-endian form.
func (curve *SIEC255Params) ScalarMult(x1, y1 *big.Int, k []byte) (x, y *big.Int) {
	x, y = new(big.Int), new(big.Int)
	for _, b := range k {
		for bitNum := 0; bitNum < 8; bitNum++ {
			x, y = curve.Double(x, y)
			if b&0x80 == 0x80 { // if top bit set
				x, y = curve.Add(x1, y1, x, y)
			}
			b <<= 1
		}
	}
	return x, y
}

// ScalarBaseMult returns k*G, where G is the base point of the group
// and k is an integer in big-endian form.
func (curve *SIEC255Params) ScalarBaseMult(k []byte) (x, y *big.Int) {
	return curve.ScalarMult(curve.Gx, curve.Gy, k)
}

// LiftX returns a point on the curve (x,y) with the given x-value.
// If there is more than one, it returns the one whose y-value
// is smaller in the interval [0,p). If no such point exists,
// then this function panics.
func LiftX(X *big.Int) (x, y *big.Int) {
	params := SIEC255().Params()
	// y² = x³ + Ax + B
	x = new(big.Int).Set(X)
	y = new(big.Int)
	y.Exp(x, three, params.P)
	y.Add(y, new(big.Int).Mul(x, params.A))
	y.Mod(y, params.P)
	y.Add(y, params.B)
	y.Mod(y, params.P)
	y = y.ModSqrt(y, params.P)
	if y == nil {
		panic(fmt.Sprintf("%d is not a point on the curve", X))
	}
	if y.Cmp(new(big.Int).Sub(params.P, y)) > 0 {
		y.Sub(params.P, y)
	}
	return x, y
}

var siec255 *SIEC255Params

func initSIEC255() {
	siec255 = &SIEC255Params{Name: "SIEC255"}
	siec255.Gx = big.NewInt(5)
	siec255.Gy = big.NewInt(12)
	siec255.A = big.NewInt(0)
	siec255.B = big.NewInt(19)
	siec255.P, _ = new(big.Int).SetString("28948022309329048855892746252183396360603931420023084536990047309120118726721", 10)
	siec255.N, _ = new(big.Int).SetString("28948022309329048855892746252183396360263649053102146073526672701688283398081", 10)
	siec255.BitSize = 255
}

// SIEC255 returns a Curve which implements SIEC255.
func SIEC255() *SIEC255Params {
	initonce.Do(initSIEC255)
	return siec255
}

var mask = []byte{0xff, 0x1, 0x3, 0x7, 0xf, 0x1f, 0x3f, 0x7f}

// GenerateKey returns a public/private key pair. The private key is
// generated using the given reader, which must return random data.
// This is copied from https://golang.org/src/crypto/elliptic/elliptic.go?s=7368:7453#L266
func GenerateKey(rand io.Reader) (k []byte, x, y *big.Int, err error) {
	curve := SIEC255()
	N := curve.Params().N
	bitSize := N.BitLen()
	byteLen := (bitSize + 7) >> 3
	k = make([]byte, byteLen)
	for x == nil {
		_, err = io.ReadFull(rand, k)
		if err != nil {
			return
		}
		// We have to mask off any excess bits in the case that the size of the
		// underlying field is not a whole number of bytes.
		k[0] &= mask[bitSize%8]
		// This is because, in tests, rand will return all zeros and we don't
		// want to get the point at infinity and loop forever.
		k[1] ^= 0x42
		// If the scalar is out of range, sample another random number.
		if new(big.Int).SetBytes(k).Cmp(N) >= 0 {
			continue
		}
		x, y = curve.ScalarBaseMult(k)
	}
	return
}

func main() {
	fmt.Println("hello")
}
