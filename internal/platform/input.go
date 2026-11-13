package platform

import (
	"github.com/go-gl/glfw/v3.4/glfw"
	"strings"
)

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
	KeyNumLock
	KeyHome
	KeyPageUp
	KeyPageDown
	KeyF9
	KeyF10
	KeySpace
	KeyTarget         // Native English client: DirectInput 0x29 (grave accent).
	KeyPreviousTarget // Press edge only; Backspace still repeats inside text fields.
)

// Input is one frame of input: the state plus the events since last frame.
type Input struct {
	KeypadChars      string // character events that belong to keypad action edges
	ActionHeld       uint32 // held number-row/keypad slots, independent of repeat events
	Actions          []int  // second-bar keypad press edges, flattened slot indices
	MouseX, MouseY   int
	MouseLook        bool    // captured pointer; UI coordinates stay at the drag anchor
	MouseDX, MouseDY float64 // framebuffer displacement during captured look
	MouseDown        bool    // left button held
	Clicked          bool    // left button released this frame
	Pressed          bool    // left button pressed this frame
	Chars            []rune
	Keys             []Key
	Shift            bool
	MiddlePressed    bool    // middle button pressed this frame
	Released         string  // released A..Z, for native double-tap controls
	RightDown        bool    // right button held
	RightPressed     bool    // right button pressed this frame
	Space            bool    // space held
	Wheel            float64 // wheel delta in Win32 units (120 per notch)
	Down             string  // snapshot: held A..Z and arrow markers < > ^ v
	SuppressAutoRun  bool    // modal/chat updates pause automatic travel
}

// Held reports whether a letter key or an arrow ('<' '>' '^' 'v') is down.
func (in *Input) Held(k rune) bool { return strings.ContainsRune(in.Down, k) }

func (in *Input) HasKey(k Key) bool {
	for _, x := range in.Keys {
		if x == k {
			return true
		}
	}
	return false
}

var keyMap = map[glfw.Key]Key{
	glfw.KeyGraveAccent: KeyTarget,
	glfw.KeyEnter:       KeyEnter, glfw.KeyKPEnter: KeyEnter, glfw.KeyTab: KeyTab,
	glfw.KeyBackspace: KeyBackspace, glfw.KeyEscape: KeyEscape, glfw.KeyUp: KeyUp,
	glfw.KeyHome: KeyHome, glfw.KeyPageUp: KeyPageUp, glfw.KeyPageDown: KeyPageDown,
	glfw.KeyF9: KeyF9, glfw.KeyF10: KeyF10, glfw.KeySpace: KeySpace,
	glfw.KeyDown: KeyDown, glfw.KeyLeft: KeyLeft, glfw.KeyRight: KeyRight, glfw.KeyDelete: KeyDelete,
}

