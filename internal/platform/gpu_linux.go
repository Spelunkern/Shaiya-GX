//go:build linux

package platform

import (
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type linuxGPU struct {
	pci, driver string
	dedicated   bool
	vram        uint64
}

// Driver overrides are intentional, including empty values. Never replace a
// user's PRIME selection or software-rendering/debug configuration.
var gpuOverrideKeys = []string{"DRI_PRIME", "__NV_PRIME_RENDER_OFFLOAD", "__NV_PRIME_RENDER_OFFLOAD_PROVIDER", "__GLX_VENDOR_LIBRARY_NAME", "LIBGL_ALWAYS_SOFTWARE", "MESA_LOADER_DRIVER_OVERRIDE", "GALLIUM_DRIVER", "LIBGL_ALWAYS_INDIRECT", "__EGL_VENDOR_LIBRARY_FILENAMES", "__EGL_VENDOR_LIBRARY_DIRS"}

func gpuPreference(gpus []linuxGPU, lookup func(string) (string, bool)) map[string]string {
	for _, key := range gpuOverrideKeys {
		if _, ok := lookup(key); ok {
			return nil
		}
	}
	if len(gpus) < 2 {
		return nil // one usable device: no offload is needed
	}
	candidates := append([]linuxGPU(nil), gpus...)
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].dedicated != candidates[j].dedicated {
			return candidates[i].dedicated
		}
		return candidates[i].vram > candidates[j].vram
	})
	best := candidates[0]
	if !best.dedicated {
		return nil // unknown hardware: leave the driver's default alone
	}
	vars := map[string]string{"DRI_PRIME": "pci-" + strings.NewReplacer(":", "_", ".", "_").Replace(best.pci)}
	if best.driver == "nvidia" {
		vars["__NV_PRIME_RENDER_OFFLOAD"] = "1"
		vars["__GLX_VENDOR_LIBRARY_NAME"] = "nvidia"
	}
	return vars
}

func detectLinuxGPUs(root string) []linuxGPU {
	nodes, _ := filepath.Glob(filepath.Join(root, "renderD*"))
	var gpus []linuxGPU
	for _, node := range nodes {
		renderNode := filepath.Join("/dev/dri", filepath.Base(node))
		fd, err := os.Open(renderNode)
		if err != nil {
			continue
		} // inaccessible devices cannot render for this user
		fd.Close()
		device, err := filepath.EvalSymlinks(filepath.Join(node, "device"))
		if err != nil {
			continue
		}
		pci := filepath.Base(device)
		// PRIME's PCI syntax applies only to PCI devices, not virtual/platform DRM.
		if len(pci) != 12 || pci[4] != ':' || pci[7] != ':' || pci[10] != '.' {
			continue
		}
		driverPath, err := filepath.EvalSymlinks(filepath.Join(device, "driver"))
		if err != nil {
			continue
		}
		gpu := linuxGPU{pci: pci, driver: filepath.Base(driverPath)}
		switch gpu.driver {
		case "nvidia", "nouveau":
			gpu.dedicated = true
		case "amdgpu":
			// The kernel's FUSION bit distinguishes APUs from discrete AMD cards;
			// shared UMA memory size is not a reliable indicator.
			gpu.dedicated = amdDiscrete(renderNode)
		}
		if memory, err := os.ReadFile(filepath.Join(device, "mem_info_vram_total")); err == nil {
			gpu.vram, _ = strconv.ParseUint(strings.TrimSpace(string(memory)), 10, 64)
		}
		gpus = append(gpus, gpu)
	}
	return gpus
}

var gpuPreferenceOnce sync.Once

func prepareGPU() {
	gpuPreferenceOnce.Do(func() {
		vars := gpuPreference(detectLinuxGPUs("/sys/class/drm"), os.LookupEnv)
		for key, value := range vars {
			if err := os.Setenv(key, value); err != nil {
				log.Printf("GPU preference %s: %v", key, err)
			}
		}
		if len(vars) > 0 {
			log.Printf("GPU preference: dedicated adapter (%s); actual renderer follows in OpenGL log", vars["DRI_PRIME"])
		}
	})
}
