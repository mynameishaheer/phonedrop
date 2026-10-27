# FAQ

The short list is in the [README](../README.md#faq). The rest is here.

## About PhoneDrop

| Question | Answer |
|---|---|
| **Is this AirDrop?** | No. AirDrop uses Apple's private Wi-Fi link (AWDL), which Windows can't speak. PhoneDrop sends over your normal network and gives you similar one-tap sharing. |
| **Why Shortcuts instead of an app?** | An App Store app means a developer account, review and a constantly running background service on the phone. Shortcuts already live in the share sheet, on Back Tap and on the Action Button. The cost: the PC can't push to the phone, so you run **Receive from PC** to collect. |
| **Does it work with Android?** | Not today. The shortcuts are for iOS, and the PC app only knows the PhoneDrop protocol. The protocol is plain HTTP and fully documented in [PROTOCOL.md](PROTOCOL.md), so a client for another phone is possible. Contributions welcome. |
| **Does it work on a Mac?** | There's no macOS app, and a Mac already has AirDrop. The server is a Windows tray app. |
| **Was this written by AI?** | Largely, yes, with a person deciding, testing on real devices and reviewing. [HOW-IT-WAS-BUILT.md](HOW-IT-WAS-BUILT.md) says who did what, what was checked, and what is unverified. |

## Networks and safety

| Question | Answer |
|---|---|
| **Does it work away from my home Wi-Fi?** | On any network where both devices can reach each other and Windows calls the network *Private* (home, and most offices that allow device-to-device traffic). Not across the internet or between different networks yet. A relay for that is on the [roadmap](ROADMAP.md). |
| **Is it safe on public Wi-Fi?** | By default PhoneDrop doesn't listen at all on networks Windows calls *Public*, so there is nothing to attack. If you click **Use here** on one, only your paired phone can send, but without *Encrypt transfers* (beta) other people on that network could read what you send. Prefer your own network. |
| **Windows warns that the app is unsigned.** | Yes, it isn't code-signed yet. You can inspect the source and build it yourself with `scripts/build.sh`. CI builds the released exe from the tagged commit. See [Verifying a download](../README.md#verifying-a-download). |

More on the threat model: [SECURITY-MODEL.md](SECURITY-MODEL.md).

## Resource use

| | |
|---|---|
| **Idle** | 0% CPU (blocking waits only, no timers while the panel is hidden) and a small, flat memory footprint (target: under 15 MB private working set). |
| **Transfers** | Stream to disk, so memory stays flat whatever the file size. |
| **Speed** | Tested phone → PC: 216 MB in 19 s (11 MB/s) over ordinary home Wi-Fi. |