// InstallInput registers callbacks; call Frame once per frame.
func (w *Window) InstallInput() *InputQueue {
	q := &InputQueue{}
	w.SetCharCallback(func(_ *glfw.Window, r rune) {
		if q.keypadChar && r >= '0' && r <= '9' {
			q.keypadChar = false
			q.cur.KeypadChars += string(r)
		}
		q.cur.Chars = append(q.cur.Chars, r)
	})
	w.SetKeyCallback(func(_ *glfw.Window, k glfw.Key, _ int, a glfw.Action, mods glfw.ModifierKey) {
		q.cur.Shift = mods&glfw.ModShift != 0
		q.keypadChar = false
		if k >= glfw.KeyKP0 && k <= glfw.KeyKP9 {
			if a == glfw.Press {
				digit := int(k - glfw.KeyKP0)
				q.cur.Actions = append(q.cur.Actions, 10+(digit+9)%10)
				q.keypadChar = true
			}
			return
		}
		if a == glfw.Release {
			if k >= glfw.KeyA && k <= glfw.KeyZ {
				q.cur.Released += string(rune('A' + k - glfw.KeyA))
			}
			return
		}
		if k == glfw.KeyNumLock {
			if a == glfw.Press {
				q.cur.Keys = append(q.cur.Keys, KeyNumLock)
			}
			return
		}
		if k == glfw.KeyBackspace && a == glfw.Press {
			q.cur.Keys = append(q.cur.Keys, KeyPreviousTarget)
		}
		if mk, ok := keyMap[k]; ok && (a == glfw.Press || mk <= KeyDelete && mk != KeyTab) {
			q.cur.Keys = append(q.cur.Keys, mk)
		}
	})
	w.SetScrollCallback(func(_ *glfw.Window, _, dy float64) { q.cur.Wheel += dy * 120 })
	w.SetMouseButtonCallback(func(_ *glfw.Window, b glfw.MouseButton, a glfw.Action, _ glfw.ModifierKey) {
		if b == glfw.MouseButtonMiddle {
			q.cur.MiddlePressed = a == glfw.Press
			return
		}
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
	win              *Window
	cur              Input
	mouselook        bool
	keypadChar       bool
	anchorX, anchorY float64
	lookX, lookY     float64
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
	if q.mouselook {
		in.MouseLook = true
		in.MouseDX, in.MouseDY = (x-q.lookX)*scale, (y-q.lookY)*scale
		in.MouseX, in.MouseY = int(q.anchorX*scale), int(q.anchorY*scale)
		q.lookX, q.lookY = x, y
	}
	if q.win.GetAttrib(glfw.Focused) == 0 {
		in.RightDown, in.RightPressed, in.MouseDown = false, false, false
		in.MouseDX, in.MouseDY = 0, 0
	}
	var down [32]byte
	n := 0
	for key := glfw.KeyA; key <= glfw.KeyZ; key++ {
		if q.win.GetKey(key) == glfw.Press {
			down[n] = byte('A' + key - glfw.KeyA)
			n++
		}
	}
	for i, key := range []glfw.Key{glfw.KeyLeft, glfw.KeyRight, glfw.KeyUp, glfw.KeyDown, glfw.KeyPageUp, glfw.KeyPageDown} {
		if q.win.GetKey(key) == glfw.Press {
			down[n] = "<>^v+-"[i]
			n++
		}
	}
	in.Down = string(down[:n])
	in.Space = q.win.GetKey(glfw.KeySpace) == glfw.Press
	for digit := 0; digit < 10; digit++ {
		slot := (digit + 9) % 10
		if q.win.GetKey(glfw.Key0+glfw.Key(digit)) == glfw.Press {
			in.ActionHeld |= 1 << slot
		}
		if q.win.GetKey(glfw.KeyKP0+glfw.Key(digit)) == glfw.Press {
			in.ActionHeld |= 1 << (10 + slot)
		}
	}
	if q.win.GetAttrib(glfw.Focused) == 0 {
		in.Down, in.Space, in.SuppressAutoRun = "", false, true
		in.ActionHeld = 0
		in.Chars, in.Keys, in.Actions, in.Released = nil, nil, nil, ""
	}
	q.cur = Input{MouseDown: in.MouseDown, RightDown: in.RightDown, Shift: in.Shift}
	return in
}

// SetMouseLook provides unbounded right-drag deltas and restores the original
// pointer location on release, matching Game.exe 0x5316e0.
func (q *InputQueue) SetMouseLook(active bool) {
	active = active && q.win.GetAttrib(glfw.Focused) != 0
	if active == q.mouselook {
		return
	}
	q.mouselook = active
	if active {
		q.anchorX, q.anchorY = q.win.GetCursorPos()
		q.win.SetInputMode(glfw.CursorMode, glfw.CursorDisabled)
		q.lookX, q.lookY = q.win.GetCursorPos()
	} else {
		q.win.SetInputMode(glfw.CursorMode, glfw.CursorNormal)
		q.win.SetCursorPos(q.anchorX, q.anchorY)
	}
}
