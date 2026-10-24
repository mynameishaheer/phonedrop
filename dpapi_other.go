//go:build !windows

package main

// Non-Windows builds exist only for tests; PhoneDrop never stores secrets there. Identity
// functions keep the portable code testable.

func dpapiProtect(plain []byte) ([]byte, error) { return append([]byte(nil), plain...), nil }

func dpapiUnprotect(blob []byte) ([]byte, error) { return append([]byte(nil), blob...), nil }
