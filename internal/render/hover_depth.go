package render

import "github.com/go-gl/gl/v4.5-core/gl"

type hoverDepthSample struct {
	frame               uint64
	x, y, width, height int
	reversed            bool
	value               float32
}
type hoverDepthSlot struct {
	sample hoverDepthSample
	buffer uint32
	fence  uintptr
}
type hoverDepthReadback struct {
	slots  [3]hoverDepthSlot
	ready  hoverDepthSample
	issued uint64
}

// NameHoverVisibleAt never waits for GPU work. A tiny PBO ring copies depth,
// and zero-timeout fences allow reading only completed transfers. Until a
// matching sample arrives, the cursor stays neutral. Clicks use NameVisibleAt.
func (s *Sprites) NameHoverVisibleAt(x, y, width, height int, depth float32) bool {
	if s == nil || !s.depthValid || s.depthTextureID == 0 || s.depthW != width || s.depthH != height || x < 0 || y < 0 || x >= width || y >= height {
		return false
	}
	r := &s.hoverDepth
	if r.issued != s.depthFrame {
		var available *hoverDepthSlot
		for n := range r.slots {
			slot := &r.slots[n]
			if slot.fence != 0 {
				status := gl.ClientWaitSync(slot.fence, 0, 0)
				if status == gl.ALREADY_SIGNALED || status == gl.CONDITION_SATISFIED {
					var value float32
					gl.GetNamedBufferSubData(slot.buffer, 0, 4, gl.Ptr(&value))
					slot.sample.value = value
					if slot.sample.reversed {
						slot.sample.value = 1 - slot.sample.value
					}
					if slot.sample.frame > r.ready.frame {
						r.ready = slot.sample
					}
					gl.DeleteSync(slot.fence)
					slot.fence = 0
				} else if status == gl.WAIT_FAILED {
					gl.DeleteSync(slot.fence)
					slot.fence = 0
				}
			}
			if slot.fence == 0 && available == nil {
				available = slot
			}
		}
		if available != nil {
			if available.buffer == 0 {
				gl.CreateBuffers(1, &available.buffer)
				gl.NamedBufferData(available.buffer, 4, nil, gl.STREAM_READ)
				addGPUBytes(4)
			}
			available.sample = hoverDepthSample{frame: s.depthFrame, x: x, y: y, width: width, height: height, reversed: s.depthReversed}
			gl.BindBuffer(gl.PIXEL_PACK_BUFFER, available.buffer)
			gl.GetTextureSubImage(s.depthTextureID, 0, int32(x), int32(height-1-y), 0, 1, 1, 1, gl.DEPTH_COMPONENT, gl.FLOAT, 4, gl.PtrOffset(0))
			gl.BindBuffer(gl.PIXEL_PACK_BUFFER, 0)
			available.fence = gl.FenceSync(gl.SYNC_GPU_COMMANDS_COMPLETE, 0)
		}
		r.issued = s.depthFrame
	}
	sample := r.ready
	return sample.frame != 0 && s.depthFrame-sample.frame <= 3 && sample.x == x && sample.y == y && sample.width == width && sample.height == height && sample.reversed == s.depthReversed && sample.value+.000001 >= depth
}

// ResetHoverDepth drops samples across map changes, resizes and shutdown.
func (s *Sprites) ResetHoverDepth() {
	if s == nil {
		return
	}
	for _, slot := range s.hoverDepth.slots {
		if slot.fence != 0 {
			gl.DeleteSync(slot.fence)
		}
		if slot.buffer != 0 {
			gl.DeleteBuffers(1, &slot.buffer)
			removeGPUBytes(4)
		}
	}
	s.hoverDepth = hoverDepthReadback{}
}
