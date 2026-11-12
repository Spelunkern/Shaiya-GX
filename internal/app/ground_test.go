package app

import (
	"math"
	"testing"
)

func TestGroundBrightness(t *testing.T) {
	for _, c := range []struct {
		age     float64
		typ, id uint8
		want    float64
	}{
		{0, 25, 1, .3}, {.175, 25, 1, .65}, {.35, 25, 1, 1},
		{.525, 25, 1, .65}, {.7, 25, 1, .3}, {5, 44, 246, 1},
	} {
		if got := groundBrightness(c.age, c.typ, c.id); math.Abs(float64(got)-c.want) > .0001 {
			t.Errorf("age %v type %d/%d: got %v, want %v", c.age, c.typ, c.id, got, c.want)
		}
	}
}
