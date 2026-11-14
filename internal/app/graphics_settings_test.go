package app

import (
	"github.com/Spelunkern/Shaiya-GX/internal/platform"
	"testing"
)

func TestLegacyFullscreenPreferenceMigratesToWindow(t *testing.T) {
	if platform.WindowMode(platform.DisplayMode{Width: 1600, Height: 900, Kind: platform.DisplayWindowed}) != (platform.DisplayMode{Width: 1600, Height: 900}) || platform.WindowMode(platform.DisplayMode{Kind: platform.DisplayFullscreen}) != (platform.DisplayMode{}) {
		t.Fatal("legacy display migration")
	}
}
