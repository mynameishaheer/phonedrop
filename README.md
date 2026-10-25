# PhoneDrop

**AirDrop-style sharing between your iPhone and your Windows PC: no iPhone app, one tiny tray exe, nothing running when idle.**

<sub>An independent project, not affiliated with or endorsed by Apple or Microsoft. AirDrop and iPhone are trademarks of Apple Inc.</sub>

[![build](https://github.com/mynameishaheer/phonedrop/actions/workflows/build.yml/badge.svg)](https://github.com/mynameishaheer/phonedrop/actions/workflows/build.yml)
[![License: MIT](https://img.shields.io/github/license/mynameishaheer/phonedrop)](LICENSE)
[![Latest release](https://img.shields.io/github/v/release/mynameishaheer/phonedrop?include_prereleases&sort=semver)](https://github.com/mynameishaheer/phonedrop/releases/latest)
[![Go version](https://img.shields.io/github/go-mod/go-version/mynameishaheer/phonedrop)](go.mod)

<!--
  DEMO GIF (record before launch, then uncomment the line below and delete this comment).
  Save it as docs/media/demo.gif, ideally under 3 MB and about 800 px wide. Show, in about 10
  seconds with no cuts: share a photo from the iPhone share sheet -> "Send to PC", then the toast
  on the PC with Open / Show in folder. If you can fit it, add the other direction: right-click a
  file on the PC -> Send to -> PhoneDrop, then "Receive from PC" on the phone. A screen recording
  of the phone (QuickTime or Control Center) next to a Windows screen capture, side by side,
  works well. A static screenshot of the tray panel (docs/media/panel.png) also helps.

  <p align="center"><img src="docs/media/demo.gif" alt="Sending a photo from an iPhone to a Windows PC with PhoneDrop" width="720"></p>
-->

- **No iPhone app.** Send from the share sheet, Back Tap, the Action Button, Control Center or
  Siri, using shortcuts you build once in Apple's Shortcuts app.
- **One small tray app on the PC.** Single 7 MB exe, no browser engine, no runtime, no
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

## Why PhoneDrop?

Other tools do more, or do it on more platforms. PhoneDrop's niche is narrow: **iPhone ↔ Windows,
with nothing to install on the phone and nothing running on the PC when you aren't using it.**

| | PhoneDrop | Microsoft Phone Link | LocalSend | Intel Unison | PairDrop / Snapdrop |
|---|---|---|---|---|---|
| App on the iPhone | **None** (Shortcuts) | Yes | Yes | Yes | None (web page) |
| On the PC | One tray exe | Built into Windows | App | App | A browser tab |
| Files iPhone → PC | Yes | Rolling out; often unreliable | Yes | Yes (discontinued) | Yes |
| Open source | Yes (MIT) | No | Yes | No | Yes |
| Platforms | iPhone + Windows only | Android, limited iPhone | Most | Was iPhone/Android + Windows | Anything with a browser |

Intel Unison was discontinued in 2025. Phone Link's iPhone file sharing has been rolling out
since late 2024 and is often reported as failing from iPhone to PC. LocalSend and PairDrop are excellent and far more general; pick
them if you need other platforms, or if installing an app or keeping a tab open doesn't bother
you. (Details change; check each project for its current state.)

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

Import the ready-made shortcuts (iCloud links coming with v1.0), or build them by hand with
**[docs/SHORTCUTS.md](docs/SHORTCUTS.md)** (about 20 minutes, once). Until the iCloud links
exist, importing means signing the files on a Mac; see [shortcuts/README.md](shortcuts/README.md).
Then on the PC click **Pair**, run **PhoneDrop Setup** on the iPhone, scan the code, and click
**Allow** on the PC.

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

## FAQ

**Is this AirDrop?** No. AirDrop uses Apple's private Wi-Fi link (AWDL), which Windows can't
speak. PhoneDrop sends over your normal network and gives you similar one-tap sharing.

**Does it work away from my home Wi-Fi?** It works on any network where both devices can reach
each other and Windows calls the network *Private* (home, and most offices that allow device to
device traffic). It does not work across the internet or between different networks yet; a relay
for that is on the [roadmap](docs/ROADMAP.md).

**Is it safe on public Wi-Fi?** By default PhoneDrop doesn't listen at all on networks Windows
calls *Public*, so there is nothing to attack. If you click **Use here** on one, only your paired
phone can send, but without *Encrypt transfers* (beta) other people on that network could read
what you send. Prefer your own network.

**Does it work with Android?** Not today. The shortcuts are for iOS, and the PC app only knows
the PhoneDrop protocol. The protocol is plain HTTP and fully documented in
[docs/PROTOCOL.md](docs/PROTOCOL.md), so a client for another phone is possible; contributions
welcome.

**Does it work on a Mac?** There's no macOS app, and a Mac already has AirDrop. The server is a
Windows tray app.

**Why Shortcuts instead of an app?** An App Store app means a developer account, review, and a
constantly running background service on the phone. Shortcuts already live in the share sheet,
on Back Tap and on the Action Button. The cost is that the PC can't push to the phone: you run
**Receive from PC** to collect.

**Windows warns that the app is unsigned.** Yes, it isn't code-signed yet. You can inspect the
source and build it yourself with `./build.sh`; CI builds the released exe from the tagged commit.
See [Verifying a download](#verifying-a-download).

**Was this written by AI?** Largely, yes, with a person deciding, testing on real devices and
reviewing. [docs/HOW-IT-WAS-BUILT.md](docs/HOW-IT-WAS-BUILT.md) says who did what, what was
checked, and what is still unverified.

## Troubleshooting

| Symptom | Fix |
|---|---|
| Panel says **Paused on this network** | Windows calls the Wi-Fi *Public*. Click **Use here** (or make the network Private in Windows settings). |
| Phone says it can't connect | Same Wi-Fi on both? Some guest and office networks block device-to-device traffic. Check the firewall rule: `.\install.ps1` from an admin terminal adds it. |
| *A server with the specified hostname could not be found* | `.local` names don't resolve on this network. In Files → iCloud Drive → Shortcuts → PhoneDrop, edit `config.json`: replace the host in `base` with one of the `ips` (keep `:47474`). |
| Every request fails with *unauthorized* | The PC logs the reason (clock off, unknown phone, signature mismatch) in `%LOCALAPPDATA%\PhoneDrop\phonedrop.log`. A clock more than 90 s off, or a hand-built shortcut with stray spaces around `\|`, are the usual causes. |
| SmartScreen blocks the exe | **More info → Run anyway**, after [verifying the download](#verifying-a-download). |

## Verifying a download

PhoneDrop is only published at **github.com/mynameishaheer/phonedrop** (and the winget and
Scoop packages that point there). Each release has a `SHA256SUMS.txt`, and the exe carries a
GitHub build-provenance attestation that ties it to the commit and CI run that built it:

```powershell
Get-FileHash .\phonedrop.exe -Algorithm SHA256       # compare with SHA256SUMS.txt
gh attestation verify .\phonedrop.exe --repo mynameishaheer/phonedrop
```

## Contributing

Bug reports, ideas and pull requests are welcome. Start with [CONTRIBUTING.md](CONTRIBUTING.md);
issues labelled
[good first issue](https://github.com/mynameishaheer/phonedrop/labels/good%20first%20issue) are a
good way in. Questions go in
[Discussions](https://github.com/mynameishaheer/phonedrop/discussions). Security problems: see
[SECURITY.md](SECURITY.md), not a public issue.

## Roadmap

Details in [docs/ROADMAP.md](docs/ROADMAP.md); changes so far in [CHANGELOG.md](CHANGELOG.md).

- **v1.0**: first stable release.
- **iCloud shortcut links**: one-tap install of all six shortcuts, no Mac or signing needed.
- **Optional push notifications** via [ntfy](https://ntfy.sh), so the phone can learn that
  something is waiting without opening Shortcuts.
- **Relay for off-LAN sending**, for when the phone and PC aren't on the same network.
- **Encryption out of beta**, once name-constraint enforcement is verified on real iPhones.

## Licence

MIT. Icons: [Fluent UI System Icons](https://github.com/microsoft/fluentui-system-icons) (MIT).
