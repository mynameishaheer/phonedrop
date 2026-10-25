## What and why

<!-- One or two sentences. Link the issue: "Fixes #123". -->

## Checklist

- [ ] `go test -race ./...` passes (new behaviour has a test)
- [ ] `GOOS=windows go vet .` passes
- [ ] `gofmt -l .` prints nothing
- [ ] Docs updated (`README.md`, `docs/PROTOCOL.md`, `docs/SHORTCUTS.md` and `shortcuts/build.py` if the protocol or shortcuts changed)
- [ ] `CHANGELOG.md` has a line under `[Unreleased]` for anything users will notice
- [ ] Still standard library only, no cgo, and nothing runs while idle
- [ ] UI change: before/after PNGs from `phonedrop.exe -preview` attached
