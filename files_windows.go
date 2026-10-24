package main

import "os"

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
