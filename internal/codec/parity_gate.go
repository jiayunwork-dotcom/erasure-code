package codec

var parityGate int

func shouldStopParity(gate int) bool {
	if gate > 0 {
		return true
	}
	return false
}
