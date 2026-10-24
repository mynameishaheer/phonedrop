# PhoneDrop

AirDrop-style sharing between an **iPhone** and a **Windows PC** over your home Wi-Fi.

- **No iPhone app.** Send from the share sheet, Back Tap, the Action Button, Control Center or
  Siri, using shortcuts you build once in Apple's Shortcuts app.
- **One small tray app on the PC.** Single 6 MB exe, no browser engine, no runtime, no
  background polling: it uses no CPU while idle.
- **Only your phone can send.** Phone and PC are paired once with a QR code and confirmed on
  both sides; strangers on the same Wi-Fi get a bare `401` and never a pop-up.
- **Direct.** Phone → PC over the LAN, no server or account in between. Speed is your Wi-Fi's.

| From the phone | On the PC |
|---|---|
| Photos, videos, files | Saved to Downloads (or a folder you pick), with a toast: **Open** · **Show in folder** |
| Text, clipboard | Copied to the PC's clipboard |
| Links | Toast with **Open link** (never opened automatically) |
| Latest screenshot | One Back Tap |

And the other way: right-click a file → **Send to → PhoneDrop (to phone)**, or press
**Ctrl+Alt+P** to send the clipboard (text, an image or copied files). The phone collects it
with **Receive from PC**: photos go to Photos, files to Files, text to the clipboard.

## What it isn't

It's not real AirDrop. AirDrop runs over Apple's private Wi-Fi link (AWDL), which Windows Wi-Fi
drivers can't speak, so PhoneDrop uses your normal network instead. Both devices have to be on
the same Wi-Fi, and the PC must be awake. There are no push notifications to the phone (that
would need an App Store app), so you collect PC → phone items by running a shortcut.

## Install

**PC (Windows 10 or 11)**

1. Download `phonedrop.exe` and `install.ps1` from the [latest release](../../releases/latest)
   into one folder.
2. Right-click the folder background → **Open in Terminal**, then run
   `powershell -ExecutionPolicy Bypass -File .\install.ps1`.
   From an *admin* terminal it also adds the firewall rule; otherwise Windows asks once:
   allow **Private networks** only.
3. Windows SmartScreen may warn about an unsigned app: **More info → Run anyway**.
4. Click the tray icon (a phone with an arrow; it may be under **^**). It should say
   **Ready to receive**. If it says **Paused on this network**, Windows calls your Wi-Fi *Public*
   (the default for new networks): click **Use here**. Windows asks once for permission to let
   PhoneDrop through its firewall.

**iPhone**

Build the shortcuts with the step-by-step guide: **[docs/SHORTCUTS.md](docs/SHORTCUTS.md)**
(about 20 minutes, once). Then on the PC click **Pair**, run **PhoneDrop Setup** on the iPhone,
scan the code, and click **Allow** on the PC.

Uninstall: `.\install.ps1 -Uninstall`. Your settings and paired phones stay in
`%LOCALAPPDATA%\PhoneDrop` until you delete that folder.

## Security

| | |
|---|---|
| **Pairing** | The PC shows a QR code holding a fresh 256-bit secret, valid once for 2 minutes. The secret never crosses the network. The PC then asks you to **Allow** the phone. |
| **Every request** | Signed with the phone's secret (nested SHA-256, computable in Shortcuts), with a timestamp (±90 s) and a single-use nonce. Files are hashed while they stream in and kept only if the hash matches. |
| **Strangers** | Unsigned or bad requests all get the same `401`, show nothing on screen, and an IP that keeps failing is shut out for 5 minutes. |
| **Networks** | PhoneDrop listens only on networks Windows calls *Private* (or Domain). On a café or hotel network, nothing is listening at all. Connections from outside private address ranges are dropped. |
| **At rest** | Phone secrets are encrypted with Windows DPAPI (your user account). Revoke a phone in the panel and its secret is gone. |
| **Received files** | Sanitized names (no paths, no reserved names), never overwritten, marked as downloaded from the internet (Mark-of-the-Web). Programs and scripts are never opened from a toast, only shown in Explorer. |

**Threat model.** Without encryption, someone already on your Wi-Fi who can see the traffic
could read what you send (as with any unencrypted LAN traffic); they still can't send to your PC
or impersonate your phone. On the phone, the pairing file lives in iCloud Drive
(`Shortcuts/PhoneDrop/config.json`), since Shortcuts has no keychain, so it syncs to your other
Apple devices and iCloud.com. Anyone who gets it can, while on the same network as the PC, send
files to it and collect whatever is waiting in its outbox, until you remove the phone in the panel.

**Encrypt transfers (beta).** Settings → *Encrypt transfers* makes the PC a tiny certificate
authority that may only vouch for itself (X.509 name constraints: its own `.local` name and
private IPs). Your iPhone installs it as a profile, and from then on that phone talks HTTPS only.
Name constraints are what stop a stolen CA key from being used against other websites, and
**iOS's enforcement of them on user-installed roots hasn't been verified on a device yet**, so
it's opt-in. See [PROTOCOL.md §7](docs/PROTOCOL.md#7-optional-tls-beta).

## Resource use

Idle: 0% CPU (blocking waits only, no timers while the panel is hidden) and a small, flat
memory footprint (target: under 15 MB private working set).
Transfers stream to disk, so memory stays flat whatever the file size. Tested phone → PC:
216 MB in 19 s (11 MB/s) over ordinary home Wi-Fi.

## Build

Go 1.24+ on any OS, standard library only:

```sh
./build.sh                       # runs the tests, then cross-compiles phonedrop.exe
phonedrop.exe -preview <folder>  # renders every panel state to PNGs, no windows
```

Code map: [docs/DEV.md](docs/DEV.md). Wire protocol: [docs/PROTOCOL.md](docs/PROTOCOL.md).

## Licence

MIT. Icons: [Fluent UI System Icons](https://github.com/microsoft/fluentui-system-icons) (MIT).
