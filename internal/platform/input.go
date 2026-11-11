package platform

import "github.com/go-gl/glfw/v3.4/glfw"

// Key is a subset of keys the UI uses.
type Key int

const (
	KeyNone Key = iota
	KeyEnter
	KeyTab
	KeyBackspace
	KeyEscape
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeyDelete
)

// Input is one frame of input: the state plus the events since last frame.
type Input struct {
	MouseX, MouseY int
	MouseDown      bool // left button held
	Clicked        bool // left button released this frame
	Pressed        bool // left button pressed this frame
	Chars          []rune
	Keys           []Key
	Shift          bool
	RightDown      bool    // right button held
	RightPressed   bool    // right button pressed this frame
	Wheel          float64 // wheel delta in Win32 units (120 per notch)
	held           func(rune) bool
}

// Held reports whether a letter key or an arrow ('<' '>' '^' 'v') is down.
func (in *Input) Held(k rune) bool { return in.held != nil && in.held(k) }

func (in *Input) HasKey(k Key) bool {
	for _, x := range in.Keys {
		if x == k {
			return true
		}
	}
	return false
}

var keyMap = map[glfw.Key]Key{
	glfw.KeyEnter: KeyEnter, glfw.KeyKPEnter: KeyEnter, glfw.KeyTab: KeyTab,
	glfw.KeyBackspace: KeyBackspace, glfw.KeyEscape: KeyEscape, glfw.KeyUp: KeyUp,
	glfw.KeyDown: KeyDown, glfw.KeyLeft: KeyLeft, glfw.KeyRight: KeyRight, glfw.KeyDelete: KeyDelete,
}

// InstallInput registers callbacks; call Frame once per frame.
func (w *Window) InstallInput() *InputQueue {
	q := &InputQueue{}
	w.SetCharCallback(func(_ *glfw.Window, r rune) { q.cur.Chars = append(q.cur.Chars, r) })
	w.SetKeyCallback(func(_ *glfw.Window, k glfw.Key, _ int, a glfw.Action, mods glfw.ModifierKey) {
		q.cur.Shift = mods&glfw.ModShift != 0
		if a == glfw.Release {
			return
		}
		if mk, ok := keyMap[k]; ok {
			q.cur.Keys = append(q.cur.Keys, mk)
		}
	})
	w.SetScrollCallback(func(_ *glfw.Window, _, dy float64) { q.cur.Wheel += dy * 120 })
	w.SetMouseButtonCallback(func(_ *glfw.Window, b glfw.MouseButton, a glfw.Action, _ glfw.ModifierKey) {
		if b == glfw.MouseButtonRight {
			q.cur.RightDown = a == glfw.Press
			if a == glfw.Press {
				q.cur.RightPressed = true
			}
			return
		}
		if b != glfw.MouseButtonLeft {
			return
		}
		if a == glfw.Press {
			q.cur.Pressed, q.cur.MouseDown = true, true
		} else if a == glfw.Release {
			q.cur.Clicked, q.cur.MouseDown = true, false
		}
	})
	q.win = w
	return q
}

type InputQueue struct {
	win *Window
	cur Input
}

// Frame returns this frame's input and starts collecting the next one.
// Mouse coordinates are in framebuffer pixels.
func (q *InputQueue) Frame() Input {
	x, y := q.win.GetCursorPos()
	ww, _ := q.win.GetSize()
	fw, _ := q.win.GetFramebufferSize()
	scale := 1.0
	if ww > 0 {
		scale = float64(fw) / float64(ww)
	}
	in := q.cur
	in.MouseX, in.MouseY = int(x*scale), int(y*scale)
	win := q.win
	in.held = func(k rune) bool {
		var key glfw.Key
		switch {
		case k >= 'A' && k <= 'Z':
			key = glfw.Key(int(glfw.KeyA) + int(k-'A'))
		case k == '<':
			key = glfw.KeyLeft
		case k == '>':
			key = glfw.KeyRight
		case k == '^':
			key = glfw.KeyUp
		case k == 'v':
			key = glfw.KeyDown
		default:
			return false
		}
		return win.GetKey(key) == glfw.Press
	}
	q.cur = Input{MouseDown: in.MouseDown, RightDown: in.RightDown, Shift: in.Shift}
	return in
}
