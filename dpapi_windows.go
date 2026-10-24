package main

import (
	"errors"
	"runtime"
	"syscall"
	"unsafe"
)

// DPAPI with user scope: only this Windows user, on this machine, can unprotect. There is no
// fallback; if DPAPI fails, nothing is stored.

var (
	crypt32         = syscall.NewLazyDLL("crypt32.dll")
	cryptProtect    = crypt32.NewProc("CryptProtectData")
	cryptUnprotect  = crypt32.NewProc("CryptUnprotectData")
	localFree       = kernel32.NewProc("LocalFree")
	dpapiEntropy    = []byte("PhoneDrop v1") // binds blobs to this app, not just this user
	dpapiDescriptor = wstr(appName)
)

const cryptprotectUIForbidden = 0x1

type dataBlob struct {
	cbData uint32
	pbData *byte
}

func blobOf(b []byte) dataBlob {
	if len(b) == 0 {
		return dataBlob{}
	}
	return dataBlob{cbData: uint32(len(b)), pbData: &b[0]}
}

func dpapiProtect(plain []byte) ([]byte, error) {
	return dpapiCall(cryptProtect, plain, true)
}

func dpapiUnprotect(blob []byte) ([]byte, error) {
	return dpapiCall(cryptUnprotect, blob, false)
}

func dpapiCall(proc *syscall.LazyProc, data []byte, protect bool) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("dpapi: empty input")
	}
	in, ent := blobOf(data), blobOf(dpapiEntropy)
	var out dataBlob
	desc := uintptr(0)
	if protect {
		desc = uintptr(unsafe.Pointer(dpapiDescriptor))
	}
	// Both functions take (in, description, entropy, reserved, prompt, flags, out); for
	// CryptUnprotectData the description is an out-parameter we don't want (NULL).
	r, _, err := proc.Call(ptr(&in), desc, ptr(&ent), 0, 0, cryptprotectUIForbidden, ptr(&out))
	runtime.KeepAlive(data)
	if r == 0 {
		return nil, errors.New(proc.Name + ": " + err.Error())
	}
	defer localFree.Call(uintptr(unsafe.Pointer(out.pbData)))
	res := make([]byte, out.cbData)
	copy(res, unsafe.Slice(out.pbData, out.cbData))
	if !protect {
		clear(unsafe.Slice(out.pbData, out.cbData)) // don't leave the plaintext in freed memory
	}
	return res, nil
}
