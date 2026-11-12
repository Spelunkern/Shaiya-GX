package ui

import (
	"github.com/Spelunkern/Shaiya-GX/internal/data/text"
	"strconv"
)

func MapName(messages text.SysMsg, id uint16) string {
	if message, ok := mapNameMsg[id]; ok {
		if name := messages.Get(message); name != "" {
			return name
		}
	}
	return strconv.Itoa(int(id))
}
