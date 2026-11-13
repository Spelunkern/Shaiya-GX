package platform

import (
	"fmt"
	"github.com/go-gl/glfw/v3.4/glfw"
	"sort"
)

// The zero mode restores the ordinary resizable window. Nonzero modes use
// a monitor-advertised 24/32-bit fullscreen resolution with the same GL context.
type DisplayMode struct{ Width, Height int }

func (m DisplayMode) Valid() bool {
	return m == DisplayMode{} || m.Width >= 640 && m.Height >= 480 && m.Width <= 16384 && m.Height <= 16384
}
func (m DisplayMode) Label() string {
	if m == (DisplayMode{}) {
		return "Window"
	}
	return fmt.Sprintf("%dx%d", m.Width, m.Height)
}
func NormalizeDisplayModes(modes []DisplayMode) []DisplayMode {
	unique := map[DisplayMode]bool{{}: true}
	for _, m := range modes {
		if m.Valid() {
			unique[m] = true
		}
	}
	out := make([]DisplayMode, 0, len(unique))
	for m := range unique {
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Width != out[j].Width {
			return out[i].Width < out[j].Width
		}
		return out[i].Height < out[j].Height
	})
	return out
}
func (w *Window) displayMonitor() *glfw.Monitor {
	if m := w.GetMonitor(); m != nil {
		return m
	}
	// Wayland deliberately does not expose a window's global position.
	if glfw.GetPlatform() == glfw.PlatformWayland {
		return glfw.GetPrimaryMonitor()
	}
	x, y := w.GetPos()
	width, height := w.GetSize()
	best, overlap := glfw.GetPrimaryMonitor(), 0
	for _, m := range glfw.GetMonitors() {
		v := m.GetVideoMode()
		if v == nil {
			continue
		}
		mx, my := m.GetPos()
		area := max(0, min(x+width, mx+v.Width)-max(x, mx)) * max(0, min(y+height, my+v.Height)-max(y, my))
		if area > overlap {
			best, overlap = m, area
		}
	}
	return best
}
func (w *Window) DisplayModes() []DisplayMode {
	var out []DisplayMode
	if m := w.displayMonitor(); m != nil {
		for _, v := range m.GetVideoModes() {
			if v.RedBits >= 8 && v.GreenBits >= 8 && v.BlueBits >= 8 {
				out = append(out, DisplayMode{v.Width, v.Height})
			}
		}
	}
	return NormalizeDisplayModes(out)
}
func (w *Window) ApplyDisplayMode(mode DisplayMode) (err error) {
	defer func() {
		if failure := recover(); failure != nil {
			if e, ok := failure.(*glfw.Error); ok {
				err = fmt.Errorf("display change: %w", e)
			} else {
				panic(failure)
			}
		}
	}()
	if !mode.Valid() {
		return fmt.Errorf("invalid display mode %dx%d", mode.Width, mode.Height)
	}
	if mode == (DisplayMode{}) {
		if w.GetMonitor() == nil {
			return nil
		}
		width, height := w.normalSize.Width, w.normalSize.Height
		if width < 320 || height < 240 {
			width, height = 1024, 768
		}
		w.SetMonitor(nil, w.windowedX, w.windowedY, width, height, glfw.DontCare)
		if w.windowedMaximized {
			w.Maximize()
		}
		return nil
	}
	monitor := w.displayMonitor()
	if monitor == nil {
		return fmt.Errorf("no display monitor")
	}
	refresh := -1
	for _, v := range monitor.GetVideoModes() {
		if v.Width == mode.Width && v.Height == mode.Height && v.RedBits >= 8 && v.GreenBits >= 8 && v.BlueBits >= 8 {
			refresh = max(refresh, v.RefreshRate)
		}
	}
	if refresh < 0 {
		return fmt.Errorf("display mode %s is unavailable", mode.Label())
	}
	if w.GetMonitor() == nil {
		if glfw.GetPlatform() != glfw.PlatformWayland {
			w.windowedX, w.windowedY = w.GetPos()
		}
		w.windowedMaximized = w.GetAttrib(glfw.Maximized) != 0
		if !w.windowedMaximized {
			width, height := w.GetSize()
			w.normalSize.Width, w.normalSize.Height = width, height
		}
	}
	w.SetMonitor(monitor, 0, 0, mode.Width, mode.Height, refresh)
	return nil
}
