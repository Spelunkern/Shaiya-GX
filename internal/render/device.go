// Package render draws with OpenGL 4.5 core (DSA, GLSL 450). One renderer
// per pass; states mirror the original D3D9 ones documented in
// docs/apariencia.md.
package render

import (
	"fmt"
	"log"
	"strings"
	"unsafe"

	"github.com/go-gl/gl/v4.5-core/gl"
)

// Init loads GL entry points; debug installs a KHR_debug logger.
func Init(debug bool) error {
	if err := gl.Init(); err != nil {
		return fmt.Errorf("gl: %w", err)
	}
	var major, minor int32
	gl.GetIntegerv(gl.MAJOR_VERSION, &major)
	gl.GetIntegerv(gl.MINOR_VERSION, &minor)
	if major < 4 || (major == 4 && minor < 5) {
		return fmt.Errorf("OpenGL 4.5 is required, the driver offers %d.%d", major, minor)
	}
	log.Printf("OpenGL %s, %s", gl.GoStr(gl.GetString(gl.VERSION)), gl.GoStr(gl.GetString(gl.RENDERER)))
	gpuBytes, drawCalls = 0, 0
	gpuCapacity = 0
	var extensionCount int32
	gl.GetIntegerv(gl.NUM_EXTENSIONS, &extensionCount)
	for i := int32(0); i < extensionCount; i++ {
		if gl.GoStr(gl.GetStringi(gl.EXTENSIONS, uint32(i))) == "GL_NVX_gpu_memory_info" {
			var kib int32
			gl.GetIntegerv(0x9047, &kib) // GPU_MEMORY_INFO_DEDICATED_VIDMEM_NVX
			if kib > 0 {
				gpuCapacity = uint64(kib) * 1024
			}
			break
		}
	}
	fallbackDiffuse = nil
	if debug {
		gl.Enable(gl.DEBUG_OUTPUT)
		gl.Enable(gl.DEBUG_OUTPUT_SYNCHRONOUS)
		gl.DebugMessageCallback(func(source, gltype, id, severity uint32, length int32, message string, _ unsafe.Pointer) {
			if severity == gl.DEBUG_SEVERITY_NOTIFICATION {
				return
			}
			log.Printf("gl debug [%#x]: %s", severity, message)
		}, nil)
	}
	return nil
}

func compile(kind uint32, src string) (uint32, error) {
	s := gl.CreateShader(kind)
	cs, free := gl.Strs(src + "\x00")
	gl.ShaderSource(s, 1, cs, nil)
	free()
	gl.CompileShader(s)
	var ok int32
	gl.GetShaderiv(s, gl.COMPILE_STATUS, &ok)
	if ok == gl.FALSE {
		var n int32
		gl.GetShaderiv(s, gl.INFO_LOG_LENGTH, &n)
		msg := strings.Repeat("\x00", int(n+1))
		gl.GetShaderInfoLog(s, n, nil, gl.Str(msg))
		gl.DeleteShader(s)
		return 0, fmt.Errorf("shader: %s", strings.TrimRight(msg, "\x00"))
	}
	return s, nil
}

// Program links a vertex + fragment program.
func Program(vs, fs string) (uint32, error) {
	v, err := compile(gl.VERTEX_SHADER, vs)
	if err != nil {
		return 0, err
	}
	defer gl.DeleteShader(v)
	f, err := compile(gl.FRAGMENT_SHADER, fs)
	if err != nil {
		return 0, err
	}
	defer gl.DeleteShader(f)
	p := gl.CreateProgram()
	gl.AttachShader(p, v)
	gl.AttachShader(p, f)
	gl.LinkProgram(p)
	var ok int32
	gl.GetProgramiv(p, gl.LINK_STATUS, &ok)
	if ok == gl.FALSE {
		var n int32
		gl.GetProgramiv(p, gl.INFO_LOG_LENGTH, &n)
		msg := strings.Repeat("\x00", int(n+1))
		gl.GetProgramInfoLog(p, n, nil, gl.Str(msg))
		return 0, fmt.Errorf("link: %s", strings.TrimRight(msg, "\x00"))
	}
	return p, nil
}
