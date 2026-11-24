// Package platform owns the OS window, the OpenGL 4.5 core context and
// input. GLFW requires every call on the main thread: call LockMainThread
// from an init function of package main.
package platform

import (
	"fmt"
	"log"
	"runtime"

	"github.com/go-gl/glfw/v3.4/glfw"
)

// LockMainThread pins the calling goroutine (main) to the OS main thread.
func LockMainThread() { runtime.LockOSThread() }

type Window struct {
	*glfw.Window
	rememberSize bool
	normalSize   windowState
	maximized    bool
}

type Config struct {
	Title         string
	Width, Height int
	Debug         bool // GL debug context
	VSync         bool
	Hidden        bool // offscreen checks (screenshots)
	RememberSize  bool // persist ordinary desktop windows only
	RestoreSize   bool // explicit command-line dimensions override saved size
}

// Open initializes GLFW and creates a window with a GL 4.5 core context.
func Open(c Config) (*Window, error) {
	prepareGPU() // driver selection must precede GLFW/context initialization
	var saved windowState
	if c.RememberSize && c.RestoreSize && !c.Hidden {
		var err error
		saved, err = loadWindowState()
		if err != nil {
			log.Printf("restore window size: %v", err)
		} else if saved.Width != 0 {
			c.Width, c.Height = saved.Width, saved.Height
		}
	}
	log.Print("Initializing GLFW")
	if err := glfw.Init(); err != nil {
		return nil, fmt.Errorf("initialize GLFW: %w", err)
	}
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 5)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	if c.Debug {
		glfw.WindowHint(glfw.OpenGLDebugContext, glfw.True)
	}
	glfw.WindowHint(glfw.Resizable, glfw.True)
	if c.Hidden {
		glfw.WindowHint(glfw.Visible, glfw.False)
	}
	contextAPI := "OpenGL"
	if glfw.GetPlatform() == glfw.PlatformWin32 {
		contextAPI = "WGL"
	}
	log.Printf("Creating %s context: OpenGL 4.5 core forward-compatible, window=%dx%d debug=%t hidden=%t vsync=%t",
		contextAPI, c.Width, c.Height, c.Debug, c.Hidden, c.VSync)
	w, err := glfw.CreateWindow(c.Width, c.Height, c.Title, nil, nil)
	if err != nil {
		glfw.Terminate()
		return nil, fmt.Errorf("create %s context (OpenGL 4.5 core is required): %w", contextAPI, err)
	}
	w.MakeContextCurrent()
	fw, fh := w.GetFramebufferSize()
	log.Printf("OpenGL context current; framebuffer=%dx%d", fw, fh)
	if c.VSync {
		glfw.SwapInterval(1)
	} else {
		glfw.SwapInterval(0)
	}
	win := &Window{Window: w, rememberSize: c.RememberSize && !c.Hidden,
		normalSize: windowState{Width: c.Width, Height: c.Height}}
	if win.rememberSize {
		w.SetSizeCallback(func(_ *glfw.Window, width, height int) {
			if w.GetMonitor() == nil && width >= 320 && height >= 240 && w.GetAttrib(glfw.Maximized) == glfw.False && w.GetAttrib(glfw.Iconified) == glfw.False {
				win.normalSize.Width, win.normalSize.Height = width, height
			}
		})
		w.SetMaximizeCallback(func(_ *glfw.Window, maximized bool) { win.maximized = maximized })
		if saved.Maximized {
			w.Maximize()
			win.maximized = true
		}
	}
	return win, nil
}

func (w *Window) Close() {
	w.saveSize()
	w.Destroy()
	glfw.Terminate()
}

// Poll processes OS events; it returns false once the window should close.
func (w *Window) Poll() bool {
	glfw.PollEvents()
	return !w.ShouldClose()
}

func (w *Window) Minimized() bool { return w.GetAttrib(glfw.Iconified) != 0 }

// SetVSync enables or disables swap-interval synchronization for the current context.
func (w *Window) SetVSync(enabled bool) {
	if enabled {
		glfw.SwapInterval(1)
	} else {
		glfw.SwapInterval(0)
	}
}

// Time is seconds since GLFW initialization.
func Time() float64 { return glfw.GetTime() }
