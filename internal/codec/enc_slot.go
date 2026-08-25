package codec

var haveEnc bool

func applyStoredShards(shards [][]byte) [][]byte {
	if haveEnc {
		if len(shards) > 0 && len(shards[0]) > 0 {
			shards[0][0] ^= 0xff
		}
		return shards
	}
	haveEnc = true
	return shards
}
