//go:build !linux

package platform

import "errors"

func ResidentBytes() (uint64, error) {
	return 0, errors.New("process RSS unavailable on this platform")
}

func TotalMemoryBytes() (uint64, error) {
	return 0, errors.New("physical RAM capacity unavailable on this platform")
}
