package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/itemstack"
	"strconv"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

const QuantityWidth, QuantityHeight = 164, 104

// Quantity is Game.exe 0x426b30/0x4268f0/0x426740, shared by stack trades.
// Original textures/positions; Latin numeric editing uses the local input queue.
type Quantity struct {
	frame            *render.Texture
	buttons          [5]Button // OK, cancel, increment, decrement, close
	X, Y             int
	value, limit     int
	editing, replace bool
	draft            string
}

func NewQuantity(t Textures) (*Quantity, error) {
	ld := &loader{t: t}
	q := &Quantity{frame: ld.get("change_item.png")}
	tex := ld.get("stats_on_button.png")
	for i := 0; i < 2; i++ {
		b := &q.buttons[i]
		*b = Button{Tex: tex, OffX: 29 + i*59, OffY: 69, DrawW: 64, DrawH: 32, HitW: 46, HitH: 20}
		for state, col := range []int{0, 1, 2, 2, 3} {
			b.UV[state] = UV{float32(col) * .25, float32(i) * .5, float32(col+1) * .25, float32(i+1) * .5}
		}
	}
	tex = ld.get("stats_on_select_button.png")
	for i := 0; i < 2; i++ {
		b := &q.buttons[i+2]
		*b = Button{Tex: tex, OffX: 108 + i*21, OffY: 31, DrawW: 32, DrawH: 32, HitW: 22, HitH: 22}
		for state, col := range []int{0, 1, 2, 2, 3} {
			b.UV[state] = UV{float32(col) * .25, float32(i) * .5, float32(col+1) * .25, float32(i+1) * .5}
		}
	}
	q.buttons[4] = nativeClose(ld.get("talk1_close_button.png"), 138, 1)
	return q, ld.err
}
func (q *Quantity) Open(mx, my, w, h, limit int) {
	q.X, q.Y = max(0, min(mx-82, w-QuantityWidth)), max(0, min(my-50, h-QuantityHeight))
	q.limit = max(1, min(limit, itemstack.Max))
	q.value, q.editing = q.limit, false
	q.draft = strconv.Itoa(q.value)
	for i := range q.buttons {
		q.buttons[i].held = false
		q.buttons[i].state = StateNormal
	}
}
func (q *Quantity) commit() {
	n, err := strconv.Atoi(q.draft)
	if err != nil {
		n = 1
	}
	q.value = max(1, min(n, q.limit))
	q.draft = strconv.Itoa(q.value)
	q.editing = false
}

// Update returns 0 while open, -1 on cancel, or the confirmed quantity.
func (q *Quantity) Update(in *platform.Input) int {
	if in.HasKey(platform.KeyEscape) {
		return -1
	}
	field := in.MouseX >= q.X+25 && in.MouseX <= q.X+106 && in.MouseY >= q.Y+35 && in.MouseY <= q.Y+49
	if in.Pressed {
		if field {
			q.editing, q.replace = true, true
		} else if q.editing {
			q.commit()
		}
	}
	if q.editing {
		if in.HasKey(platform.KeyDelete) {
			q.draft = ""
			q.replace = false
		}
		if in.HasKey(platform.KeyBackspace) {
			if q.replace {
				q.draft = ""
			} else if len(q.draft) > 0 {
				q.draft = q.draft[:len(q.draft)-1]
			}
			q.replace = false
		}
		for _, r := range in.Chars {
			if r < '0' || r > '9' {
				continue
			}
			if q.replace {
				q.draft = ""
				q.replace = false
			}
			if len(q.draft) < 3 {
				q.draft += string(r)
			}
		}
		if in.HasKey(platform.KeyEnter) {
			q.commit()
		}
	}
	for i := range q.buttons {
		b := &q.buttons[i]
		b.OriginX, b.OriginY = q.X, q.Y
		if !b.Update(in) {
			continue
		}
		q.commit()
		switch i {
		case 0:
			return q.value
		case 1, 4:
			return -1
		case 2:
			q.value = q.value%q.limit + 1
		case 3:
			q.value = (q.value+q.limit-2)%q.limit + 1
		}
		q.draft = strconv.Itoa(q.value)
	}
	return 0
}
func (q *Quantity) Draw(s *render.Sprites, f *Font) {
	s.Draw(q.frame, float32(q.X), float32(q.Y), 164, 104, 0, 0, 164.0/256, 104.0/128, white)
	text := strconv.Itoa(q.value)
	if q.editing {
		text = q.draft + "_"
	}
	f.Draw(s, text, q.X+30, q.Y+37, white, 76)
	for i := range q.buttons {
		b := &q.buttons[i]
		b.OriginX, b.OriginY = q.X, q.Y
		b.Draw(s)
	}
}
