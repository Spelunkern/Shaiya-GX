package platform

import (
	"fmt"
	"github.com/go-gl/glfw/v3.4/glfw"
	"sort"
)

// The zero mode restores the ordinary resizable window. Nonzero modes use
// a monitor-advertised 24/32-bit fullscreen resolution with the same GL context.
// Windowed nonzero modes resize the ordinary window to a fixed client size
// instead of switching the monitor.
type DisplayMode struct {
	Width, Height int
	Windowed      bool `json:",omitempty"`
}

// windowedSizes are the fixed client sizes offered in windowed mode when the
// monitor is large enough to hold them.
var windowedSizes = []DisplayMode{{Width: 1600, Height: 900, Windowed: true}}

func (m DisplayMode) Valid() bool {
	return m == DisplayMode{} || m.Width >= 640 && m.Height >= 480 && m.Width <= 16384 && m.Height <= 16384
}
func (m DisplayMode) Label() string {
	if m == (DisplayMode{}) {
		return "Window"
	}
	if m.Windowed {
		return fmt.Sprintf("%dx%d Win", m.Width, m.Height)
	}
	return fmt.Sprintf("%dx%d", m.Width, m.Height)
}

// Order: the resizable window, fixed windowed sizes, then fullscreen modes.
func displayModeRank(m DisplayMode) int {
	switch {
	case m == DisplayMode{}:
		return 0
	case m.Windowed:
		return 1
	}
	return 2
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
		if ri, rj := displayModeRank(out[i]), displayModeRank(out[j]); ri != rj {
			return ri < rj
		}
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
		maxW, maxH := 0, 0
		for _, v := range m.GetVideoModes() {
			if v.RedBits >= 8 && v.GreenBits >= 8 && v.BlueBits >= 8 {
				out = append(out, DisplayMode{Width: v.Width, Height: v.Height})
				maxW, maxH = max(maxW, v.Width), max(maxH, v.Height)
			}
		}
		for _, size := range windowedSizes {
			if size.Width <= maxW && size.Height <= maxH {
				out = append(out, size)
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
	if mode.Windowed {
		return w.applyWindowedSize(monitor, mode)
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

// applyWindowedSize leaves fullscreen or un-maximizes, then sets the client
// area to the requested size centered in the monitor's work area.
func (w *Window) applyWindowedSize(monitor *glfw.Monitor, mode DisplayMode) error {
	ax, ay, aw, ah := monitor.GetWorkarea()
	if aw <= 0 || ah <= 0 {
		if v := monitor.GetVideoMode(); v != nil {
			ax, ay = monitor.GetPos()
			aw, ah = v.Width, v.Height
		}
	}
	x, y := ax+max(0, (aw-mode.Width)/2), ay+max(0, (ah-mode.Height)/2)
	if w.GetMonitor() != nil {
		w.SetMonitor(nil, x, y, mode.Width, mode.Height, glfw.DontCare)
	} else {
		if w.GetAttrib(glfw.Maximized) != 0 {
			w.Restore()
		}
		w.SetSize(mode.Width, mode.Height)
		if glfw.GetPlatform() != glfw.PlatformWayland {
			// Keep the title bar on screen when the frame is taller than the gap.
			_, top, _, _ := w.GetFrameSize()
			w.SetPos(x, max(ay+top, y))
		}
	}
	w.windowedMaximized = false
	w.normalSize.Width, w.normalSize.Height = mode.Width, mode.Height
	return nil
}
