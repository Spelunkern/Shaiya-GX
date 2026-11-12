package platform

import (
	"os"
	"strconv"
	"strings"
)

// ResidentBytes is the process RSS, including Go, native audio and GL mappings.
func ResidentBytes() (uint64, error) {
	data, err := os.ReadFile("/proc/self/statm")
	if err != nil {
		return 0, err
	}
	fields := strings.Fields(string(data))
	if len(fields) < 2 {
		return 0, strconv.ErrSyntax
	}
	pages, err := strconv.ParseUint(fields[1], 10, 64)
	return pages * uint64(os.Getpagesize()), err
}

func TotalMemoryBytes() (uint64, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "MemTotal:" {
			n, err := strconv.ParseUint(fields[1], 10, 64)
			return n * 1024, err
		}
	}
	return 0, strconv.ErrSyntax
}
