package render

import (
	"unsafe"

	"github.com/go-gl/gl/v4.5-core/gl"

	img "github.com/Spelunkern/Shaiya-GX/internal/data/image"
)

type Texture struct {
	Opaque bool // proven full alpha coverage at every authored mip
	bytes  uint64
	ID     uint32
	W, H   int
}

var fallbackDiffuse *Texture

// A missing/untextured material must never inherit the preceding draw's
// sampler. D3D's untextured diffuse stage corresponds to multiplying by white.
func bindDiffuse(t *Texture) {
	if t == nil {
		if fallbackDiffuse == nil {
			fallbackDiffuse = Upload(img.Texture{RGBA: &img.RGBA{W: 1, H: 1, Pix: []byte{255, 255, 255, 255}}})
		}
		t = fallbackDiffuse
	}
	gl.BindTextureUnit(0, t.ID)
}

// Upload creates an immutable texture (DSA). DXT data is uploaded as-is.
func Upload(t img.Texture) *Texture { return uploadTexture(t, false) }

// UploadWorld requests the full mip chain like Game.exe 0x5aff00/0x5ac500.
// Existing authored levels remain byte-identical; only missing levels are built.
func UploadWorld(t img.Texture) *Texture { return uploadTexture(t, true) }

func fullMipCount(w, h int) int32 {
	n := int32(1)
	for w > 1 || h > 1 {
		w, h = max(w/2, 1), max(h/2, 1)
		n++
	}
	return n
}

func uploadTexture(t img.Texture, world bool) *Texture {
	var id uint32
	gl.CreateTextures(gl.TEXTURE_2D, 1, &id)
	tex := &Texture{ID: id}
	if world {
		tex.Opaque = img.Opaque(t)
	}
	if t.DDS != nil {
		d := t.DDS
		tex.W, tex.H = d.W, d.H
		levels := int32(len(d.Levels))
		authored := levels
		if world {
			levels = fullMipCount(d.W, d.H)
		}
		block := 0
		if d.Format != img.FormatRGBA {
			block = 16
			if d.Format == img.FormatDXT1 {
				block = 8
			}
		}
		tex.bytes = textureStorageBytes(d.W, d.H, int(levels), block)
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
		if levels > authored {
			gl.TextureParameteri(id, gl.TEXTURE_BASE_LEVEL, authored-1)
			gl.GenerateTextureMipmap(id)
			gl.TextureParameteri(id, gl.TEXTURE_BASE_LEVEL, 0)
		}
		if levels > 1 {
			gl.TextureParameteri(id, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_LINEAR)
		} else {
			gl.TextureParameteri(id, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
		}
	} else {
		r := t.RGBA
		tex.W, tex.H = r.W, r.H
		levels := int32(1)
		if world {
			levels = fullMipCount(r.W, r.H)
		}
		tex.bytes = textureStorageBytes(r.W, r.H, int(levels), 0)
		gl.TextureStorage2D(id, levels, gl.RGBA8, int32(r.W), int32(r.H))
		gl.TextureSubImage2D(id, 0, 0, 0, int32(r.W), int32(r.H), gl.RGBA, gl.UNSIGNED_BYTE, unsafe.Pointer(&r.Pix[0]))
		gl.TextureParameteri(id, gl.TEXTURE_MIN_FILTER, gl.LINEAR)
		gl.TextureParameteri(id, gl.TEXTURE_MAX_LEVEL, levels-1)
		if levels > 1 {
			gl.GenerateTextureMipmap(id)
		}
	}
	if world {
		gl.TextureParameteri(id, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_NEAREST)
	}
	gl.TextureParameteri(id, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
	gl.TextureParameteri(id, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TextureParameteri(id, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	addGPUBytes(tex.bytes)
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

func (t *Texture) Delete() {
	if t == nil {
		return
	}
	gl.DeleteTextures(1, &t.ID)
	removeGPUBytes(t.bytes)
	t.bytes = 0
	t.ID = 0
}

func textureStorageBytes(w, h, levels, block int) uint64 {
	var n uint64
	for i := 0; i < levels; i++ {
		if block == 0 {
			n += uint64(w * h * 4)
		} else {
			n += uint64(((w + 3) / 4) * ((h + 3) / 4) * block)
		}
		w, h = max(1, w/2), max(1, h/2)
	}
	return n
}
