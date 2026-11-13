package scene

import "testing"

func TestMirroredHaloUVsMeetAtCenter(t *testing.T) {
	v, indices := mirroredEffectQuad()
	if len(v) != 9 || len(indices) != 24 {
		t.Fatal("mirror quad lost a quadrant")
	}
	if v[4].UV != ([2]float32{0, 1}) {
		t.Fatal("halo centre uses the outside texture edge", v[4])
	}
	for i := range v {
		opposite := len(v) - 1 - i
		if v[i].UV != v[opposite].UV {
			t.Fatal("opposing halo quadrants point in different directions")
		}
	}
}
