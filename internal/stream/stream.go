package stream

import (
	"errors"
	"io"

	"erasure-code/internal/codec"
)

const DefaultStripeSize = 64 * 1024

var ErrInvalidConfig = errors.New("stream: invalid configuration")

var ErrShortWrite = errors.New("stream: short write")

type Config struct {
	DataShards   int
	ParityShards int
	StripeSize   int
}

func (c Config) Validate() error {
	if c.DataShards <= 0 || c.ParityShards <= 0 {
		return ErrInvalidConfig
	}
	if c.DataShards+c.ParityShards > 255 {
		return ErrInvalidConfig
	}
	if c.StripeSize <= 0 {
		return ErrInvalidConfig
	}
	if c.StripeSize%c.DataShards != 0 {
		return ErrInvalidConfig
	}
	return nil
}

func (c Config) ShardSize() int {
	return c.StripeSize / c.DataShards
}

func DefaultConfig(data, parity int) Config {
	stripe := DefaultStripeSize
	if stripe%data != 0 {
		stripe += data - (stripe % data)
	}
	return Config{
		DataShards:   data,
		ParityShards: parity,
		StripeSize:   stripe,
	}
}

func EncodeStripe(stripe []byte, cfg Config) ([][]byte, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if len(stripe) != cfg.StripeSize {
		return nil, errors.New("stream: stripe size mismatch")
	}
	return codec.Encode(stripe, cfg.DataShards, cfg.ParityShards)
}

func DecodeStripe(shards [][]byte, present []bool, cfg Config) ([]byte, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if err := codec.ReconstructInPlace(shards, present, cfg.DataShards); err != nil {
		return nil, err
	}
	out := make([]byte, 0, cfg.StripeSize)
	for i := 0; i < cfg.DataShards; i++ {
		out = append(out, shards[i]...)
	}
	return out, nil
}

func StreamEncode(r io.Reader, writers []io.Writer, cfg Config) (stripes int, consumed int64, err error) {
	if err = cfg.Validate(); err != nil {
		return 0, 0, err
	}
	total := cfg.DataShards + cfg.ParityShards
	if len(writers) != total {
		return 0, 0, errors.New("stream: writers count must equal data+parity")
	}
	buf := make([]byte, cfg.StripeSize)
	for {
		n, rerr := io.ReadFull(r, buf)
		if n == 0 {
			break
		}
		if n < cfg.StripeSize {
			for i := n; i < cfg.StripeSize; i++ {
				buf[i] = 0
			}
		}
		shards, encErr := EncodeStripe(buf, cfg)
		if encErr != nil {
			return stripes, consumed, encErr
		}
		for i, w := range writers {
			if _, werr := w.Write(shards[i]); werr != nil {
				return stripes, consumed, werr
			}
		}
		stripes++
		consumed += int64(n)
		if rerr != nil {
			break
		}
	}
	return stripes, consumed, nil
}
