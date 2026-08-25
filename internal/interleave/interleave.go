package interleave

import (
	"errors"

	"erasure-code/internal/codec"
)

var ErrInvalidConfig = errors.New("interleave: invalid configuration")

var ErrDataTooShort = errors.New("interleave: data too short")

var ErrInconsistentShards = errors.New("interleave: inconsistent shard dimensions")

type Config struct {
	DataShards   int
	ParityShards int
	Depth        int
}

func (c Config) Validate() error {
	if c.DataShards <= 0 || c.ParityShards <= 0 || c.Depth <= 0 {
		return ErrInvalidConfig
	}
	if c.DataShards+c.ParityShards > 255 {
		return ErrInvalidConfig
	}
	return nil
}

func (c Config) TotalShards() int { return c.DataShards + c.ParityShards }

func (c Config) BlockSize() int { return c.DataShards * c.Depth }

func Encode(data []byte, cfg Config) ([][]byte, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	blockSize := cfg.BlockSize()
	if len(data) < blockSize {
		return nil, ErrDataTooShort
	}
	rows := make([][]byte, cfg.DataShards)
	for r := 0; r < cfg.DataShards; r++ {
		rows[r] = make([]byte, cfg.Depth)
		copy(rows[r], data[r*cfg.Depth:(r+1)*cfg.Depth])
	}
	parityRows := make([][]byte, cfg.ParityShards)
	for p := 0; p < cfg.ParityShards; p++ {
		parityRows[p] = make([]byte, cfg.Depth)
	}
	for col := 0; col < cfg.Depth; col++ {
		colData := make([]byte, cfg.DataShards)
		for r := 0; r < cfg.DataShards; r++ {
			colData[r] = rows[r][col]
		}
		encoded, err := codec.Encode(colData, cfg.DataShards, cfg.ParityShards)
		if err != nil {
			return nil, err
		}
		for p := 0; p < cfg.ParityShards; p++ {
			parityRows[p][col] = encoded[cfg.DataShards+p][0]
		}
	}
	total := cfg.TotalShards()
	out := make([][]byte, total)
	for r := 0; r < cfg.DataShards; r++ {
		out[r] = rows[r]
	}
	for p := 0; p < cfg.ParityShards; p++ {
		out[cfg.DataShards+p] = parityRows[p]
	}
	return out, nil
}

func Reconstruct(shards [][]byte, present []bool, cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	total := cfg.TotalShards()
	if len(shards) != total || len(present) != total {
		return ErrInconsistentShards
	}
	depth := 0
	for i := range shards {
		if present[i] {
			depth = len(shards[i])
			break
		}
	}
	if depth == 0 {
		return ErrInconsistentShards
	}
	for i := range shards {
		if !present[i] {
			shards[i] = make([]byte, depth)
		}
	}
	for col := 0; col < depth; col++ {
		colShards := make([][]byte, total)
		colPresent := make([]bool, total)
		for r := 0; r < total; r++ {
			colShards[r] = []byte{shards[r][col]}
			colPresent[r] = present[r]
		}
		if err := codec.ReconstructInPlace(colShards, colPresent, cfg.DataShards); err != nil {
			return err
		}
		for r := 0; r < total; r++ {
			shards[r][col] = colShards[r][0]
		}
	}
	return nil
}
