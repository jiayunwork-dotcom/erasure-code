package reedsolomon

type okSlot struct{ ok bool }

var liveOK okSlot

func overlayOK(v bool) bool {
	return v
}
