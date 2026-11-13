package world

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// WorldMapRegion is one authored WorldMap.cfg row. Coordinates are relative
// to the native 512x512 wm_map image; Country 0/1/2 means both/Light/Fury.
type WorldMapRegion struct {
	Country, Map             int
	Left, Top, Right, Bottom int
	ImageX, ImageY           int
}

func ParseWorldMap(raw []byte) ([]WorldMapRegion, error) {
	sections := map[string]map[string]int{}
	var section string
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.ToLower(strings.TrimSpace(line[1 : len(line)-1]))
			sections[section] = map[string]int{}
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok || section == "" {
			continue
		}
		number, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return nil, fmt.Errorf("WorldMap.cfg %s/%s: %w", section, key, err)
		}
		sections[section][strings.ToLower(strings.TrimSpace(key))] = number
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	count := sections["worldmap"]["count"]
	if count < 1 || count > 4096 {
		return nil, fmt.Errorf("WorldMap.cfg invalid count %d", count)
	}
	out := make([]WorldMapRegion, 0, count)
	for i := 0; i < count; i++ {
		row := sections[strconv.Itoa(i)]
		if row == nil {
			return nil, fmt.Errorf("WorldMap.cfg missing region %d", i)
		}
		for _, key := range []string{"country", "mapnum", "click_left", "click_top", "click_right", "click_bottom", "imgpos_x", "imgpos_y"} {
			if _, ok := row[key]; !ok {
				return nil, fmt.Errorf("WorldMap.cfg region %d missing %s", i, key)
			}
		}
		r := WorldMapRegion{row["country"], row["mapnum"], row["click_left"], row["click_top"], row["click_right"], row["click_bottom"], row["imgpos_x"], row["imgpos_y"]}
		if r.Country < 0 || r.Country > 2 || r.Map < -1 || r.Right < r.Left || r.Bottom < r.Top {
			return nil, fmt.Errorf("WorldMap.cfg invalid region %d", i)
		}
		out = append(out, r)
	}
	return out, nil
}
func (r WorldMapRegion) Contains(x, y float32) bool {
	return x >= float32(r.Left) && x <= float32(r.Right) && y >= float32(r.Top) && y <= float32(r.Bottom)
}
func (r WorldMapRegion) Available(fury bool) bool {
	return r.Country == 0 || r.Country == 1 && !fury || r.Country == 2 && fury
}
