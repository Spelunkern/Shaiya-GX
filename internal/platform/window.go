// Package platform owns the OS window, the OpenGL 4.5 core context and
// input. GLFW requires every call on the main thread: call LockMainThread
// from an init function of package main.
package platform

import (
	"fmt"
	"runtime"

	"github.com/go-gl/glfw/v3.4/glfw"
)

// LockMainThread pins the calling goroutine (main) to the OS main thread.
func LockMainThread() { runtime.LockOSThread() }

type Window struct {
	*glfw.Window
}

type Config struct {
	Title         string
	Width, Height int
	Debug         bool // GL debug context
	VSync         bool
	Hidden        bool // offscreen checks (screenshots)
}

// Open initializes GLFW and creates a window with a GL 4.5 core context.
func Open(c Config) (*Window, error) {
	if err := glfw.Init(); err != nil {
		return nil, fmt.Errorf("glfw: %w", err)
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
	w, err := glfw.CreateWindow(c.Width, c.Height, c.Title, nil, nil)
	if err != nil {
		glfw.Terminate()
		return nil, fmt.Errorf("window (OpenGL 4.5 core is required): %w", err)
	}
	w.MakeContextCurrent()
	if c.VSync {
		glfw.SwapInterval(1)
	} else {
		glfw.SwapInterval(0)
	}
	return &Window{w}, nil
}

func (w *Window) Close() {
	w.Destroy()
	glfw.Terminate()
}

// Poll processes OS events; it returns false once the window should close.
func (w *Window) Poll() bool {
	glfw.PollEvents()
	return !w.ShouldClose()
}

// Time is seconds since GLFW initialization.
func Time() float64 { return glfw.GetTime() }
