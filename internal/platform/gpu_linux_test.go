//go:build linux

package platform

import (
	"reflect"
	"testing"
)

func TestGPUPreferenceSelectsDedicatedInsteadOfFirst(t *testing.T) {
	noOverride := func(string) (string, bool) { return "", false }
	intel := linuxGPU{pci: "0000:00:02.0", driver: "i915"}
	nvidia := linuxGPU{pci: "0000:01:00.0", driver: "nvidia", dedicated: true}
	amd := linuxGPU{pci: "0000:03:00.0", driver: "amdgpu", dedicated: true, vram: 8 << 30}
	for _, devices := range [][]linuxGPU{{intel, nvidia}, {nvidia, intel}} {
		got := gpuPreference(devices, noOverride)
		if got["DRI_PRIME"] != "pci-0000_01_00_0" || got["__NV_PRIME_RENDER_OFFLOAD"] != "1" || got["__GLX_VENDOR_LIBRARY_NAME"] != "nvidia" {
			t.Fatal("did not prefer dedicated NVIDIA", got)
		}
	}
	if got := gpuPreference([]linuxGPU{intel, amd}, noOverride); !reflect.DeepEqual(got, map[string]string{"DRI_PRIME": "pci-0000_03_00_0"}) {
		t.Fatal("Mesa device not selected by PCI address", got)
	}
	apu := linuxGPU{pci: "0000:04:00.0", driver: "amdgpu", vram: 16 << 30}
	if got := gpuPreference([]linuxGPU{apu, amd}, noOverride); got["DRI_PRIME"] != "pci-0000_03_00_0" {
		t.Fatal("shared APU memory outranked discrete memory", got)
	}
	for _, devices := range [][]linuxGPU{nil, {nvidia}, {intel, apu}} {
		if got := gpuPreference(devices, noOverride); got != nil {
			t.Fatal("changed default unnecessarily", got)
		}
	}
}

func TestGPUPreferenceRespectsDriverOverrides(t *testing.T) {
	devices := []linuxGPU{{pci: "0000:00:02.0"}, {pci: "0000:01:00.0", driver: "nvidia", dedicated: true}}
	for _, key := range gpuOverrideKeys {
		for _, value := range []string{"", "0", "1"} {
			lookup := func(k string) (string, bool) { return value, k == key }
			if got := gpuPreference(devices, lookup); got != nil {
				t.Fatalf("overrode %s=%q: %v", key, value, got)
			}
		}
	}
	if amdDiscrete("/no/such/render/node") {
		t.Fatal("unavailable AMD device classified as dedicated")
	}
}
