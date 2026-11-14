package platform

import (
	"fmt"
	"github.com/go-gl/glfw/v3.4/glfw"
	"sort"
)

// Rendering is always windowed. Kind is retained only to migrate older settings.
type DisplayKind uint8

const (
	DisplayLegacy DisplayKind = iota
	DisplayWindowed
	DisplayFullscreen
)

type DisplayMode struct {
	Width, Height int
	Kind          DisplayKind `json:",omitempty"`
}

func (m DisplayMode) Valid() bool {
	if m == (DisplayMode{}) || m == (DisplayMode{Kind: DisplayFullscreen}) {
		return true
	}
	minimumWidth, minimumHeight := 320, 240
	return m.Kind <= DisplayWindowed && m.Width >= minimumWidth && m.Height >= minimumHeight && m.Width <= 16384 && m.Height <= 16384
}

// WindowMode migrates legacy fullscreen settings to an ordinary window size.
// Zero remains an internal "keep current size" value and is never a menu row.
func WindowMode(m DisplayMode) DisplayMode {
	if m.Width > 0 && m.Height > 0 {
		return DisplayMode{Width: m.Width, Height: m.Height}
	}
	return DisplayMode{}
}

func (m DisplayMode) Label() string {
	if m.Width == 0 || m.Height == 0 {
		return ""
	}
	return fmt.Sprintf("%dx%d", m.Width, m.Height)
}
func NormalizeDisplayModes(modes []DisplayMode) []DisplayMode {
	unique := map[DisplayMode]bool{}
	for _, m := range modes {
		if !m.Valid() {
			continue
		}
		m = WindowMode(m)
		if m.Valid() && m.Width > 0 {
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

// Common client sizes are windowed choices, independent of the monitor's
// exclusive mode list. Fit in logical work-area units including decorations.
var commonWindowSizes = [][2]int{
	{640, 480}, {800, 600}, {1024, 768}, {1152, 864}, {1280, 720}, {1280, 800},
	{1280, 960}, {1280, 1024}, {1360, 768}, {1366, 768}, {1440, 900}, {1600, 900},
	{1600, 1000}, {1600, 1200}, {1680, 1050}, {1920, 1080}, {1920, 1200}, {2048, 1152},
	{2560, 1080}, {2560, 1440}, {2560, 1600}, {3440, 1440}, {3840, 1600}, {3840, 2160},
	{5120, 1440}, {5120, 2160}, {5120, 2880}, {7680, 2160}, {7680, 4320},
}

func addWindowModes(out []DisplayMode, width, height, currentWidth, currentHeight int) []DisplayMode {
	for _, size := range commonWindowSizes {
		if size[0] <= width && size[1] <= height {
			out = append(out, DisplayMode{Width: size[0], Height: size[1], Kind: DisplayWindowed})
		}
	}
	current := DisplayMode{Width: currentWidth, Height: currentHeight, Kind: DisplayWindowed}
	if current.Valid() {
		out = append(out, current)
	}
	return NormalizeDisplayModes(out)
}

func (w *Window) windowWorkarea() (x, y, width, height int) {
	if m := w.displayMonitor(); m != nil {
		x, y, width, height = m.GetWorkarea()
		l, t, r, b := w.GetFrameSize()
		return x + l, y + t, width - l - r, height - t - b
	}
	return
}

func (w *Window) CurrentDisplayMode() DisplayMode {
	width, height := w.GetSize()
	return DisplayMode{Width: width, Height: height}
}

func (w *Window) DisplayModes() []DisplayMode {
	var out []DisplayMode
	if m := w.displayMonitor(); m != nil {
		for _, v := range m.GetVideoModes() {
			if v.RedBits >= 8 && v.GreenBits >= 8 && v.BlueBits >= 8 {
				out = append(out, DisplayMode{Width: v.Width, Height: v.Height})
			}
		}
	}
	_, _, width, height := w.windowWorkarea()
	cw, ch := w.GetSize()
	return addWindowModes(out, width, height, cw, ch)
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
	mode = WindowMode(mode)
	if mode == (DisplayMode{}) {
		return nil
	}
	available := false
	for _, candidate := range w.DisplayModes() {
		available = available || candidate == mode
	}
	if !available {
		return fmt.Errorf("window resolution %s is unavailable", mode.Label())
	}
	if w.CurrentDisplayMode() == mode && w.GetAttrib(glfw.Maximized) == 0 {
		return nil
	}
	x, y, width, height := w.windowWorkarea()
	if w.GetAttrib(glfw.Maximized) != 0 {
		w.Restore()
	}
	// Keep the title bar reachable for native-size choices that extend beyond
	// the work area once decorations are included. Preserve the requested size.
	if glfw.GetPlatform() == glfw.PlatformWayland {
		w.SetSize(mode.Width, mode.Height)
	} else {
		w.SetMonitor(nil, x+max(0, (width-mode.Width)/2), y+max(0, (height-mode.Height)/2), mode.Width, mode.Height, glfw.DontCare)
	}
	w.normalSize.Width, w.normalSize.Height = mode.Width, mode.Height
	w.maximized = false
	return nil
}
