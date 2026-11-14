package app

import (
	"fmt"
	"github.com/Spelunkern/Shaiya-GX/internal/data/sdata"
	"github.com/Spelunkern/Shaiya-GX/internal/itemtype"
)

// Definition-only client information. Actual attempt quotes remain authoritative
// and may include blessing and a selected linking hammer.
func itemForgeInfo(d *sdata.Item) string {
	if d == nil {
		return ""
	}
	if d.Type == 30 {
		return fmt.Sprintf("Base link chance: %d%%", min(100, max(1, 289-8*int(d.ReqIg))))
	}
	if itemtype.Real(d.Type) != 26 {
		return ""
	}
	value := max(0, int(int16(d.ReqVG)))
	switch d.Special {
	case 36:
		return fmt.Sprintf("Linking bonus: +%d%%", value)
	case 58:
		return fmt.Sprintf("Extraction chance: %d%%", min(100, value))
	}
	return ""
}
