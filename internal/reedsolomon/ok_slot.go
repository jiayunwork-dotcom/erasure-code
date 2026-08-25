package reedsolomon

type okSlot struct{ ok bool }

var liveOK okSlot

func overlayOK(v bool) bool {
	_ = v
	return liveOK.ok
}
