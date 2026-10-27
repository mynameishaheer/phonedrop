# Changelog

All notable changes to PhoneDrop are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project aims to follow
[Semantic Versioning](https://semver.org/) from 1.0.

## [Unreleased]

### Added
- Community files: contributing guide, code of conduct, security policy, issue forms and a pull
  request template.
- `docs/ROADMAP.md` and a README with a comparison table, FAQ and roadmap.
- `.gitattributes` so line endings and binary files behave on every platform.

### Changed
- The Go module path is now `github.com/mynameishaheer/phonedrop`.
- Shorter, more visual docs: README with screenshots, plus new `docs/FAQ.md` and `docs/SECURITY-MODEL.md`.

## [0.9.2] - 2026-10-24

First public version. Everything below exists today; the protocol is v1.

### Added
- **Windows tray app**: a single small exe (Go, standard library only, no cgo, no browser
  engine) with a native panel for status, pairing, recent transfers and settings. Idle costs no
  CPU: it only blocks on network and window events.
- **Phone → PC**: photos, videos and files are streamed to disk (Downloads or a folder you
  choose, never overwritten, sanitized names, Mark-of-the-Web). Text and clipboard go to the PC
  clipboard. Links show a toast with **Open link** and are never opened automatically. Toasts
  offer **Open** and **Show in folder**; programs and scripts are only ever shown in Explorer.
- **PC → phone**: Explorer **Send to → PhoneDrop (to phone)**, the panel's **Files…** and
  **Clipboard** buttons, and **Ctrl+Alt+P** to send the clipboard (text, an image or copied
  files). Items wait in an outbox for 24 hours until the phone collects them.
- **Pairing** by QR code: a fresh 256-bit secret, valid once for 2 minutes and never sent over
  the network, then an explicit **Allow** on the PC. Paired phones can be revoked in the panel.
- **Signed requests**: nested SHA-256 over method, path, timestamp (±90 s), single-use nonce and
  body hash, all computable with stock Shortcuts actions. Uploads are hashed while they stream
  and kept only if the hash matches.
- **Network safety**: listens only on networks Windows calls Private or Domain, drops
  connections from outside private address ranges, answers every bad request with the same bare
  `401`, and locks out an IP that keeps failing for 5 minutes. **Use here** lets you trust a
  Public network you choose.
- **Secrets at rest** protected with Windows DPAPI.
- **Optional encrypted transfers (beta)**: the PC acts as a name-constrained certificate
  authority delivered to the phone as a configuration profile, pinned by the QR code. iOS's
  enforcement of name constraints on user-installed roots is not yet verified on a device.
- **iPhone shortcuts**: six shortcuts (Sign, Setup, Send to PC, Send Text to PC, Send Latest
  Screenshot, Receive from PC), documented step by step in `docs/SHORTCUTS.md`, plus a generator
  (`shortcuts/build.py`) and signing scripts for importable files. Importable shortcuts are a
  work in progress.
- `install.ps1` installer (per-user, no admin needed; start at sign-in, Start-menu entry, Send
  to entry, optional firewall rule) with `-Uninstall`.
- Troubleshooting log at `%LOCALAPPDATA%\PhoneDrop\phonedrop.log` that records why a request was
  refused, never secrets or contents.
- `phonedrop.exe -preview <folder>` renders every panel state to PNGs without opening windows.
- Protocol specification (`docs/PROTOCOL.md`), developer notes (`docs/DEV.md`), and CI that
  checks formatting, vets for Windows, runs the tests with the race detector and builds the exe.

[Unreleased]: https://github.com/mynameishaheer/phonedrop/compare/v0.9.2...HEAD
[0.9.2]: https://github.com/mynameishaheer/phonedrop/releases/tag/v0.9.2
