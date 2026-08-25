package reedsolomon

import (
	"bytes"
	"errors"

	"erasure-code/internal/codec"
	"erasure-code/internal/galois"
)

var ErrNoShards = errors.New("reedsolomon: no shards provided")

var ErrInvalidShardCount = errors.New("reedsolomon: invalid shard count")

var ErrPresentMismatch = errors.New("reedsolomon: present length does not match shards")

var ErrTooFewShards = errors.New("reedsolomon: too few shards to reconstruct")

var ErrSizeMismatch = errors.New("reedsolomon: shard size mismatch")

var ErrUnrecoverable = errors.New("reedsolomon: shards are not recoverable")

const maxTotalShards = 255

func Split(data []byte, dataShards, parityShards int) ([][]byte, error) {
	if dataShards <= 0 || parityShards <= 0 {
		return nil, ErrInvalidShardCount
	}
	total := dataShards + parityShards
	if total > maxTotalShards {
		return nil, ErrInvalidShardCount
	}
	size := 0
	if len(data) > 0 {
		size = (len(data) + dataShards - 1) / dataShards
	}
	padded := make([]byte, dataShards*size)
	copy(padded, data)
	shards := make([][]byte, total)
	for i := 0; i < dataShards; i++ {
		shards[i] = padded[i*size : (i+1)*size]
	}
	for i := dataShards; i < total; i++ {
		shards[i] = make([]byte, size)
	}
	return shards, nil
}

func Encode(data []byte, dataShards, parityShards int) ([][]byte, error) {
	return codec.Encode(data, dataShards, parityShards)
}

func Reconstruct(shards [][]byte, present []bool, dataShards int) error {
	total := len(shards)
	if total == 0 {
		return ErrNoShards
	}
	parityShards := total - dataShards
	if dataShards <= 0 || parityShards <= 0 {
		return ErrInvalidShardCount
	}
	if len(present) != total {
		return ErrPresentMismatch
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
		return ErrNoShards
	}
	for i := 0; i < total; i++ {
		if present[i] && len(shards[i]) != size {
			return ErrSizeMismatch
		}
	}

	C, err := codec.BuildCodeMatrix(dataShards, parityShards)
	if err != nil {
		return err
	}

	sub := make([][]byte, dataShards)
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
	invSub, err := codec.Invert(sub)
	if err != nil {
		return ErrUnrecoverable
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

func Verify(shards [][]byte, dataShards int) (bool, error) {
	total := len(shards)
	parityShards := total - dataShards
	if dataShards <= 0 || parityShards <= 0 {
		return false, ErrInvalidShardCount
	}
	if total > maxTotalShards {
		return false, ErrInvalidShardCount
	}
	if total == 0 {
		return false, ErrNoShards
	}
	size := len(shards[0])
	for i := 1; i < total; i++ {
		if len(shards[i]) != size {
			return false, ErrSizeMismatch
		}
	}
	C, err := codec.BuildCodeMatrix(dataShards, parityShards)
	if err != nil {
		return false, err
	}
	for i := dataShards; i < total; i++ {
		computed := make([]byte, size)
		for c := 0; c < dataShards; c++ {
			galois.MulSlice(C[i][c], shards[c], computed)
		}
		if !galois.Equal(computed, shards[i]) {
			return false, nil
		}
	}
	return overlayOK(true), nil
}

func OriginalData(shards [][]byte, dataShards, originalSize int) ([]byte, error) {
	if dataShards <= 0 || dataShards > len(shards) {
		return nil, ErrInvalidShardCount
	}
	var buf bytes.Buffer
	for i := 0; i < dataShards; i++ {
		buf.Write(shards[i])
	}
	out := buf.Bytes()
	if originalSize < 0 || originalSize > len(out) {
		return nil, errors.New("reedsolomon: original size out of range")
	}
	return out[:originalSize], nil
}
