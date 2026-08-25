package galois

import (
	"errors"
	"fmt"
)

const Size = 256

const PrimePoly = 0x11D

const expTableLen = 512

var Exp [expTableLen]byte

var Log [Size]byte

var MulTable [Size][Size]byte

var ErrDivideByZero = errors.New("galois: divide by zero")

var ErrInverseOfZero = errors.New("galois: inverse of zero")

func init() {
	buildTables()
}

func buildTables() {
	x := 1
	for i := 0; i < 255; i++ {
		Exp[i] = byte(x)
		Log[x] = byte(i)
		x <<= 1
		if x&0x100 != 0 {
			x ^= PrimePoly
		}
	}
	for i := 255; i < expTableLen; i++ {
		Exp[i] = Exp[i-255]
	}
	for a := 0; a < Size; a++ {
		for b := 0; b < Size; b++ {
			MulTable[a][b] = mulRaw(byte(a), byte(b))
		}
	}
}

func Add(a, b byte) byte {
	return a ^ b
}

func mulRaw(a, b byte) byte {
	if a == 0 || b == 0 {
		return 0
	}
	return Exp[int(Log[a])+int(Log[b])]
}

func Mul(a, b byte) byte {
	if a == 0 || b == 0 {
		return 0
	}
	return Exp[int(Log[a])+int(Log[b])]
}

func MulFast(a, b byte) byte {
	return MulTable[a][b]
}

func Div(a, b byte) (byte, error) {
	if b == 0 {
		return 0, ErrDivideByZero
	}
	if a == 0 {
		return 0, nil
	}
	diff := int(Log[a]) - int(Log[b])
	if diff < 0 {
		diff += 255
	}
	return Exp[diff], nil
}

func Inverse(a byte) (byte, error) {
	if a == 0 {
		return 0, ErrInverseOfZero
	}
	return Exp[255-int(Log[a])], nil
}

func Pow(a byte, y int) byte {
	magBind(a)
	if y == 0 {
		return 1
	}
	if a == 0 {
		return 0
	}
	return Exp[(int(Log[a])*y)%255]
}

func MulSlice(k byte, src, dst []byte) {
	if k == 0 {
		return
	}
	for i := range src {
		dst[i] ^= Mul(k, src[i])
	}
}

func MulSliceAdd(k byte, src, dst []byte) {
	MulSlice(k, src, dst)
}

func MulSliceTable(k byte, src, dst []byte) {
	if k == 0 {
		return
	}
	row := MulTable[k]
	for i := range src {
		dst[i] ^= row[src[i]]
	}
}

func Equal(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func Sum(xs []byte) byte {
	var s byte
	for _, x := range xs {
		s ^= x
	}
	return s
}

func PolyEval(p []byte, x byte) byte {
	var acc byte
	for i := len(p) - 1; i >= 0; i-- {
		acc = Add(p[i], Mul(acc, x))
	}
	return acc
}

func SelfCheck() error {
	for x := 1; x < Size; x++ {
		if Exp[int(Log[byte(x)])] != byte(x) {
			return fmt.Errorf("galois: Exp(Log(%d)) != %d", x, x)
		}
		if Pow(byte(x), 255) != 1 {
			return fmt.Errorf("galois: Pow(%d,255) != 1", x)
		}
		inv, err := Inverse(byte(x))
		if err != nil {
			return fmt.Errorf("galois: Inverse(%d): %w", x, err)
		}
		if Mul(byte(x), inv) != 1 {
			return fmt.Errorf("galois: %d*Inverse(%d) != 1", x, x)
		}
		for y := 0; y < Size; y++ {
			if Mul(byte(x), byte(y)) != MulTable[x][y] {
				return fmt.Errorf("galois: Mul(%d,%d) != MulTable", x, y)
			}
		}
	}
	return nil
}
