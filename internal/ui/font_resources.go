package ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	textFont "github.com/go-text/typesetting/font"
)

// FontSource indexes external coverage without parsing every fallback font at startup.
// Parsed faces and rune resolutions are shared by the UI fonts on their owner thread.
type FontSource struct {
	primaryData []byte
	primary     *textFont.Face
	read        func(string) ([]byte, error)
	entries     []fontResource
	resolved    map[rune]*textFont.Face
	color       map[*textFont.Face]bool
	emoji       *textFont.Face
}
type fontResource struct {
	File      string    `json:"file"`
	Ranges    [][2]rune `json:"ranges"`
	Color     bool      `json:"color,omitempty"`
	face      *textFont.Face
	attempted bool
}
type fontManifest struct {
	Primary   string         `json:"primary"`
	Fallbacks []fontResource `json:"fallbacks"`
}

func LoadFontSource(read func(string) ([]byte, error)) (*FontSource, error) {
	manifest := fontManifest{Primary: "fonts/ui.ttf"}
	if b, err := read("fonts/fonts.json"); err == nil {
		if err = json.Unmarshal(b, &manifest); err != nil {
			return nil, fmt.Errorf("font manifest: %w", err)
		}
	}
	var data []byte
	var err error
	if override := os.Getenv("SHAIYA_FONT"); override != "" {
		data, err = os.ReadFile(override)
	} else {
		data, err = read(manifest.Primary)
	}
	if err != nil {
		return nil, fmt.Errorf("external UI font %s: %w", manifest.Primary, err)
	}
	face, err := textFont.ParseTTF(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("UI font: %w", err)
	}
	return &FontSource{primaryData: data, primary: face, read: read, entries: manifest.Fallbacks, resolved: map[rune]*textFont.Face{}, color: map[*textFont.Face]bool{}}, nil
}
func (s *FontSource) load(entry *fontResource) *textFont.Face {
	if !entry.attempted {
		entry.attempted = true
		if data, err := s.read(entry.File); err == nil {
			entry.face, _ = textFont.ParseTTF(bytes.NewReader(data))
		}
		if entry.face != nil && entry.Color {
			s.color[entry.face] = true
			s.emoji = entry.face
		}
	}
	return entry.face
}
func (s *FontSource) emojiFace() *textFont.Face {
	if s.emoji != nil {
		return s.emoji
	}
	for i := range s.entries {
		if s.entries[i].Color {
			return s.load(&s.entries[i])
		}
	}
	return nil
}
func (s *FontSource) ResolveFace(r rune) *textFont.Face {
	if face := s.resolved[r]; face != nil {
		return face
	}
	if _, ok := s.primary.NominalGlyph(r); ok {
		s.resolved[r] = s.primary
		return s.primary
	}
	for i := range s.entries {
		entry := &s.entries[i]
		index := sort.Search(len(entry.Ranges), func(i int) bool { return entry.Ranges[i][1] >= r })
		if index == len(entry.Ranges) || entry.Ranges[index][0] > r {
			continue
		}
		if face := s.load(entry); face != nil {
			if _, ok := face.NominalGlyph(r); ok {
				s.resolved[r] = face
				return face
			}
		}
	}
	s.resolved[r] = s.primary
	return s.primary
}
func (s *FontSource) Load(height int) (*Font, error) {
	f, err := loadFontData(s.primaryData, height, false)
	if err == nil {
		f.unicode.primary = s.primary
		f.unicode.source = s
	}
	return f, err
}
func externalFontSource() (*FontSource, error) {
	dir := os.Getenv("SHAIYA_DATA")
	if dir == "" {
		if exe, err := os.Executable(); err == nil {
			dir = filepath.Join(filepath.Dir(exe), "data")
		}
	}
	return LoadFontSource(func(name string) ([]byte, error) { return os.ReadFile(filepath.Join(dir, filepath.FromSlash(name))) })
}
