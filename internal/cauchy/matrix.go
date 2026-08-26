package cauchy

import (
	"errors"

	"erasure-code/internal/galois"
)

var ErrInvalidParams = errors.New("cauchy: invalid parameters")

var ErrTooManyShards = errors.New("cauchy: too many shards for GF(2^8)")

var ErrNotEnoughShards = errors.New("cauchy: not enough shards to reconstruct")

var ErrShardSizeMismatch = errors.New("cauchy: shard size mismatch")

const maxHalfField = 127

type Matrix [][]byte

func NewMatrix(rows, cols int) Matrix {
	m := make(Matrix, rows)
	for r := range m {
		m[r] = make([]byte, cols)
	}
	return m
}

func (m Matrix) Rows() int { return len(m) }

func (m Matrix) Cols() int {
	if len(m) == 0 {
		return 0
	}
	return len(m[0])
}

func (m Matrix) Get(r, c int) byte { return m[r][c] }

func (m Matrix) Set(r, c int, v byte) { m[r][c] = v }

func CauchyMatrix(data, parity int) (Matrix, error) {
	if data <= 0 || parity <= 0 {
		return nil, ErrInvalidParams
	}
	total := data + parity
	if total > maxHalfField {
		return nil, ErrTooManyShards
	}
	m := NewMatrix(parity, data)
	for i := 0; i < parity; i++ {
		xi := byte(data + i)
		for j := 0; j < data; j++ {
			yj := byte(j)
			sum := galois.Add(xi, yj)
			inv, _ := galois.Inverse(sum)
			m[i][j] = inv
		}
	}
	return m, nil
}

func FullMatrix(data, parity int) (Matrix, error) {
	cm, err := CauchyMatrix(data, parity)
	if err != nil {
		return nil, err
	}
	total := data + parity
	full := NewMatrix(total, data)
	for i := 0; i < data; i++ {
		full[i][i] = 1
	}
	for i := 0; i < parity; i++ {
		copy(full[data+i], cm[i])
	}
	return full, nil
}

func SubMatrix(m Matrix, rows []int) Matrix {
	cols := m.Cols()
	sub := NewMatrix(len(rows), cols)
	for i, r := range rows {
		copy(sub[i], m[r])
	}
	return sub
}

func Invert(m Matrix) (Matrix, error) {
	n := m.Rows()
	if n == 0 || m.Cols() != n {
		return nil, errors.New("cauchy: cannot invert non-square matrix")
	}
	aug := NewMatrix(n, 2*n)
	for i := 0; i < n; i++ {
		copy(aug[i][:n], m[i])
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
			return nil, errors.New("cauchy: matrix is singular")
		}
		aug[pivot], aug[col] = aug[col], aug[pivot]
		invPivot, _ := galois.Inverse(aug[col][col])
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
	out := NewMatrix(n, n)
	for i := 0; i < n; i++ {
		copy(out[i], aug[i][n:])
	}
	return out, nil
}
