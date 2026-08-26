package stream

import (
	"encoding/binary"
	"errors"
	"io"
)

type Header struct {
	DataShards   uint8
	ParityShards uint8
	StripeSize   uint32
	OriginalSize uint64
}

const headerSize = 1 + 1 + 4 + 8

var ErrBadHeader = errors.New("stream: malformed header")

func MarshalHeader(h Header) []byte {
	buf := make([]byte, headerSize)
	buf[0] = h.DataShards
	buf[1] = h.ParityShards
	binary.BigEndian.PutUint32(buf[2:6], h.StripeSize)
	binary.BigEndian.PutUint64(buf[6:14], h.OriginalSize)
	return buf
}

func UnmarshalHeader(buf []byte) (Header, error) {
	if len(buf) < headerSize {
		return Header{}, ErrBadHeader
	}
	h := Header{
		DataShards:   buf[0],
		ParityShards: buf[1],
		StripeSize:   binary.BigEndian.Uint32(buf[2:6]),
		OriginalSize: binary.BigEndian.Uint64(buf[6:14]),
	}
	if h.DataShards == 0 || h.ParityShards == 0 || h.StripeSize == 0 {
		return Header{}, ErrBadHeader
	}
	return h, nil
}

type ShardReader struct {
	r         io.Reader
	header    Header
	shardSize int
	stripe    int
	totalStr  int
	done      bool
}

func NewShardReader(r io.Reader) (*ShardReader, error) {
	buf := make([]byte, headerSize)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	h, err := UnmarshalHeader(buf)
	if err != nil {
		return nil, err
	}
	stripeSize := int(h.StripeSize)
	shardSize := stripeSize / int(h.DataShards)
	totalStripes := int((h.OriginalSize + uint64(stripeSize) - 1) / uint64(stripeSize))
	return &ShardReader{
		r:         r,
		header:    h,
		shardSize: shardSize,
		totalStr:  totalStripes,
	}, nil
}

func (sr *ShardReader) Header() Header { return sr.header }

func (sr *ShardReader) ReadShard() ([]byte, error) {
	if sr.done {
		return nil, io.EOF
	}
	buf := make([]byte, sr.shardSize)
	n, err := io.ReadFull(sr.r, buf)
	if n == 0 && err != nil {
		sr.done = true
		return nil, io.EOF
	}
	if err != nil && err != io.ErrUnexpectedEOF {
		return nil, err
	}
	if n < sr.shardSize {
		for i := n; i < sr.shardSize; i++ {
			buf[i] = 0
		}
	}
	return buf, nil
}

func (sr *ShardReader) Remaining() int {
	left := sr.totalStr - sr.stripe
	if left < 0 {
		return 0
	}
	return left
}

func (sr *ShardReader) Advance() {
	sr.stripe++
	if sr.stripe >= sr.totalStr {
		sr.done = true
	}
}
