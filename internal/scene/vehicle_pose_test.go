package scene

import (
	"testing"

	"github.com/Spelunkern/Shaiya-GX/internal/data/model"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

func TestVehicleSeatUsesCurrentPoseBeforeDrawing(t *testing.T) {
	clip := &Clip{ani: &model.ANI{End: 30, Bones: []model.AniBone{{Parent: -1, Trans: []model.TransKey{
		{Frame: 0, T: [3]float32{0, 1, 0}},
		{Frame: 30, T: [3]float32{0, 3, 0}},
	}}}}, restLocal: []render.M4{render.Identity()}}
	c := &Character{clip: clip, World: render.Identity()}
	v := &Vehicle{CreatureActor: &CreatureActor{Character: c}, seatBone: 0}
	if got := v.Seat()[1]; got != 1 {
		t.Fatalf("initial seat height = %v", got)
	}
	first := c.finals()
	if &first[0] != &c.finals()[0] {
		t.Fatal("unchanged pose was sampled again")
	}
	c.Update(.5)
	if got := v.Seat()[1]; got != 2 {
		t.Fatalf("seat did not follow the new pose: %v", got)
	}
	c.clip = &Clip{ani: &model.ANI{End: 30, Bones: []model.AniBone{{Parent: -1, Trans: []model.TransKey{{T: [3]float32{0, 4, 0}}}}}}, restLocal: []render.M4{render.Identity()}}
	if got := v.Seat()[1]; got != 4 {
		t.Fatalf("seat retained a pose from the old clip: %v", got)
	}
}
