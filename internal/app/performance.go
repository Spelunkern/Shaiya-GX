package app

import (
	"fmt"
	"math"
	"time"

	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"github.com/Spelunkern/Shaiya-GX/internal/render"
)

type performanceHUD struct {
	meter                       *render.FrameMeter
	updateCPU, drawCPU          time.Duration
	elapsed, cpuTotal, gpuTotal float64
	frames                      int
	fps, frameMS, cpuMS, gpuMS  float64
	ram, vram                   uint64
	ramTotal, vramTotal         uint64
	calls                       int
}

func (p *performanceHUD) sample(dt float64) {
	if dt <= 0 {
		return
	}
	p.elapsed += dt
	p.frames++
	p.cpuTotal += float64(p.updateCPU+p.drawCPU) / float64(time.Millisecond)
	p.gpuTotal += p.meter.Milliseconds()
	if p.elapsed < .25 {
		return
	}
	p.fps = float64(p.frames) / p.elapsed
	p.frameMS = p.elapsed * 1000 / float64(p.frames)
	p.cpuMS, p.gpuMS = p.cpuTotal/float64(p.frames), p.gpuTotal/float64(p.frames)
	p.ram, _ = platform.ResidentBytes()
	p.vram = render.GPUBytes()
	if p.ramTotal == 0 {
		p.ramTotal, _ = platform.TotalMemoryBytes()
	}
	p.vramTotal = render.GPUCapacity()
	p.calls = render.DrawCalls()
	p.elapsed, p.cpuTotal, p.gpuTotal, p.frames = 0, 0, 0, 0
}
func (a *App) drawPerformance(w int) {
	p := &a.performance
	fps := "--"
	if p.fps > 0 {
		fps = fmt.Sprintf("%.0f", p.fps)
	}
	gpu := "--"
	if p.gpuMS > 0 {
		gpu = fmt.Sprintf("%.1f", p.gpuMS)
	}
	ram := "--"
	if p.ram > 0 {
		ram = fmt.Sprintf("%.0f", float64(p.ram)/1048576)
	}
	spans := []performanceSpan{
		{fps, performanceFPSColor(p.fps)}, {" FPS | ", performanceNeutral},
		{fmt.Sprintf("%.1f", p.frameMS), performanceTimeColor(p.frameMS)}, {" ms | CPU ", performanceNeutral},
		{fmt.Sprintf("%.1f", p.cpuMS), performanceTimeColor(p.cpuMS)}, {" / GPU ", performanceNeutral},
		{gpu, performanceTimeColor(p.gpuMS)}, {" ms | RAM ", performanceNeutral},
		{ram, performanceMemoryColor(p.ram, p.ramTotal)}, {" / VRAM~", performanceNeutral},
		{fmt.Sprintf("%.0f", float64(p.vram)/1048576), performanceMemoryColor(p.vram, p.vramTotal)}, {" MiB | DC ", performanceNeutral},
		{fmt.Sprint(p.calls), performanceCallsColor(p.calls)},
	}
	width := 0
	for _, span := range spans {
		width += a.font.Width(span.text)
	}
	x := max(2, (w-width)/2)
	for _, span := range spans {
		a.font.DrawOutlined(a.sprites, span.text, x, 3, span.color)
		x += a.font.Width(span.text)
	}
}

type performanceSpan struct {
	text  string
	color [4]float32
}

var (
	performanceNeutral = [4]float32{.80, .84, .80, .85}
	performanceGood    = [4]float32{.21, .91, .47, .95}
	performanceFair    = [4]float32{1, .89, .20, .95}
	performanceWarn    = [4]float32{1, .62, .13, .95}
	performanceBad     = [4]float32{1, .23, .21, .95}
)

// Budget reference: 60 FPS (16.67ms). Counts are an explicit diagnostic heuristic.
func performanceFPSColor(fps float64) [4]float32 {
	if fps <= 0 || math.IsNaN(fps) {
		return performanceNeutral
	}
	switch {
	case fps >= 60:
		return performanceGood
	case fps >= 40:
		return performanceFair
	case fps >= 30:
		return performanceWarn
	default:
		return performanceBad
	}
}
func performanceTimeColor(ms float64) [4]float32 {
	if ms <= 0 || math.IsNaN(ms) {
		return performanceNeutral
	}
	switch {
	case ms <= 1000.0/60:
		return performanceGood
	case ms <= 25:
		return performanceFair
	case ms <= 1000.0/30:
		return performanceWarn
	default:
		return performanceBad
	}
}
func performanceMemoryColor(used, total uint64) [4]float32 {
	if used == 0 || total == 0 {
		return performanceNeutral
	}
	ratio := float64(used) / float64(total)
	switch {
	case ratio <= .5:
		return performanceGood
	case ratio <= .7:
		return performanceFair
	case ratio <= .85:
		return performanceWarn
	default:
		return performanceBad
	}
}
func performanceCallsColor(calls int) [4]float32 {
	switch {
	case calls <= 0:
		return performanceNeutral
	case calls <= 2000:
		return performanceGood
	case calls <= 5000:
		return performanceFair
	case calls <= 10000:
		return performanceWarn
	default:
		return performanceBad
	}
}
