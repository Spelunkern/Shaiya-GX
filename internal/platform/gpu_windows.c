//go:build windows

// NVIDIA Optimus / AMD PowerXpress requests. OS/user preferences take priority.
#include <stdint.h>
__declspec(dllexport) uint32_t NvOptimusEnablement = 1;
__declspec(dllexport) uint32_t AmdPowerXpressRequestHighPerformance = 1;
