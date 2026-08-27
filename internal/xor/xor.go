package xor

import (
	"encoding/binary"
	"errors"
	"unsafe"
)

var ErrLengthMismatch = errors.New("xor: slice length mismatch")

const wordSize = int(unsafe.Sizeof(uintptr(0)))

func Bytes(dst, src []byte) error {
	if len(dst) != len(src) {
		return ErrLengthMismatch
	}
	n := len(dst)
	if n == 0 {
		return nil
	}
	i := 0
	for ; i+8 <= n; i += 8 {
		d := binary.LittleEndian.Uint64(dst[i:])
		s := binary.LittleEndian.Uint64(src[i:])
		binary.LittleEndian.PutUint64(dst[i:], d^s)
	}
	for ; i < n; i++ {
		dst[i] ^= src[i]
	}
	return nil
}

func BytesTo(dst, a, b []byte) error {
	if len(a) != len(b) || len(dst) != len(a) {
		return ErrLengthMismatch
	}
	n := len(dst)
	i := 0
	for ; i+8 <= n; i += 8 {
		va := binary.LittleEndian.Uint64(a[i:])
		vb := binary.LittleEndian.Uint64(b[i:])
		binary.LittleEndian.PutUint64(dst[i:], va^vb)
	}
	for ; i < n; i++ {
		dst[i] = a[i] ^ b[i]
	}
	return nil
}

func Multi(dst []byte, srcs ...[]byte) error {
	for i := range dst {
		dst[i] = 0
	}
	for _, src := range srcs {
		if len(src) != len(dst) {
			return ErrLengthMismatch
		}
		if err := Bytes(dst, src); err != nil {
			return err
		}
	}
	return nil
}

func Equal(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var acc byte
	for i := range a {
		acc |= a[i] ^ b[i]
	}
	return acc == 0
}

func HammingDistance(a, b []byte) (int, error) {
	if len(a) != len(b) {
		return 0, ErrLengthMismatch
	}
	count := 0
	for i := range a {
		v := a[i] ^ b[i]
		for v != 0 {
			v &= v - 1
			count++
		}
	}
	return count, nil
}

func Fold(slices [][]byte) ([]byte, error) {
	if len(slices) == 0 {
		return nil, nil
	}
	n := len(slices[0])
	result := make([]byte, n)
	copy(result, slices[0])
	for i := 1; i < len(slices); i++ {
		if len(slices[i]) != n {
			return nil, ErrLengthMismatch
		}
		if err := Bytes(result, slices[i]); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func Parity(rows [][]byte) ([]byte, error) {
	return Fold(rows)
}

func RepairXOR(parity []byte, remaining [][]byte) ([]byte, error) {
	if len(parity) == 0 {
		return nil, errors.New("xor: empty parity")
	}
	n := len(parity)
	result := make([]byte, n)
	copy(result, parity)
	for _, r := range remaining {
		if len(r) != n {
			return nil, ErrLengthMismatch
		}
		if err := Bytes(result, r); err != nil {
			return nil, err
		}
	}
	return result, nil
}
