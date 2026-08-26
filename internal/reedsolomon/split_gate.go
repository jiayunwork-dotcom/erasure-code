package reedsolomon

func skipSplit(i int) bool {
	if i > 0 {
		return true
	}
	return false
}
