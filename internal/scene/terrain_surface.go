package scene

// terrainFanHeight reconstructs EP5 0x5b3720: four planes around the centre
// of a 4x4-sample square (8x8 world units). The original ROAM ceiling uses
// these same five vertices. Intermediate source samples are not mesh detail.
// See docs/terrain.md for executable identity and native-oracle validation.
func terrainFanHeight(heights []float32, side int, x, z float32) float32 {
	x = min(max(x, 0), float32(side-1))
	z = min(max(z, 0), float32(side-1))
	ix, iz := min(int(x)/4*4, side-5), min(int(z)/4*4, side-5)
	dx, dz := x-float32(ix), z-float32(iz)
	a, b := heights[iz*side+ix], heights[iz*side+ix+4]
	c, d := heights[(iz+4)*side+ix], heights[(iz+4)*side+ix+4]
	m := heights[(iz+2)*side+ix+2]
	if dz <= dx {
		if 4-dx < dz {
			return b + (4-dx)*(m-b)/2 + (dz-(4-dx))*(d-b)/4
		}
		return a + dx*(b-a)/4 + dz*(m-(a+b)/2)/2
	}
	if 4-dx < dz {
		return c + dx*(d-c)/4 + (4-dz)*(m-(c+d)/2)/2
	}
	return a + dz*(c-a)/4 + dx*(m-(a+c)/2)/2
}
