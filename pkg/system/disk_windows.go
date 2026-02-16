//go:build windows

package system

import (
	"log/slog"
	"syscall"
	"unsafe"
)

// getDiskUsage gets disk usage for the given path on Windows
func getDiskUsage(path string) (*DiskUsage, error) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getDiskFreeSpaceEx := kernel32.NewProc("GetDiskFreeSpaceExW")

	var freeBytesAvailable, totalBytes, totalFreeBytes uint64

	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		slog.Error("Failed to convert path", "path", path, "error", err)
		return nil, err
	}

	ret, _, err := getDiskFreeSpaceEx.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFreeBytes)),
	)
	if ret == 0 {
		slog.Error("Failed to get disk stats", "path", path, "error", err)
		return nil, err
	}

	return &DiskUsage{
		Total: totalBytes,
		Free:  freeBytesAvailable,
	}, nil
}
