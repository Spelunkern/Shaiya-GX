package platform

import (
	"github.com/go-gl/glfw/v3.4/glfw"
	"image"
)

// Cursors owns OS cursor handles; its lifetime must end before glfw.Terminate.
type Cursors struct {
	handles [7]*glfw.Cursor
	window  *Window
	current int
}

func NewCursors(w *Window) *Cursors { return &Cursors{window: w, current: -1} }
func (c *Cursors) Load(index int, pixels []byte, width, height, hx, hy int) {
	if index < 0 || index >= len(c.handles) {
		return
	}
	if old := c.handles[index]; old != nil {
		old.Destroy()
	}
	c.handles[index] = glfw.CreateCursor(&image.NRGBA{Pix: pixels, Stride: width * 4, Rect: image.Rect(0, 0, width, height)}, hx, hy)
	c.current = -1
}
func (c *Cursors) Set(index int) {
	if index < 0 || index >= len(c.handles) || c.handles[index] == nil {
		index = 0
	}
	if index == c.current {
		return
	}
	c.window.SetCursor(c.handles[index])
	c.current = index
}
func (c *Cursors) Close() {
	c.window.SetCursor(nil)
	for i, h := range c.handles {
		if h != nil {
			h.Destroy()
			c.handles[i] = nil
		}
	}
}
