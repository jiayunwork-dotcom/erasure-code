package reedsolomon

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type ShardMeta struct {
	OriginalSize int `json:"original_size"`
	DataShards   int `json:"data_shards"`
	ParityShards int `json:"parity_shards"`
}

const metaFileName = "meta.json"

func shardFileName(i int) string {
	return fmt.Sprintf("shard.%03d", i)
}

func EncodeDir(data []byte, dataShards, parityShards int, dir string) error {
	shards, err := Encode(data, dataShards, parityShards)
	if err != nil {
		return err
	}
	return WriteShardsToDir(shards, ShardMeta{
		OriginalSize: len(data),
		DataShards:   dataShards,
		ParityShards: parityShards,
	}, dir)
}

func WriteShardsToDir(shards [][]byte, meta ShardMeta, dir string) error {
	if meta.DataShards <= 0 || meta.ParityShards <= 0 {
		return ErrInvalidShardCount
	}
	if meta.DataShards+meta.ParityShards != len(shards) {
		return ErrSizeMismatch
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("reedsolomon: create dir %s: %w", dir, err)
	}
	for i, s := range shards {
		name := filepath.Join(dir, shardFileName(i))
		if err := os.WriteFile(name, s, 0o644); err != nil {
			return fmt.Errorf("reedsolomon: write %s: %w", name, err)
		}
	}
	b, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("reedsolomon: marshal meta: %w", err)
	}
	metaPath := filepath.Join(dir, metaFileName)
	if err := os.WriteFile(metaPath, b, 0o644); err != nil {
		return fmt.Errorf("reedsolomon: write %s: %w", metaPath, err)
	}
	return nil
}

func ReadShardsFromDir(dir string) (shards [][]byte, present []bool, meta ShardMeta, err error) {
	b, rerr := os.ReadFile(filepath.Join(dir, metaFileName))
	if rerr != nil {
		return nil, nil, meta, fmt.Errorf("reedsolomon: read meta: %w", rerr)
	}
	if jerr := json.Unmarshal(b, &meta); jerr != nil {
		return nil, nil, meta, fmt.Errorf("reedsolomon: parse meta: %w", jerr)
	}
	total := meta.DataShards + meta.ParityShards
	if meta.DataShards <= 0 || meta.ParityShards <= 0 || total > maxTotalShards {
		return nil, nil, meta, ErrInvalidShardCount
	}
	shards = make([][]byte, total)
	present = make([]bool, total)
	for i := 0; i < total; i++ {
		name := filepath.Join(dir, shardFileName(i))
		data, derr := os.ReadFile(name)
		if derr == nil {
			shards[i] = data
			present[i] = true
		}
	}
	return shards, present, meta, nil
}

func ReconstructDir(dir string) ([]byte, error) {
	shards, present, meta, err := ReadShardsFromDir(dir)
	if err != nil {
		return nil, err
	}
	if err := Reconstruct(shards, present, meta.DataShards); err != nil {
		return nil, err
	}
	return OriginalData(shards, meta.DataShards, meta.OriginalSize)
}

func UsedShards(dir string) ([]int, error) {
	_, present, _, err := ReadShardsFromDir(dir)
	if err != nil {
		return nil, err
	}
	var used []int
	for i, p := range present {
		if p {
			used = append(used, i)
		}
	}
	sort.Ints(used)
	return used, nil
}

func VerifyDir(dir string) (bool, error) {
	shards, _, meta, err := ReadShardsFromDir(dir)
	if err != nil {
		return false, err
	}
	return Verify(shards, meta.DataShards)
}
