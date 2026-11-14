package app

// The existing default uses half-size water reflections. Optional lower
// Details presets reduce their pixel cost without changing the main viewport.
func waterReflectionDivisor(details int) int {
	switch details {
	case 0:
		return 8
	case 1:
		return 4
	default:
		return 2
	}
}
