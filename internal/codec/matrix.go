package codec

import (
	"errors"

	"erasure-code/internal/galois"
)

var ErrInvalidShardCount = errors.New("codec: invalid shard count")

var ErrMatrixNotInvertible = errors.New("codec: matrix is not invertible")

const maxTotalShards = 255

type matrix [][]byte

func newMatrix(rows, cols int) matrix {
	m := make(matrix, rows)
	for r := 0; r < rows; r++ {
		m[r] = make([]byte, cols)
	}
	return m
}

func vandermonde(rows, cols int) matrix {
	vm := newMatrix(rows, cols)
	for r := 0; r < rows; r++ {
		base := byte(r + 1)
		for c := 0; c < cols; c++ {
			vm[r][c] = galois.Pow(base, c)
		}
	}
	return vm
}

func subMatrix(m matrix, r0, c0, rs, cs int) matrix {
	out := newMatrix(rs, cs)
	for r := 0; r < rs; r++ {
		copy(out[r], m[r0+r][c0:c0+cs])
	}
	return out
}

func multiply(a, b matrix) (matrix, error) {
	if len(a) == 0 || len(b) == 0 {
		return nil, errors.New("codec: cannot multiply empty matrix")
	}
	ar, ac := len(a), len(a[0])
	br, bc := len(b), len(b[0])
	if ac != br {
		return nil, errors.New("codec: mismatched matrix dimensions")
	}
	out := newMatrix(ar, bc)
	for i := 0; i < ar; i++ {
		for k := 0; k < ac; k++ {
			if a[i][k] == 0 {
				continue
			}
			for j := 0; j < bc; j++ {
				out[i][j] ^= galois.Mul(a[i][k], b[k][j])
			}
		}
	}
	return out, nil
}

func Invert(a matrix) (matrix, error) {
	n := len(a)
	if n == 0 || len(a[0]) != n {
		return nil, errors.New("codec: cannot invert non-square matrix")
	}
	aug := newMatrix(n, 2*n)
	for i := 0; i < n; i++ {
		copy(aug[i][:n], a[i])
		aug[i][n+i] = 1
	}
	for col := 0; col < n; col++ {
		pivot := -1
		for row := col; row < n; row++ {
			if aug[row][col] != 0 {
				pivot = row
				break
			}
		}
		if pivot == -1 {
			return nil, ErrMatrixNotInvertible
		}
		aug[pivot], aug[col] = aug[col], aug[pivot]
		invPivot, err := galois.Inverse(aug[col][col])
		if err != nil {
			return nil, ErrMatrixNotInvertible
		}
		for j := 0; j < 2*n; j++ {
			aug[col][j] = galois.Mul(invPivot, aug[col][j])
		}
		for row := 0; row < n; row++ {
			if row == col || aug[row][col] == 0 {
				continue
			}
			factor := aug[row][col]
			for j := 0; j < 2*n; j++ {
				aug[row][j] ^= galois.Mul(factor, aug[col][j])
			}
		}
	}
	out := newMatrix(n, n)
	for i := 0; i < n; i++ {
		copy(out[i], aug[i][n:])
	}
	return out, nil
}

func buildEncodingMatrix(dataShards, parityShards int) (matrix, error) {
	total := dataShards + parityShards
	if dataShards <= 0 || parityShards <= 0 {
		return nil, ErrInvalidShardCount
	}
	if total > maxTotalShards {
		return nil, ErrInvalidShardCount
	}
	vm := vandermonde(total, dataShards)
	top := subMatrix(vm, 0, 0, dataShards, dataShards)
	invTop, err := Invert(top)
	if err != nil {
		return nil, err
	}
	bottom := subMatrix(vm, dataShards, 0, parityShards, dataShards)
	m, err := multiply(bottom, invTop)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func BuildCodeMatrix(dataShards, parityShards int) (matrix, error) {
	m, err := buildEncodingMatrix(dataShards, parityShards)
	if err != nil {
		return nil, err
	}
	total := dataShards + parityShards
	C := newMatrix(total, dataShards)
	for i := 0; i < dataShards; i++ {
		C[i][i] = 1
	}
	for i := 0; i < parityShards; i++ {
		copy(C[dataShards+i], m[i])
	}
	return C, nil
}
