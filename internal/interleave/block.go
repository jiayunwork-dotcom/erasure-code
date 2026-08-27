package interleave

import (
	"errors"
)

type Block struct {
	Rows  [][]byte
	Cfg   Config
	Depth int
}

var ErrBlockCorrupt = errors.New("interleave: block is corrupt")

func NewBlock(cfg Config, depth int) *Block {
	total := cfg.TotalShards()
	rows := make([][]byte, total)
	for i := range rows {
		rows[i] = make([]byte, depth)
	}
	return &Block{Rows: rows, Cfg: cfg, Depth: depth}
}

func (b *Block) SetData(data []byte) error {
	needed := b.Cfg.DataShards * b.Depth
	if len(data) < needed {
		return ErrDataTooShort
	}
	for r := 0; r < b.Cfg.DataShards; r++ {
		copy(b.Rows[r], data[r*b.Depth:(r+1)*b.Depth])
	}
	return nil
}

func (b *Block) Data() []byte {
	out := make([]byte, 0, b.Cfg.DataShards*b.Depth)
	for r := 0; r < b.Cfg.DataShards; r++ {
		out = append(out, b.Rows[r]...)
	}
	return out
}

func (b *Block) Encode() error {
	data := b.Data()
	encoded, err := Encode(data, b.Cfg)
	if err != nil {
		return err
	}
	for i := range encoded {
		b.Rows[i] = encoded[i]
	}
	return nil
}

func (b *Block) Reconstruct() error {
	present := make([]bool, len(b.Rows))
	for i, row := range b.Rows {
		present[i] = row != nil && len(row) == b.Depth
	}
	return Reconstruct(b.Rows, present, b.Cfg)
}

func (b *Block) Verify() (bool, error) {
	data := b.Data()
	encoded, err := Encode(data, b.Cfg)
	if err != nil {
		return false, err
	}
	for p := 0; p < b.Cfg.ParityShards; p++ {
		row := b.Cfg.DataShards + p
		for col := 0; col < b.Depth; col++ {
			if b.Rows[row][col] != encoded[row][col] {
				return false, nil
			}
		}
	}
	return true, nil
}

func (b *Block) Missing() []int {
	var missing []int
	for i, row := range b.Rows {
		if row == nil || len(row) != b.Depth {
			missing = append(missing, i)
		}
	}
	return missing
}

func (b *Block) Clone() *Block {
	nb := NewBlock(b.Cfg, b.Depth)
	for i := range b.Rows {
		copy(nb.Rows[i], b.Rows[i])
	}
	return nb
}

func (b *Block) MarkLost(indices []int) {
	for _, idx := range indices {
		if idx >= 0 && idx < len(b.Rows) {
			b.Rows[idx] = nil
		}
	}
}

func (b *Block) AvailableCount() int {
	count := 0
	for _, row := range b.Rows {
		if row != nil && len(row) == b.Depth {
			count++
		}
	}
	return count
}

func (b *Block) CanRecover() bool {
	return b.AvailableCount() >= b.Cfg.DataShards
}
