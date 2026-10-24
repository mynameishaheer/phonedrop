#!/bin/sh
# Cross-compile PhoneDrop for Windows (from any OS with Go 1.24+). Writes phonedrop.exe.
# winres/ holds the exe's icon, manifest and version info; after changing it, regenerate the
# resource file with: go run github.com/tc-hib/go-winres@v0.3.3 make --arch amd64 --in winres/winres.json
set -e
cd "$(dirname "$0")"
go test ./...
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags "-s -w -H windowsgui" -o phonedrop.exe .
echo "built phonedrop.exe"
