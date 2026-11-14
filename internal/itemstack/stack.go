// Package itemstack shares GX item quantity limits and client revision.
package itemstack

const Max = 255
const ClientVersion uint32 = 0x11122202

// Decode the retired wide-count revision for offline fixtures. New writers
// use the original opcodes and byte quantities.
const WideFlag uint16 = 0x4000

func Wide(op uint16) bool { return op >= 0x4000 && op < 0x7000 }
func Base(op uint16) uint16 {
	if Wide(op) {
		return op &^ WideFlag
	}
	return op
}
func Limit(old uint16) uint16 {
	if old > 1 {
		return Max
	}
	return 1
}
