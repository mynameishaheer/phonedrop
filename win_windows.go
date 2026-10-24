// Win32 plumbing shared by the tray UI and the network watcher: DLLs, window-class and message
// structs, and registry helpers. Stdlib only (raw syscall), so there's no cgo and no runtime to ship.
package main

import (
	"sync"
	"syscall"
	"unsafe"
)

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	user32   = syscall.NewLazyDLL("user32.dll")
	advapi32 = syscall.NewLazyDLL("advapi32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	shcore   = syscall.NewLazyDLL("shcore.dll")
	ole32    = syscall.NewLazyDLL("ole32.dll")
	oleaut32 = syscall.NewLazyDLL("oleaut32.dll")
	iphlpapi = syscall.NewLazyDLL("iphlpapi.dll")

	defWindow    = user32.NewProc("DefWindowProcW")
	registerCls  = user32.NewProc("RegisterClassExW")
	createWindow = user32.NewProc("CreateWindowExW")
	getMessage   = user32.NewProc("GetMessageW")
	dispatchMsg  = user32.NewProc("DispatchMessageW")
	postMsg      = user32.NewProc("PostMessageW")
	getModule    = kernel32.NewProc("GetModuleHandleW")
	regGetValu   = advapi32.NewProc("RegGetValueW")

	winProcs = map[string]*syscall.LazyProc{}
	winMu    sync.Mutex
)

// win calls a Windows API function by name. Pointer arguments must be converted inline at the
// call (uintptr(unsafe.Pointer(&x))) so the compiler keeps them alive for the call.
//
//go:uintptrescapes
func win(dll *syscall.LazyDLL, name string, args ...uintptr) uintptr {
	key := dll.Name + name
	winMu.Lock()
	p, ok := winProcs[key]
	if !ok {
		p = dll.NewProc(name)
		winProcs[key] = p
	}
	winMu.Unlock()
	r, _, _ := p.Call(args...)
	return r
}

type wndClassEx struct {
	size, style                        uint32
	wndProc                            uintptr
	clsExtra, wndExtra                 int32
	instance, icon, cursor, background uintptr
	menuName, className                *uint16
	iconSm                             uintptr
}

type msg struct {
	hwnd           uintptr
	message        uint32
	wParam, lParam uintptr
	time           uint32
	pt             struct{ x, y int32 }
	private        uint32
}

type rect struct{ left, top, right, bottom int32 }

const (
	hkcu = 0x80000001
	hklm = 0x80000002
)

func regDword(key, value string) (uint32, bool) {
	var data, size uint32 = 0, 4
	r, _, _ := regGetValu.Call(hkcu, uintptr(unsafe.Pointer(wstr(key))), uintptr(unsafe.Pointer(wstr(value))), 0x10, /*RRF_RT_REG_DWORD*/
		0, uintptr(unsafe.Pointer(&data)), uintptr(unsafe.Pointer(&size)))
	return data, r == 0
}

func regBinary(key, value string) []byte {
	buf := make([]byte, 64)
	size := uint32(len(buf))
	r, _, _ := regGetValu.Call(hkcu, uintptr(unsafe.Pointer(wstr(key))), uintptr(unsafe.Pointer(wstr(value))), 0x08, /*RRF_RT_REG_BINARY*/
		0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	if r != 0 {
		return nil
	}
	return buf[:size]
}

func regString(root uintptr, key, value string) (string, bool) {
	buf := make([]uint16, 1024)
	size := uint32(len(buf) * 2)
	r, _, _ := regGetValu.Call(root, uintptr(unsafe.Pointer(wstr(key))), uintptr(unsafe.Pointer(wstr(value))), 0x02, /*RRF_RT_REG_SZ*/
		0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	if r != 0 {
		return "", false
	}
	return syscall.UTF16ToString(buf), true
}

const runKey = `Software\Microsoft\Windows\CurrentVersion\Run`

// setRunAtLogin adds or removes the HKCU Run entry that starts PhoneDrop at sign-in (no admin).
func setRunAtLogin(on bool, exe string) bool {
	var key uintptr
	if r, _, _ := advapi32.NewProc("RegCreateKeyExW").Call(hkcu, uintptr(unsafe.Pointer(wstr(runKey))), 0, 0, 0,
		0x20006 /*KEY_WRITE*/, 0, uintptr(unsafe.Pointer(&key)), 0); r != 0 {
		return false
	}
	defer advapi32.NewProc("RegCloseKey").Call(key)
	if !on {
		r, _, _ := advapi32.NewProc("RegDeleteValueW").Call(key, uintptr(unsafe.Pointer(wstr(appName))))
		return r == 0 || r == 2 // ERROR_FILE_NOT_FOUND: already off
	}
	v, _ := syscall.UTF16FromString(`"` + exe + `"`)
	r, _, _ := advapi32.NewProc("RegSetValueExW").Call(key, uintptr(unsafe.Pointer(wstr(appName))), 0, 1, /*REG_SZ*/
		uintptr(unsafe.Pointer(&v[0])), uintptr(len(v)*2))
	return r == 0
}

func runAtLogin() bool {
	_, ok := regString(hkcu, runKey, appName)
	return ok
}

func shellOpen(target string) {
	win(shell32, "ShellExecuteW", 0, uintptr(unsafe.Pointer(wstr("open"))), uintptr(unsafe.Pointer(wstr(target))), 0, 0, 1)
}
