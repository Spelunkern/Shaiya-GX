package scene

import "github.com/Spelunkern/Shaiya-GX/internal/render"

// EFT Unknown29 is mirrorTexture. Subdivide the native mirrored-repeat quad
// as Phoenix does: U=1,0,1 and V=0,1,0 across its two axes. This preserves
// the inward halo without changing the shared texture's sampler or 3DE UVs.
func mirroredEffectQuad() ([]render.Vertex, []uint32) {
	vertices := make([]render.Vertex, 0, 9)
	for y := 0; y < 3; y++ {
		for x := 0; x < 3; x++ {
			vertices = append(vertices, render.Vertex{Pos: render.V3{float32(x)*.5 - .5, float32(y)*.5 - .5, 0}, UV: [2]float32{float32(1 - x%2), float32(y % 2)}})
		}
	}
	indices := make([]uint32, 0, 24)
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			i := uint32(y*3 + x)
			indices = append(indices, i, i+1, i+3, i+3, i+1, i+4)
		}
	}
	return vertices, indices
}
