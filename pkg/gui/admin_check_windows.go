//go:build windows
// +build windows

package gui

import (
	"syscall"
	"unsafe"
)

var (
	kernel32        = syscall.NewLazyDLL("kernel32.dll")
	advapi32        = syscall.NewLazyDLL("advapi32.dll")
	procOpenToken   = advapi32.NewProc("OpenProcessToken")
	procGetTokenInformation = advapi32.NewProc("GetTokenInformation")
	procGetCurrentProcess = kernel32.NewProc("GetCurrentProcess")
)

const (
	TOKEN_QUERY = 0x0008
	TokenElevation = 20
)

type TOKEN_ELEVATION struct {
	TokenIsElevated uint32
}

// IsRunningAsAdmin checks if the current process is running with administrator privileges
func IsRunningAsAdmin() bool {
	var token syscall.Token
	var elevated TOKEN_ELEVATION
	var retLen uint32

	// Get current process handle
	proc, _, _ := procGetCurrentProcess.Call()
	if proc == 0 {
		return false
	}

	// Open the process token
	ret, _, _ := procOpenToken.Call(
		uintptr(proc),
		TOKEN_QUERY,
		uintptr(unsafe.Pointer(&token)),
	)
	if ret == 0 {
		return false
	}
	defer token.Close()

	// Get token information
	ret, _, _ = procGetTokenInformation.Call(
		uintptr(token),
		TokenElevation,
		uintptr(unsafe.Pointer(&elevated)),
		unsafe.Sizeof(elevated),
		uintptr(unsafe.Pointer(&retLen)),
	)
	if ret == 0 {
		return false
	}

	return elevated.TokenIsElevated != 0
}

// GetAdminRequiredFeatures returns a list of features that require admin privileges
func GetAdminRequiredFeatures() []string {
	return []string{
		"Direct SPD memory reading for detailed memory information",
		"Low-level hardware access for accurate sensor readings",
		"Advanced storage detection with full hardware details",
		"System performance counters",
		"Driver-level hardware monitoring",
	}
}