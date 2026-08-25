package stream

import (
	"io"
)

type ShardWriter struct {
	w         io.Writer
	shardSize int
	written   int
}

func NewShardWriter(w io.Writer, h Header) (*ShardWriter, error) {
	hdr := MarshalHeader(h)
	if _, err := w.Write(hdr); err != nil {
		return nil, err
	}
	stripeSize := int(h.StripeSize)
	shardSize := stripeSize / int(h.DataShards)
	return &ShardWriter{
		w:         w,
		shardSize: shardSize,
	}, nil
}

func (sw *ShardWriter) WriteShard(shard []byte) error {
	if len(shard) != sw.shardSize {
		return ErrShortWrite
	}
	n, err := sw.w.Write(shard)
	if err != nil {
		return err
	}
	if n != sw.shardSize {
		return ErrShortWrite
	}
	sw.written++
	return nil
}

func (sw *ShardWriter) Written() int { return sw.written }

func (sw *ShardWriter) ShardSize() int { return sw.shardSize }

type MultiWriter struct {
	writers []*ShardWriter
}

func NewMultiWriter(ws []io.Writer, h Header) (*MultiWriter, error) {
	sws := make([]*ShardWriter, len(ws))
	for i, w := range ws {
		sw, err := NewShardWriter(w, h)
		if err != nil {
			return nil, err
		}
		sws[i] = sw
	}
	return &MultiWriter{writers: sws}, nil
}

func (mw *MultiWriter) WriteStripe(shards [][]byte) error {
	if len(shards) != len(mw.writers) {
		return ErrShortWrite
	}
	for i, sw := range mw.writers {
		if err := sw.WriteShard(shards[i]); err != nil {
			return err
		}
	}
	return nil
}

func (mw *MultiWriter) Flush() error {
	for _, sw := range mw.writers {
		if f, ok := sw.w.(interface{ Flush() error }); ok {
			if err := f.Flush(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (mw *MultiWriter) WrittenPerShard() []int {
	counts := make([]int, len(mw.writers))
	for i, sw := range mw.writers {
		counts[i] = sw.Written()
	}
	return counts
}
