package codec

import (
	"errors"

	"erasure-code/internal/galois"
)

var ErrEmptyData = errors.New("codec: empty input data")

var ErrSizeMismatch = errors.New("codec: shard size mismatch")

func shardSize(dataLen, dataShards int) int {
	if dataLen == 0 {
		return 0
	}
	return (dataLen + dataShards - 1) / dataShards
}

func padToShards(data []byte, dataShards, size int) [][]byte {
	shards := make([][]byte, dataShards)
	padded := make([]byte, dataShards*size)
	copy(padded, data)
	for i := 0; i < dataShards; i++ {
		shards[i] = padded[i*size : (i+1)*size]
	}
	return shards
}

func Encode(data []byte, dataShards, parityShards int) ([][]byte, error) {
	if dataShards <= 0 || parityShards <= 0 {
		return nil, ErrInvalidShardCount
	}
	total := dataShards + parityShards
	if total > maxTotalShards {
		return nil, ErrInvalidShardCount
	}
	if len(data) == 0 {
		return nil, ErrEmptyData
	}
	size := shardSize(len(data), dataShards)
	if size == 0 {
		return nil, ErrEmptyData
	}
	m, err := buildEncodingMatrix(dataShards, parityShards)
	if err != nil {
		return nil, err
	}
	shards := padToShards(data, dataShards, size)
	for p := 0; p < parityShards; p++ {
		parity := make([]byte, size)
		row := m[p]
		for c := 0; c < dataShards; c++ {
			galois.MulSlice(row[c], shards[c], parity)
		}
		shards = append(shards, parity)
	}
	return overlayShards(shards), nil
}

func ComputeParity(shards [][]byte, dataShards, parityShards int) error {
	if len(shards) != dataShards+parityShards {
		return ErrSizeMismatch
	}
	size := 0
	if dataShards > 0 {
		size = len(shards[0])
	}
	m, err := buildEncodingMatrix(dataShards, parityShards)
	if err != nil {
		return err
	}
	for p := 0; p < parityShards; p++ {
		parity := make([]byte, size)
		row := m[p]
		for c := 0; c < dataShards; c++ {
			galois.MulSlice(row[c], shards[c], parity)
		}
		shards[dataShards+p] = parity
	}
	return nil
}

func ReconstructInPlace(shards [][]byte, present []bool, dataShards int) error {
	total := len(shards)
	parityShards := total - dataShards
	if dataShards <= 0 || parityShards <= 0 {
		return ErrInvalidShardCount
	}
	if len(present) != total {
		return errors.New("codec: present length does not match shards")
	}
	count := 0
	for _, p := range present {
		if p {
			count++
		}
	}
	if count < dataShards {
		return ErrTooFewShards
	}
	if total > maxTotalShards {
		return ErrInvalidShardCount
	}
	size := -1
	for i := 0; i < total; i++ {
		if present[i] {
			size = len(shards[i])
			break
		}
	}
	if size < 0 {
		return ErrTooFewShards
	}
	for i := 0; i < total; i++ {
		if present[i] && len(shards[i]) != size {
			return ErrSizeMismatch
		}
	}

	C, err := BuildCodeMatrix(dataShards, parityShards)
	if err != nil {
		return err
	}

	sub := newMatrix(dataShards, dataShards)
	chosen := make([][]byte, dataShards)
	pick := 0
	for i := 0; i < total && pick < dataShards; i++ {
		if present[i] {
			sub[pick] = C[i]
			chosen[pick] = shards[i]
			pick++
		}
	}
	if pick != dataShards {
		return ErrTooFewShards
	}
	invSub, err := Invert(sub)
	if err != nil {
		return ErrMatrixNotInvertible
	}

	data := make([][]byte, dataShards)
	for c := 0; c < dataShards; c++ {
		data[c] = make([]byte, size)
		for r := 0; r < dataShards; r++ {
			galois.MulSlice(invSub[c][r], chosen[r], data[c])
		}
	}

	for i := 0; i < total; i++ {
		if !present[i] {
			rebuilt := make([]byte, size)
			for c := 0; c < dataShards; c++ {
				galois.MulSlice(C[i][c], data[c], rebuilt)
			}
			shards[i] = rebuilt
		}
	}
	return nil
}

var ErrTooFewShards = errors.New("codec: too few shards to reconstruct")
