package app

// projectionAspect reproduces the client's legacy quirk (0x408240 /
// 0x543a00): resolution modes 800x600, 1024x768 and 1280x1024 force 1.25;
// other modes use width/height (Phoenix ep5_camera_controls.md).
func projectionAspect(w, h int) float32 {
	switch {
	case w == 800 && h == 600, w == 1024 && h == 768, w == 1280 && h == 1024:
		return 1.25
	}
	return float32(w) / float32(h)
}
