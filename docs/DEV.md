# PhoneDrop: developer notes

Architecture and conventions. The wire protocol, [PROTOCOL.md](PROTOCOL.md), is the contract: read it first.

## Ground rules

- **Go 1.24, standard library only.** No third-party modules, no cgo. Windows APIs are called
  through `syscall` (see `win_windows.go`, `gdip_windows.go`, `netwatch_windows.go`).
- One `package main`, in `cmd/phonedrop/`. Files ending `_windows.go` are Windows-only (UI, Win32, DPAPI, COM).
  Everything else is **portable** and must build and test on Linux: `go test ./...`.
  `main_other.go` gives non-Windows builds a stub `main`.
- Build with `scripts/build.sh` (runs tests, then a `GOOS=windows` cross-compile). Also check `GOOS=windows go vet .` passes.
- Idle must cost nothing: no polling loops, no tickers while the panel is hidden. Block on accepts and waits; check expiry lazily.
- Never buffer a whole received file in memory; stream to disk.
- Never log secrets, signatures, file names or text contents. `log.Printf` goes to
  `%LOCALAPPDATA%\PhoneDrop\phonedrop.log` (see `log.go`).
- Match the existing style: short doc comments that say *why*, compact code, `gofmt`.

## Data on the laptop

`%LOCALAPPDATA%\PhoneDrop\` (`dataDir()` in `settings.go`):

| File | Contents |
|---|---|
| `settings.json` | `settings` struct |
| `phones.json` | paired phones; secrets DPAPI-protected (base64), never plaintext |
| `history.json` | recent transfers (last 50), for the panel |
| `outbox/` | laptop → phone items that need their own copy (clipboard text/images) |
| `tls/` | `ca.crt` (PEM) and `ca.key` (DPAPI-protected PKCS#8, base64); the leaf lives in memory only |
| `phonedrop.log` | troubleshooting log, recreated each run |

## Threads

- **UI thread** (`runUI`, locked OS thread): owns all windows, GDI+ and the clipboard. Other
  goroutines never touch UI state directly: they update fields under `ui.mu` and
  `PostMessage(u.hwnd, wmRefresh, ...)`, or send on a channel the UI drains in a `WM_APP+n`
  handler.
- **Network watcher** (`watchNetworks`): calls `server.bind` with the Private-network addresses.
- **HTTP server** (`server.go`): one goroutine per connection (net/http).

## Server-side seams (portable)

The server talks to the rest of the app only through the `appHooks` interface, so it can be
tested on Linux with a fake:

```go
type appHooks interface {
	// confirmPairing asks the user to allow a new phone; it blocks until they answer or ctx ends.
	confirmPairing(ctx context.Context, phoneName string) bool
	// received is called after an item from a phone has been saved (file) or read (text/url).
	received(it receivedItem)
	// pairingChanged is called when phones are added or removed, so the UI can redraw.
	phonesChanged()
}
```

Secrets at rest go through `protect`/`unprotect func([]byte) ([]byte, error)`: DPAPI
(`CryptProtectData`, user scope) on Windows, identity in tests.

## Portable / scripted install

The normal install is **PhoneDrop-Setup.exe** (see the [README](../README.md#install)). Each release also has the plain `phonedrop.exe` and `install.ps1` (source: `scripts/install.ps1`) for a per-user, scripted install with no installer.

1. Download both files from the [latest release](../../../releases/latest) into one folder.
2. Right-click the folder background → **Open in Terminal**, then run:

   ```powershell
   powershell -ExecutionPolicy Bypass -File .\install.ps1
   ```

3. From an *admin* terminal it also adds the firewall rule. Otherwise Windows asks once: allow **Private networks** only.
4. To remove it: `.\install.ps1 -Uninstall`. Settings and paired phones stay in `%LOCALAPPDATA%\PhoneDrop` until you delete that folder.

It installs to `%LOCALAPPDATA%\PhoneDrop` (no admin needed), starts at sign-in, and adds a Start-menu entry and the **Send to** entry. Re-run it to update.
