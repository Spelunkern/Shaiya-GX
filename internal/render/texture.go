package render

import (
	"unsafe"

	"github.com/go-gl/gl/v4.5-core/gl"

	img "github.com/Spelunkern/shaiya-go-client/internal/data/image"
)

type Texture struct {
	ID   uint32
	W, H int
}

// Upload creates an immutable texture (DSA). DXT data is uploaded as-is.
func Upload(t img.Texture) *Texture {
	var id uint32
	gl.CreateTextures(gl.TEXTURE_2D, 1, &id)
	tex := &Texture{ID: id}
	if t.DDS != nil {
		d := t.DDS
		tex.W, tex.H = d.W, d.H
		levels := int32(len(d.Levels))
		if d.Format == img.FormatRGBA {
			gl.TextureStorage2D(id, levels, gl.RGBA8, int32(d.W), int32(d.H))
		} else {
			gl.TextureStorage2D(id, levels, dxtFormat(d.Format), int32(d.W), int32(d.H))
		}
		w, h := d.W, d.H
		for i, l := range d.Levels {
			if d.Format == img.FormatRGBA {
				gl.TextureSubImage2D(id, int32(i), 0, 0, int32(w), int32(h), gl.RGBA, gl.UNSIGNED_BYTE, unsafe.Pointer(&l[0]))
			} else {
				gl.CompressedTextureSubImage2D(id, int32(i), 0, 0, int32(w), int32(h), dxtFormat(d.Format), int32(len(l)), unsafe.Pointer(&l[0]))
			}
			w, h = max(w/2, 1), max(h/2, 1)
		}
		gl.TextureParameteri(id, gl.TEXTURE_MAX_LEVEL, levels-1)
		if levels > 1 {
			gl.TextureParameteri(id, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_LINEAR)
		} else {
			gl.TextureParameteri(id, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
		}
	} else {
		r := t.RGBA
		tex.W, tex.H = r.W, r.H
		gl.TextureStorage2D(id, 1, gl.RGBA8, int32(r.W), int32(r.H))
		gl.TextureSubImage2D(id, 0, 0, 0, int32(r.W), int32(r.H), gl.RGBA, gl.UNSIGNED_BYTE, unsafe.Pointer(&r.Pix[0]))
		gl.TextureParameteri(id, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
	}
	gl.TextureParameteri(id, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TextureParameteri(id, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TextureParameteri(id, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	return tex
}

func dxtFormat(f img.Format) uint32 {
	switch f {
	case img.FormatDXT1:
		return gl.COMPRESSED_RGBA_S3TC_DXT1_EXT
	case img.FormatDXT3:
		return gl.COMPRESSED_RGBA_S3TC_DXT3_EXT
	default:
		return gl.COMPRESSED_RGBA_S3TC_DXT5_EXT
	}
}

func (t *Texture) Delete() { gl.DeleteTextures(1, &t.ID) }
