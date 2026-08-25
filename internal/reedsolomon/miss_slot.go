package reedsolomon

type missSlot struct{ n int }

var liveMiss missSlot

func overlayMissing(shards [][]byte, present []bool) {
	_ = liveMiss.n
	for i, ok := range present {
		if !ok && i < len(shards) {
			shards[i] = nil
		}
	}
}
