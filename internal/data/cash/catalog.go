// Package cash defines the shared, external Item Mall catalog.
package cash

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/itemstack"
	"io"
	"strconv"
	"strings"
	"unicode/utf8"
)

const Filename = "cash.csv"
const Slots = 24
const MaxBytes = 16 << 20

type Item struct {
	ItemID uint32
	Count  uint16
}
type Product struct {
	Index, Bag, Icon, Cost  uint32
	Items                   [Slots]Item
	Name, Code, Description string
}

func Header() []string {
	fields := []string{"index", "category", "icon", "price", "code", "name", "description"}
	for i := 1; i <= Slots; i++ {
		fields = append(fields, fmt.Sprintf("item_id_%d", i), fmt.Sprintf("quantity_%d", i))
	}
	return fields
}

func Validate(products []Product) error {
	codes := map[string]bool{}
	for row, p := range products {
		fail := func(reason string) error { return fmt.Errorf("cash.csv row %d: %s", row+2, reason) }
		if len(p.Code) < 1 || len(p.Code) > 20 {
			return fail("code must contain 1..20 ASCII bytes")
		}
		for _, r := range p.Code {
			if r < 33 || r > 126 {
				return fail("code must use printable ASCII without spaces")
			}
		}
		if codes[p.Code] {
			return fail("duplicate code " + p.Code)
		}
		codes[p.Code] = true
		if p.Bag > 4 {
			return fail("category must be 0 (hidden) or 1..4")
		}
		if !utf8.ValidString(p.Name) || !utf8.ValidString(p.Description) || strings.ContainsRune(p.Name, '\x00') || strings.ContainsRune(p.Description, '\x00') {
			return fail("invalid UTF-8 text or NUL character")
		}
		items := 0
		for slot, it := range p.Items {
			if it.ItemID == 0 && it.Count == 0 {
				continue
			}
			if it.ItemID == 0 || it.Count == 0 || it.Count > itemstack.Max || it.ItemID/1000 == 0 || it.ItemID/1000 > 255 || it.ItemID%1000 == 0 || it.ItemID%1000 > 255 {
				return fail(fmt.Sprintf("invalid item/quantity in slot %d", slot+1))
			}
			items++
		}
		if items == 0 && p.Cost != 0 {
			return fail("a priced product needs at least one item")
		}
	}
	return nil
}

func Parse(raw []byte) ([]Product, error) {
	if len(raw) > MaxBytes {
		return nil, fmt.Errorf("cash.csv exceeds %d bytes", MaxBytes)
	}
	raw = bytes.TrimPrefix(raw, []byte{0xef, 0xbb, 0xbf})
	if !utf8.Valid(raw) {
		return nil, fmt.Errorf("cash.csv must be UTF-8")
	}
	r := csv.NewReader(bytes.NewReader(raw))
	r.FieldsPerRecord = len(Header())
	fields, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("cash.csv header: %w", err)
	}
	for i, want := range Header() {
		if fields[i] != want {
			return nil, fmt.Errorf("cash.csv column %d: expected %q, got %q", i+1, want, fields[i])
		}
	}
	products := []Product{}
	for row := 2; ; row++ {
		values, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("cash.csv row %d: %w", row, err)
		}
		number := func(column, bits int) (uint32, error) {
			s := values[column]
			if s == "" {
				return 0, nil
			}
			n, e := strconv.ParseUint(s, 10, bits)
			if e != nil {
				return 0, fmt.Errorf("cash.csv row %d column %s: invalid unsigned integer %q", row, Header()[column], s)
			}
			return uint32(n), nil
		}
		var p Product
		ptrs := []*uint32{&p.Index, &p.Bag, &p.Icon, &p.Cost}
		for col, ptr := range ptrs {
			n, e := number(col, 32)
			if e != nil {
				return nil, e
			}
			*ptr = n
		}
		p.Code, p.Name, p.Description = values[4], values[5], values[6]
		for i := range p.Items {
			id, e := number(7+2*i, 32)
			if e != nil {
				return nil, e
			}
			count, e := number(8+2*i, 16)
			if e != nil {
				return nil, e
			}
			p.Items[i] = Item{id, uint16(count)}
		}
		products = append(products, p)
	}
	if err := Validate(products); err != nil {
		return nil, err
	}
	return products, nil
}

// Encode is canonical: UTF-8, comma separators and LF, preserving row/slot order.
func Encode(products []Product) ([]byte, error) {
	if err := Validate(products); err != nil {
		return nil, err
	}
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	w.Write(Header())
	n := func(v uint32) string { return strconv.FormatUint(uint64(v), 10) }
	for _, p := range products {
		row := []string{n(p.Index), n(p.Bag), n(p.Icon), n(p.Cost), p.Code, p.Name, p.Description}
		for _, it := range p.Items {
			row = append(row, n(it.ItemID), n(uint32(it.Count)))
		}
		if err := w.Write(row); err != nil {
			return nil, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	if b.Len() > MaxBytes {
		return nil, fmt.Errorf("cash.csv exceeds %d bytes", MaxBytes)
	}
	return b.Bytes(), nil
}

func Hash(products []Product) ([32]byte, error) {
	raw, err := Encode(products)
	if err != nil {
		return [32]byte{}, err
	}
	return sha256.Sum256(raw), nil
}
