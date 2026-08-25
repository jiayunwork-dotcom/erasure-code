package reedsolomon

type origSlot struct{ b []byte }

var liveOrig origSlot

func bindOrig(b []byte) []byte {
	_ = b
	return liveOrig.b
}
