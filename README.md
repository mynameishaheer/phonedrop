# PhoneDrop

AirDrop-style sharing between an **iPhone** and a **Windows PC**, over your home Wi-Fi.
No iPhone app: you send with Shortcuts from the share sheet, Back Tap or the Action Button.
On the PC, PhoneDrop is a single small tray app with no browser engine and no runtime.

> **Status: early development (milestone 1 of 6).** The tray app, settings and network
> detection work. Pairing and transfers come next. It can't receive anything yet.

## How it stays private

- **Paired phones only.** Each phone is paired once by scanning a QR code on the PC and
  confirming on both sides. Anything else gets a plain `401`, never a pop-up.
- **Home networks only.** PhoneDrop listens only on networks Windows calls *Private*. On a
  *Public* network (café, hotel, airport) nothing is listening at all.
- **LAN peers only.** Connections from outside private address ranges are dropped, and IPs
  that keep failing are shut out for a few minutes.

## Install (Windows 10/11)

1. Download `phonedrop.exe` and `install.ps1` into the same folder.
2. In PowerShell, in that folder: `powershell -ExecutionPolicy Bypass -File .\install.ps1`.
   Run it from an admin PowerShell to add the firewall rule automatically.
3. Click the tray icon. If it says *Paused on this network*, your home Wi-Fi is set to
   Public: click the card and set the network profile to **Private**.

Uninstall: `.\install.ps1 -Uninstall`.

## Build

Go 1.24+ on any OS: `./build.sh` runs the tests and cross-compiles `phonedrop.exe`.
`phonedrop.exe -preview <folder>` renders the panel to PNGs without opening windows.

## Roadmap

1. ~~Skeleton: tray, settings, Private-network check, listener~~
2. Pairing (QR code) and the Setup shortcut
3. Phone → PC: files, photos, text and links
4. PC → phone: an outbox the phone pulls from
5. TLS (if iOS honours name-constrained certificates)
6. First release

MIT licensed. Icons: [Fluent UI System Icons](https://github.com/microsoft/fluentui-system-icons) (MIT).
