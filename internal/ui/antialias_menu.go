package ui

// Antialias levels, offered in the Advanced graphics window.
const (
	AntialiasOff = iota
	AntialiasFXAA
	AntialiasMSAA2
	AntialiasMSAA4
	AntialiasMSAA8
	AntialiasMSAA16
	AntialiasTAA
	antialiasLevels
)

var antialiasLabels = [antialiasLevels]string{"Off", "FXAA", "MSAA 2x", "MSAA 4x", "MSAA 8x", "MSAA 16x", "TAA"}

// AntialiasSamples is the MSAA sample count for a level (0 for Off and FXAA).
func AntialiasSamples(level int) int {
	switch level {
	case AntialiasMSAA2:
		return 2
	case AntialiasMSAA4:
		return 4
	case AntialiasMSAA8:
		return 8
	case AntialiasMSAA16:
		return 16
	}
	return 0
}

// SetAntialiasMaxSamples offers only the MSAA levels the GPU supports.
func (m *GameMenu) SetAntialiasMaxSamples(maxSamples int) {
	var counts []int
	for _, n := range []int{2, 4, 8, 16} {
		if n <= maxSamples {
			counts = append(counts, n)
		}
	}
	m.SetAntialiasSamples(counts)
}
func (m *GameMenu) SetAntialiasSamples(counts []int) {
	m.antialiasChoices = []int{AntialiasOff, AntialiasFXAA, AntialiasTAA}
	for _, level := range []int{AntialiasMSAA2, AntialiasMSAA4, AntialiasMSAA8, AntialiasMSAA16} {
		for _, count := range counts {
			if AntialiasSamples(level) == count {
				m.antialiasChoices = append(m.antialiasChoices, level)
				break
			}
		}
	}
	m.advanced.list = -1
}

func (m *GameMenu) antialiasRows() []int {
	if len(m.antialiasChoices) == 0 {
		return []int{AntialiasOff, AntialiasFXAA, AntialiasTAA, AntialiasMSAA2, AntialiasMSAA4, AntialiasMSAA8}
	}
	return m.antialiasChoices
}

func (m *GameMenu) antialiasSelected() int {
	for i, level := range m.antialiasRows() {
		if level == m.Settings.Antialias {
			return i
		}
	}
	return 0
}

// NormalizeAntialias keeps the label and actual render mode consistent after
// moving saved settings to a GPU with fewer supported sample counts.
func NormalizeAntialias(level int, counts []int) int {
	if level < AntialiasOff || level >= antialiasLevels {
		return AntialiasOff
	}
	requested := AntialiasSamples(level)
	if requested == 0 {
		return level
	}
	best := 0
	for _, n := range counts {
		if n <= requested && n > best {
			best = n
		}
	}
	for _, candidate := range []int{AntialiasMSAA2, AntialiasMSAA4, AntialiasMSAA8, AntialiasMSAA16} {
		if AntialiasSamples(candidate) == best {
			return candidate
		}
	}
	return AntialiasOff
}
