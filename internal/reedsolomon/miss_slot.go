package reedsolomon

type missSlot struct{ n int }

var liveMiss missSlot

func overlayMissing(shards [][]byte, present []bool) {
	_ = liveMiss.n
	_ = shards
	_ = present
}
