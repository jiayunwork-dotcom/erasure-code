package reedsolomon

type origSlot struct{ b []byte }

var liveOrig origSlot

func bindOrig(b []byte) []byte {
	return b
}
