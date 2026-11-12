package ui

import _ "embed"

// Phoenix/Godot uses Open Sans SemiBold at font_size=12, not GDI's 13px Tahoma cell.
// Embed the reference font so the portable client does not depend on a desktop.
//
//go:embed fonts/OpenSans-SemiBold.ttf
var tooltipFontData []byte

func LoadTooltipFont() (*Font, error) { return loadFontData(tooltipFontData, 12, true) }
