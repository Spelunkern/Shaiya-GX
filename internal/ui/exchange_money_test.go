package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"testing"
)

func TestExchangeMoneyNumericAndZero(t *testing.T) {
	m := &ExchangeMoney{}
	m.Open(800, 600, 123)
	value, done, cancel := m.Update(&platform.Input{Chars: []rune{'5', 'x', '0'}, Keys: []platform.Key{platform.KeyEnter}})
	if !done || cancel || value != 50 {
		t.Fatal(value, done, cancel)
	}
	m.Open(800, 600, 123)
	value, done, cancel = m.Update(&platform.Input{Keys: []platform.Key{platform.KeyDelete, platform.KeyEnter}})
	if !done || cancel || value != 0 {
		t.Fatal(value, done, cancel)
	}
	m.Open(800, 600, 123)
	_, done, cancel = m.Update(&platform.Input{Keys: []platform.Key{platform.KeyEscape}})
	if !done || !cancel {
		t.Fatal("cancel")
	}
}
