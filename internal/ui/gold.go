package ui

import "strconv"

// GoldText follows 0x4ce790 -> 0x4cb420: decimal digits grouped in threes.
func GoldText(value uint32) string {
	digits := strconv.FormatUint(uint64(value), 10)
	var grouped [13]byte // maximum uint32 is 4,294,967,295
	end := len(grouped)
	for i := len(digits) - 1; i >= 0; i-- {
		end--
		grouped[end] = digits[i]
		if i > 0 && (len(digits)-i)%3 == 0 {
			end--
			grouped[end] = ','
		}
	}
	return string(grouped[end:])
}

// GoldRGB reproduces 0x4cb4a0's ARGB table after native grouping. Its string
// lengths 5/6/7/9/10/11/13 correspond to the following numeric boundaries.
func GoldRGB(value uint32) [3]uint8 {
	switch {
	case value >= 1_000_000_000:
		return [3]uint8{128, 0, 255}
	case value >= 100_000_000:
		return [3]uint8{255, 0, 255}
	case value >= 10_000_000:
		return [3]uint8{255, 128, 0}
	case value >= 1_000_000:
		return [3]uint8{255, 255, 0}
	case value >= 100_000:
		return [3]uint8{0, 255, 64}
	case value >= 10_000:
		return [3]uint8{0, 128, 255}
	case value >= 1_000:
		return [3]uint8{128, 255, 255}
	default:
		return [3]uint8{255, 255, 255}
	}
}

func goldTint(value uint32) [4]float32 {
	c := GoldRGB(value)
	return [4]float32{float32(c[0]) / 255, float32(c[1]) / 255, float32(c[2]) / 255, 1}
}
