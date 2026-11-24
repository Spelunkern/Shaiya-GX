//go:build windows

package platform

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

type processMemoryCounters struct {
	cb                         uint32
	pageFaultCount             uint32
	peakWorkingSetSize         uintptr
	workingSetSize             uintptr
	quotaPeakPagedPoolUsage    uintptr
	quotaPagedPoolUsage        uintptr
	quotaPeakNonPagedPoolUsage uintptr
	quotaNonPagedPoolUsage     uintptr
	pagefileUsage              uintptr
	peakPagefileUsage          uintptr
}

type memoryStatusEx struct {
	length               uint32
	memoryLoad           uint32
	totalPhys            uint64
	availPhys            uint64
	totalPageFile        uint64
	availPageFile        uint64
	totalVirtual         uint64
	availVirtual         uint64
	availExtendedVirtual uint64
}

var (
	kernel32                 = windows.NewLazySystemDLL("kernel32.dll")
	procGetProcessMemoryInfo = kernel32.NewProc("K32GetProcessMemoryInfo")
	procGlobalMemoryStatusEx = kernel32.NewProc("GlobalMemoryStatusEx")
)

// ResidentBytes reports the current working-set size of the client process.
func ResidentBytes() (uint64, error) {
	var counters processMemoryCounters
	counters.cb = uint32(unsafe.Sizeof(counters))
	r1, _, err := procGetProcessMemoryInfo.Call(
		uintptr(windows.CurrentProcess()),
		uintptr(unsafe.Pointer(&counters)),
		uintptr(counters.cb),
	)
	if r1 == 0 {
		return 0, err
	}
	return uint64(counters.workingSetSize), nil
}

// TotalMemoryBytes reports installed physical RAM in bytes.
func TotalMemoryBytes() (uint64, error) {
	var status memoryStatusEx
	status.length = uint32(unsafe.Sizeof(status))
	r1, _, err := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&status)))
	if r1 == 0 {
		return 0, err
	}
	return status.totalPhys, nil
}
