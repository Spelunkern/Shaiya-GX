//go:build linux

package platform

/*
#include <stdint.h>
#include <stdlib.h>
#include <fcntl.h>
#include <unistd.h>
#include <sys/ioctl.h>

// Stable Linux AMDGPU_INFO_DEV_INFO UAPI prefix through ids_flags. Keeping
// only this prefix needs no libdrm dependency or particular kernel headers.
struct gx_amd_device_info {
    uint32_t identity[8];
    uint64_t clocks[2];
    uint32_t cu[2], bitmap[4][4], render[4];
    uint64_t ids_flags;
};
struct gx_amd_query {
    uint64_t return_pointer;
    uint32_t return_size, query;
    uint64_t parameters[2];
};
static int gx_amd_discrete(const char *path) {
    int fd = open(path, O_RDONLY | O_CLOEXEC);
    if (fd < 0) return 0;
    struct gx_amd_device_info info = {0};
    struct gx_amd_query query = {0};
    query.return_pointer = (uintptr_t)&info;
    query.return_size = sizeof(info);
    query.query = 0x16; // AMDGPU_INFO_DEV_INFO
    int ok = ioctl(fd, _IOW('d', 0x45, struct gx_amd_query), &query);
    close(fd);
    return ok == 0 && !(info.ids_flags & 1); // AMDGPU_IDS_FLAGS_FUSION
}
*/
import "C"

import "unsafe"

func amdDiscrete(path string) bool {
	name := C.CString(path)
	defer C.free(unsafe.Pointer(name))
	return C.gx_amd_discrete(name) != 0
}
