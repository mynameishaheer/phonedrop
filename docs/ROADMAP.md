# Roadmap

Where PhoneDrop is going. Nothing here is a promise or a date; it's the order we'd like to do
things in. Want something moved up, or to take a piece? Open a
[Discussion](https://github.com/mynameishaheer/phonedrop/discussions) or an issue. Current state:
[CHANGELOG.md](../CHANGELOG.md). Ground rules for every item: standard library only, no cgo, idle
costs nothing ([CONTRIBUTING.md](../CONTRIBUTING.md)).

## Planned

1. **v1.0 release.** A tagged, CI-built `phonedrop.exe` with release notes, the protocol marked
   stable, and the README's idle-memory figure measured instead of targeted.
2. **iCloud shortcut links.** One-tap install of the six shortcuts for everyone. Needs the
   generated shortcuts signed on a Mac and shared from the Shortcuts app; then the README's
   "iCloud links coming with v1.0" becomes real links.
3. **Optional push notifications via ntfy.** Shortcuts can't wait in the background, so today the
   phone has to run **Receive from PC**. With an opt-in [ntfy](https://ntfy.sh) topic the PC
   could ping the phone when something is waiting. Off by default; no polling.
4. **Relay for off-LAN sending.** Send when the phone and PC aren't on the same network. The
   relay would see only end-to-end encrypted blobs. Biggest item here; needs a design discussion
   first.
5. **Encryption out of beta.** Verify on real iPhones (several iOS versions) that name
   constraints on user-installed roots are enforced, then make the encrypted mode the default or
   drop the beta label. Testers welcome; see [SECURITY.md](../SECURITY.md).
6. **Code signing** for `phonedrop.exe`, so SmartScreen stops warning.

## Good first issues

Small, well-scoped, and each has a ready-to-paste issue in
[launch/good-first-issues.md](launch/good-first-issues.md). Look for the
[good first issue](https://github.com/mynameishaheer/phonedrop/labels/good%20first%20issue) label.

- **Rename a paired phone in the panel.** Phones are named by the phone at pairing and can only
  be removed. Add a rename action to the paired-phones list, persisted in `phones.json`.
- **Drag files onto the panel to send them.** Accept dropped files and queue them in the outbox,
  like **Send to → PhoneDrop** does.
- **Measure and document idle memory.** The README targets under 15 MB private working set.
  Measure on Windows 10 and 11 and write down the real numbers and how to reproduce them.
- **Get File folder fix in the shortcuts generator.** Generated shortcuts ship with the *Get
  File* folder unresolved, so users must re-pick *Shortcuts* in iCloud Drive by hand. Find a
  serialization iOS accepts and remove that manual step.
- **More file-type sniffing.** `sniffExt` guesses extensions for uploads that arrive without
  one. Add more signatures (for example HEIC variants, WebP, MKV, Office files) with tests.
- **Localise UI strings.** Move the panel and toast text into one table so translations can be
  added; start with a second language as a proof.
- **Uninstall entry in Settings > Apps.** `install.ps1` should register under
  `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall` so PhoneDrop can be removed from
  Windows Settings, not only with `-Uninstall`.
- **Show the version in the panel.** There's currently no way to see the version without opening
  the exe's file properties; show it in a small About line.
