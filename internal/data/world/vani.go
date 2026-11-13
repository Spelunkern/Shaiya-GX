package world

import "fmt"

// VANI contains a complete vertex pose for each animation frame.
type VANI struct {
	Center            Vec3
	Radius            float32
	Bounds            Box
	FrameCount        int
	FrameMilliseconds int32
	Meshes            []VANIMesh
	FinalBounds       Box
}

type VANIMesh struct {
	Texture string
	Faces   []Face
	Frames  [][]Vertex
}

func ParseVANI(data []byte) (*VANI, error) {
	r := &reader{b: data}
	v := &VANI{Center: r.vec3(), Radius: r.f32(), Bounds: r.box()}
	meshes, frames := r.i32(), r.i32()
	v.FrameMilliseconds = r.i32() // native interval between complete poses
	if r.err != nil {
		return nil, r.err
	}
	if meshes < 0 || meshes > 65536 || frames <= 0 || frames > 65536 || int64(meshes)*12 > int64(len(data)-r.off) {
		return nil, fmt.Errorf("vani: invalid mesh/frame counts %d/%d", meshes, frames)
	}
	v.FrameCount = int(frames)
	v.Meshes = make([]VANIMesh, 0, int(meshes))
	for i := int32(0); i < meshes && r.err == nil; i++ {
		m := VANIMesh{Texture: r.lstring(), Faces: r.faces()}
		vertices := r.count(int(frames) * 36)
		m.Frames = make([][]Vertex, int(frames))
		for f := range m.Frames {
			m.Frames[f] = make([]Vertex, vertices)
			for j := range m.Frames[f] {
				m.Frames[f][j] = Vertex{Pos: r.vec3(), Normal: r.vec3(), Bone: r.i32(), UV: r.vec2()}
			}
		}
		for _, face := range m.Faces {
			if int(face[0]) >= vertices || int(face[1]) >= vertices || int(face[2]) >= vertices {
				return nil, fmt.Errorf("vani: face index outside %d vertices", vertices)
			}
		}
		v.Meshes = append(v.Meshes, m)
	}
	v.FinalBounds = r.box()
	r.i32()
	if r.err != nil {
		return nil, r.err
	}
	return v, nil
}
