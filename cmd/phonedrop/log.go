package main

import (
	"log"
	"os"
	"path/filepath"
)

// openLog sends the log to %LOCALAPPDATA%\PhoneDrop\phonedrop.log, started fresh each run so it
// never grows. It records network changes and listener results (never secrets or file names),
// for troubleshooting.
func openLog() {
	os.MkdirAll(dataDir(), 0o700)
	f, err := os.Create(filepath.Join(dataDir(), "phonedrop.log"))
	if err != nil {
		return
	}
	log.SetOutput(f)
	log.SetFlags(log.LstdFlags)
}
