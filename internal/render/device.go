// Package render draws with OpenGL 4.5 core (DSA, GLSL 450). One renderer
// per pass; states mirror the original D3D9 ones documented in
// docs/apariencia.md.
package render

import (
	"fmt"
	"log"
	"strings"
	"time"
	"unsafe"

	"github.com/go-gl/gl/v4.5-core/gl"
	"github.com/go-gl/glfw/v3.4/glfw"
)

var skinIndirectSupported bool
var drawParametersSupported bool

// Init loads GL entry points; debug installs a KHR_debug logger.
func Init(debug bool) error {
	log.Print("Loading OpenGL entry points")
	if err := loadGLEntryPoints(glfw.GetProcAddress); err != nil {
		return fmt.Errorf("load OpenGL entry points: %w", err)
	}
	var major, minor int32
	gl.GetIntegerv(gl.MAJOR_VERSION, &major)
	gl.GetIntegerv(gl.MINOR_VERSION, &minor)
	log.Printf("OpenGL %s, %s (vendor=%q GLSL=%q)",
		gl.GoStr(gl.GetString(gl.VERSION)), gl.GoStr(gl.GetString(gl.RENDERER)),
		gl.GoStr(gl.GetString(gl.VENDOR)), gl.GoStr(gl.GetString(gl.SHADING_LANGUAGE_VERSION)))
	if major < 4 || (major == 4 && minor < 5) {
		return fmt.Errorf("OpenGL 4.5 is required, the driver offers %d.%d", major, minor)
	}
	var profile, flags int32
	gl.GetIntegerv(gl.CONTEXT_PROFILE_MASK, &profile)
	gl.GetIntegerv(gl.CONTEXT_FLAGS, &flags)
	log.Printf("OpenGL context profile_mask=%#x flags=%#x", profile, flags)
	gpuBytes, drawCalls = 0, 0
	gpuCapacity = 0
	bindless, drawParameters, s3tc := false, false, false
	var extensionCount int32
	var atiMeminfo bool
	gl.GetIntegerv(gl.NUM_EXTENSIONS, &extensionCount)
	for i := int32(0); i < extensionCount; i++ {
		switch gl.GoStr(gl.GetStringi(gl.EXTENSIONS, uint32(i))) {
		case "GL_ARB_bindless_texture":
			bindless = true
		case "GL_ARB_shader_draw_parameters":
			drawParameters = true
		case "GL_EXT_texture_compression_s3tc":
			s3tc = true
		case "GL_NVX_gpu_memory_info":
			var kib int32
			gl.GetIntegerv(0x9047, &kib) // GPU_MEMORY_INFO_DEDICATED_VIDMEM_NVX
			if kib > 0 {
				gpuCapacity = uint64(kib) * 1024
			}
		case "GL_ATI_meminfo":
			atiMeminfo = true
		}
	}
	if gpuCapacity == 0 && atiMeminfo {
		var info [4]int32
		gl.GetIntegerv(0x87FC, &info[0]) // TEXTURE_FREE_MEMORY_ATI
		if info[0] > 0 {
			gpuCapacity = uint64(info[0]) * 1024
		}
	}
	skinIndirectSupported = bindless && drawParameters
	drawParametersSupported = drawParameters
	log.Printf("OpenGL capabilities: ARB_bindless_texture=%t ARB_shader_draw_parameters=%t EXT_texture_compression_s3tc=%t bindless_indirect_enabled=%t",
		bindless, drawParameters, s3tc, skinIndirectSupported)
	fallbackDiffuse = nil
	if debug {
		gl.Enable(gl.DEBUG_OUTPUT)
		gl.Enable(gl.DEBUG_OUTPUT_SYNCHRONOUS)
		gl.DebugMessageCallback(func(source, gltype, id, severity uint32, length int32, message string, _ unsafe.Pointer) {
			if severity == gl.DEBUG_SEVERITY_NOTIFICATION {
				return
			}
			log.Printf("gl debug [%#x] source=%#x type=%#x id=%d: %s", severity, source, gltype, id, message)
		}, nil)
	}
	return nil
}

func compile(name, stage string, kind uint32, src string) (uint32, error) {
	log.Printf("OpenGL compiling %q (%s shader)", name, stage)
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
		return 0, fmt.Errorf("compile %q (%s shader): %s", name, stage, strings.TrimRight(msg, "\x00"))
	}
	return s, nil
}

// Program compiles and links a named vertex and fragment program.
func Program(name, vs, fs string) (uint32, error) {
	v, err := compile(name, "vertex", gl.VERTEX_SHADER, vs)
	if err != nil {
		return 0, err
	}
	defer gl.DeleteShader(v)
	f, err := compile(name, "fragment", gl.FRAGMENT_SHADER, fs)
	if err != nil {
		return 0, err
	}
	defer gl.DeleteShader(f)
	return link(name, v, f)
}

func computeProgram(name, src string) (uint32, error) {
	shader, err := compile(name, "compute", gl.COMPUTE_SHADER, src)
	if err != nil {
		return 0, err
	}
	defer gl.DeleteShader(shader)
	return link(name, shader)
}

func link(name string, shaders ...uint32) (uint32, error) {
	p := gl.CreateProgram()
	for _, shader := range shaders {
		gl.AttachShader(p, shader)
	}
	// Write the program name before entering the driver in case it terminates the process.
	log.Printf("OpenGL linking %q", name)
	start := time.Now()
	gl.LinkProgram(p)
	var ok int32
	gl.GetProgramiv(p, gl.LINK_STATUS, &ok)
	if ok == gl.FALSE {
		var n int32
		gl.GetProgramiv(p, gl.INFO_LOG_LENGTH, &n)
		msg := strings.Repeat("\x00", int(n+1))
		gl.GetProgramInfoLog(p, n, nil, gl.Str(msg))
		return 0, fmt.Errorf("link %q: %s", name, strings.TrimRight(msg, "\x00"))
	}
	log.Printf("OpenGL linked %q in %s", name, time.Since(start).Round(time.Microsecond))
	return p, nil
}
