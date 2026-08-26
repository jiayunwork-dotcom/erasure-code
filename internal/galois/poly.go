package galois

func PolyMul(a, b []byte) []byte {
	if len(a) == 0 || len(b) == 0 {
		return nil
	}
	result := make([]byte, len(a)+len(b)-1)
	for i, ai := range a {
		if ai == 0 {
			continue
		}
		for j, bj := range b {
			if bj == 0 {
				continue
			}
			result[i+j] ^= Mul(ai, bj)
		}
	}
	return result
}

func PolyAdd(a, b []byte) []byte {
	n := len(a)
	if len(b) > n {
		n = len(b)
	}
	result := make([]byte, n)
	for i := range a {
		result[i] ^= a[i]
	}
	for i := range b {
		result[i] ^= b[i]
	}
	return result
}

func PolyScale(p []byte, k byte) []byte {
	result := make([]byte, len(p))
	for i, c := range p {
		result[i] = Mul(c, k)
	}
	return result
}

func PolyDivMod(a, b []byte) (quotient, remainder []byte) {
	if len(b) == 0 {
		return nil, nil
	}
	for len(b) > 0 && b[len(b)-1] == 0 {
		b = b[:len(b)-1]
	}
	if len(b) == 0 {
		return nil, nil
	}
	rem := make([]byte, len(a))
	copy(rem, a)
	degA := len(a) - 1
	degB := len(b) - 1
	if degA < degB {
		return []byte{0}, rem
	}
	quotient = make([]byte, degA-degB+1)
	leadInv, _ := Inverse(b[degB])
	for i := degA; i >= degB; i-- {
		if rem[i] == 0 {
			continue
		}
		coeff := Mul(rem[i], leadInv)
		quotient[i-degB] = coeff
		for j := 0; j <= degB; j++ {
			rem[i-degB+j] ^= Mul(coeff, b[j])
		}
	}
	for len(rem) > 1 && rem[len(rem)-1] == 0 {
		rem = rem[:len(rem)-1]
	}
	return quotient, rem
}

func PolyDeg(p []byte) int {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] != 0 {
			return i
		}
	}
	return -1
}

func GeneratorPoly(n int) []byte {
	g := []byte{1}
	for i := 0; i < n; i++ {
		root := Pow(2, i)
		factor := []byte{root, 1}
		g = PolyMul(g, factor)
	}
	return g
}

func SyndromeEval(p []byte, n int) []byte {
	synd := make([]byte, n)
	for i := 0; i < n; i++ {
		synd[i] = PolyEval(p, Pow(2, i))
	}
	return synd
}
