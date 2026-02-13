//go:build windows

package system

import (
	"syscall"
	"unsafe"

	"github.com/tachRoutine/beamdrop-go/pkg/logger"
)

// getDiskUsage gets disk usage for the given path on Windows
func getDiskUsage(path string) (*DiskUsage, error) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getDiskFreeSpaceEx := kernel32.NewProc("GetDiskFreeSpaceExW")

	var freeBytesAvailable, totalBytes, totalFreeBytes uint64

	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		logger.Error("Failed to convert path %s: %v", path, err)
		return nil, err
	}

	ret, _, err := getDiskFreeSpaceEx.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFreeBytes)),
	)
	if ret == 0 {
		logger.Error("Failed to get disk stats for path %s: %v", path, err)
		return nil, err
	}

	return &DiskUsage{
		Total: totalBytes,
		Free:  freeBytesAvailable,
	}, nil
}
