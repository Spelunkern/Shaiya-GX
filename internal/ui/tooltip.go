package ui

// TooltipOrigin follows Phoenix SkillTooltip.place: left of the pointer,
// right when necessary, with vertical and viewport clamping.
func TooltipOrigin(mouseX, mouseY, width, height, w, h int) (int, int) {
	x := mouseX - width
	if x < 0 {
		x = mouseX + 16
	}
	return max(0, min(x, w-width)), max(0, min(mouseY+10, h-height))
}

// Phoenix ItemTooltip uses an 8px right offset; SkillTooltip uses 16px and
// performs its initial unscaled clamp before fitting very small viewports.
func TooltipOriginFor(mouseX, mouseY, width, height, w, h int, item bool, scale float32) (int, int) {
	x, y := mouseX-width, mouseY+10
	if x < 0 {
		if item {
			x = mouseX + 8
		} else {
			x = mouseX + 16
		}
	}
	if !item {
		x = max(0, min(x, w-width))
		y = max(0, min(y, h-height))
	}
	return max(0, min(x, w-int(float32(width)*scale))), max(0, min(y, h-int(float32(height)*scale)))
}
