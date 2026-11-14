package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"testing"
)

func TestQuantityLimitsEditingAndCancellation(t *testing.T) {
	q, err := NewQuantity(marketTextures{})
	if err != nil {
		t.Fatal(err)
	}
	q.Open(0, 0, 1024, 768, 20)
	if q.X != 0 || q.Y != 0 || q.value != 20 {
		t.Fatal("initial quantity/anchor", q)
	}
	click := func(x, y int) int {
		t.Helper()
		q.Update(&platform.Input{MouseX: q.X + x, MouseY: q.Y + y, Pressed: true})
		return q.Update(&platform.Input{MouseX: q.X + x, MouseY: q.Y + y, Clicked: true})
	}
	click(109, 32)
	if q.value != 1 {
		t.Fatal("increment should wrap maximum to 1")
	}
	click(130, 32)
	if q.value != 20 {
		t.Fatal("decrement should wrap 1 to maximum")
	}
	click(30, 40)
	q.Update(&platform.Input{Chars: []rune("a255"), Keys: []platform.Key{platform.KeyEnter}})
	if q.value != 20 {
		t.Fatal("numeric input exceeded limit")
	}
	click(30, 40)
	q.Update(&platform.Input{Chars: []rune("0"), Keys: []platform.Key{platform.KeyEnter}})
	if got := click(30, 70); got != 1 {
		t.Fatal("zero should confirm as 1", got)
	}
	q.Open(1024, 768, 1024, 768, 999)
	if q.value != 255 {
		t.Fatal("quantity did not clamp to 255")
	}
	if q.X != 1024-QuantityWidth || q.Y != 768-QuantityHeight {
		t.Fatal("dialog escaped screen")
	}
	click(30, 40)
	q.Update(&platform.Input{Chars: []rune("123")})
	q.Update(&platform.Input{Keys: []platform.Key{platform.KeyBackspace}})
	if got := click(30, 70); got != 12 {
		t.Fatal("backspace did not edit quantity", got)
	}
	if got := q.Update(&platform.Input{Keys: []platform.Key{platform.KeyEscape}}); got != -1 {
		t.Fatal("escape did not cancel")
	}
	if got := click(89, 70); got != -1 {
		t.Fatal("cancel did not close")
	}
	if got := click(139, 2); got != -1 {
		t.Fatal("close did not cancel")
	}
}
