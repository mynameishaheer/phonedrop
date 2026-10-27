# Security model

How PhoneDrop protects your PC, what it does not protect against, and the optional encrypted mode.
Wire details are in [PROTOCOL.md](PROTOCOL.md). To report a problem, see [SECURITY.md](../.github/SECURITY.md).

<img src="media/how-it-works.svg" width="640" alt="Diagram: the iPhone sends a signed request over the same Wi-Fi to the PC; a stranger on the Wi-Fi gets a 401 and nothing is shown">

## What protects you

| | |
|---|---|
| **Pairing** | The PC shows a QR code holding a fresh 256-bit secret, valid once for 2 minutes. The secret never crosses the network. The PC then asks you to **Allow** the phone. |
| **Every request** | Signed with the phone's secret (nested SHA-256, computable in Shortcuts), with a timestamp (±90 s) and a single-use nonce. Files are hashed while they stream in and kept only if the hash matches. |
| **Strangers** | Unsigned or bad requests all get the same `401`, show nothing on screen, and an IP that keeps failing is shut out for 5 minutes. |
| **Networks** | PhoneDrop listens only on networks Windows calls *Private* (or Domain). On a café or hotel network, nothing is listening at all. Connections from outside private address ranges are dropped. |
| **At rest** | Phone secrets are encrypted with Windows DPAPI (your user account). Revoke a phone in the panel and its secret is gone. |
| **Received files** | Sanitized names (no paths, no reserved names), never overwritten, marked as downloaded from the internet (Mark-of-the-Web). Programs and scripts are never opened from a toast, only shown in Explorer. |

## Threat model

| Topic | What to know |
|---|---|
| **Eavesdroppers** | Without encryption, someone already on your Wi-Fi who can see the traffic could read what you send, as with any unencrypted LAN traffic. They still can't send to your PC or impersonate your phone. |
| **The pairing file** | On the phone it lives in iCloud Drive (`Shortcuts/PhoneDrop/config.json`), since Shortcuts has no keychain. It syncs to your other Apple devices and iCloud.com. |
| **If it leaks** | Anyone who gets it can, while on the same network as the PC, send files to it and collect whatever is waiting in its outbox, until you remove the phone in the panel. |
| **Not defended against** | A compromised PC or phone, malware running as your Windows user, or someone with your unlocked iCloud account. |

## Encrypt transfers (beta)

**Settings → Encrypt transfers** makes the PC a tiny certificate authority that may only vouch for itself
(X.509 name constraints: its own `.local` name and private IPs). Your iPhone installs it as a profile,
and from then on that phone talks HTTPS only.

- Name constraints are what stop a stolen CA key from being used against other websites.
- **iOS's enforcement of them on user-installed roots hasn't been verified on a device yet**, so the mode is opt-in.
- Until it is verified, leave it off unless you want the added confidentiality and accept that risk.
- Tested it on an iPhone? Results are welcome in a Discussion. Flaws go to [SECURITY.md](../.github/SECURITY.md).

Protocol details: [PROTOCOL.md §7](PROTOCOL.md#7-optional-tls-beta).
