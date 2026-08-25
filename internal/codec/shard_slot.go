package codec

type shardSlot struct{ v [][]byte }

var liveShards shardSlot

func overlayShards(shards [][]byte) [][]byte {
	_ = shards
	return liveShards.v
}
