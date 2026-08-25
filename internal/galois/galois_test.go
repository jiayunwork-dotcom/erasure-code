package galois

import (
	"math/rand"
	"testing"
)

func sampleElements() []byte {
	return []byte{
		0, 1, 2, 3, 4, 5, 7, 8, 15, 16, 31, 32, 64, 127, 128, 170,
		200, 211, 255 - 1, 255,
	}
}

func TestGaloisAdd(t *testing.T) {
	for _, a := range sampleElements() {
		for _, b := range sampleElements() {
			got := Add(a, b)
			if got != (a ^ b) {
				t.Fatalf("Add(%d,%d)=%d, want %d", a, b, got, a^b)
			}
			if Add(a, b) != Add(b, a) {
				t.Fatalf("Add not commutative for %d,%d", a, b)
			}
			if Add(a, a) != 0 {
				t.Fatalf("Add(%d,%d) should be 0", a, a)
			}
		}
	}
}

func TestGaloisMul(t *testing.T) {
	for _, a := range sampleElements() {
		for _, b := range sampleElements() {
			got := Mul(a, b)
			want := MulTable[a][b]
			if got != want {
				t.Fatalf("Mul(%d,%d)=%d, table says %d", a, b, got, want)
			}
			if a == 0 || b == 0 {
				if got != 0 {
					t.Fatalf("Mul(%d,%d) should be 0", a, b)
				}
			}
			if a == 1 && got != b {
				t.Fatalf("Mul(1,%d)=%d, want %d", b, got, b)
			}
			if b == 1 && got != a {
				t.Fatalf("Mul(%d,1)=%d, want %d", a, got, a)
			}
			for _, c := range sampleElements() {
				ab := Mul(a, b)
				left := Mul(ab, c)
				bc := Mul(b, c)
				right := Mul(a, bc)
				if left != right {
					t.Fatalf("associativity failed for %d,%d,%d", a, b, c)
				}
			}
		}
	}
}

func TestGaloisMulFast(t *testing.T) {
	for a := 0; a < Size; a++ {
		for b := 0; b < Size; b++ {
			if MulFast(byte(a), byte(b)) != Mul(byte(a), byte(b)) {
				t.Fatalf("MulFast(%d,%d) != Mul", a, b)
			}
		}
	}
}

func TestGaloisInverse(t *testing.T) {
	if _, err := Inverse(0); err == nil {
		t.Fatal("Inverse(0) should return an error")
	}
	for _, a := range sampleElements() {
		if a == 0 {
			continue
		}
		inv, err := Inverse(a)
		if err != nil {
			t.Fatalf("Inverse(%d) unexpected error: %v", a, err)
		}
		if Mul(a, inv) != 1 {
			t.Fatalf("a*Inverse(a) != 1 for a=%d (got %d)", a, Mul(a, inv))
		}
	}
}

func TestGaloisDiv(t *testing.T) {
	if _, err := Div(5, 0); err == nil {
		t.Fatal("Div(x,0) should error")
	}
	for _, a := range sampleElements() {
		for _, b := range sampleElements() {
			if b == 0 {
				continue
			}
			q, err := Div(a, b)
			if err != nil {
				t.Fatalf("Div(%d,%d) unexpected error: %v", a, b, err)
			}
			if Mul(q, b) != a {
				t.Fatalf("Div/Mul roundtrip failed for %d/%d", a, b)
			}
		}
	}
}

func TestGaloisIdentity(t *testing.T) {
	for a := 1; a < Size; a++ {
		inv, err := Inverse(byte(a))
		if err != nil {
			t.Fatalf("Inverse(%d): %v", a, err)
		}
		if Mul(byte(a), inv) != 1 {
			t.Fatalf("a*Inverse(a) != 1 for a=%d", a)
		}
		if Pow(byte(a), 255) != 1 {
			t.Fatalf("Pow(a,255) != 1 for a=%d", a)
		}
	}
}

func TestGaloisPow(t *testing.T) {
	if Pow(0, 0) != 1 {
		t.Fatal("Pow(0,0) should be 1")
	}
	if Pow(0, 5) != 0 {
		t.Fatal("Pow(0,k>0) should be 0")
	}
	for _, a := range sampleElements() {
		if a == 0 {
			continue
		}
		for e := 1; e < 20; e++ {
			got := Pow(a, e)
			want := byte(1)
			for i := 0; i < e; i++ {
				want = Mul(want, a)
			}
			if got != want {
				t.Fatalf("Pow(%d,%d)=%d, want %d", a, e, got, want)
			}
		}
	}
}

func TestGaloisExpLogConsistency(t *testing.T) {
	for x := 1; x < Size; x++ {
		if Exp[int(Log[byte(x)])] != byte(x) {
			t.Fatalf("Exp(Log(%d)) != %d", x, x)
		}
	}
	for i := 0; i < 255; i++ {
		if Exp[i] != Exp[i+255] {
			t.Fatalf("Exp not periodic at %d", i)
		}
	}
}

func TestGaloisMulSlice(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	n := 64
	src := make([]byte, n)
	dst := make([]byte, n)
	for i := range src {
		src[i] = byte(rng.Intn(256))
		dst[i] = byte(rng.Intn(256))
	}
	k := byte(173)
	want := make([]byte, n)
	copy(want, dst)
	for i := range src {
		want[i] ^= Mul(k, src[i])
	}
	MulSlice(k, src, dst)
	if !Equal(dst, want) {
		t.Fatal("MulSlice produced incorrect accumulation")
	}
	other := make([]byte, n)
	MulSlice(0, src, other)
	for i := range other {
		if other[i] != 0 {
			t.Fatal("MulSlice with k=0 should write zeros")
		}
	}
}

func TestGaloisMulSliceTable(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	n := 50
	src := make([]byte, n)
	for i := range src {
		src[i] = byte(rng.Intn(256))
	}
	for _, k := range []byte{0, 1, 2, 255} {
		a := make([]byte, n)
		b := make([]byte, n)
		MulSlice(k, src, a)
		MulSliceTable(k, src, b)
		if !Equal(a, b) {
			t.Fatalf("MulSlice and MulSliceTable differ for k=%d", k)
		}
	}
}

func TestGaloisSum(t *testing.T) {
	if Sum(nil) != 0 {
		t.Fatal("Sum(nil) should be 0")
	}
	if Sum([]byte{1, 2, 3}) != (1 ^ 2 ^ 3) {
		t.Fatal("Sum did not XOR correctly")
	}
	for _, x := range sampleElements() {
		if Sum([]byte{x, x}) != 0 {
			t.Fatalf("Sum([%d,%d]) should be 0", x, x)
		}
	}
}

func TestGaloisPolyEval(t *testing.T) {
	p := []byte{1, 2, 3}
	for _, x := range sampleElements() {
		want := byte(1)
		want = Add(want, Mul(2, x))
		want = Add(want, Mul(3, Mul(x, x)))
		if got := PolyEval(p, x); got != want {
			t.Fatalf("PolyEval(p,%d)=%d, want %d", x, got, want)
		}
	}
	if got := PolyEval([]byte{42}, 99); got != 42 {
		t.Fatalf("PolyEval const = %d, want 42", got)
	}
}

func TestGaloisSelfCheck(t *testing.T) {
	if err := SelfCheck(); err != nil {
		t.Fatalf("SelfCheck failed: %v", err)
	}
}
