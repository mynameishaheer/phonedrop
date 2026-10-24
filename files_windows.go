package main

import (
	"os"
	"syscall"
)

// markOfTheWeb tags a received file as downloaded (ZoneId=3) so SmartScreen and Office treat it
// with suspicion. It fails on file systems without alternate streams (FAT/exFAT); callers ignore that.
func markOfTheWeb(path string) error {
	f, err := os.OpenFile(path+":Zone.Identifier", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	_, err = f.WriteString("[ZoneTransfer]\r\nZoneId=3\r\n")
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}

const fileAttributeHidden = 0x2

// hideFile sets FILE_ATTRIBUTE_HIDDEN on an upload's temporary file, so a half-received file
// doesn't show in Explorer (the leading dot only hides it elsewhere). Best effort.
func hideFile(path string) { setHidden(path, true) }

// unhideFile clears it again: a rename keeps the attribute, so the finished file needs this.
func unhideFile(path string) { setHidden(path, false) }

func setHidden(path string, on bool) {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return
	}
	attrs, err := syscall.GetFileAttributes(p)
	if err != nil {
		return
	}
	if on {
		attrs |= fileAttributeHidden
	} else {
		attrs &^= fileAttributeHidden
	}
	syscall.SetFileAttributes(p, attrs)
}
