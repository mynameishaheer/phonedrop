# Security policy

## Supported versions

PhoneDrop is pre-1.0. Security fixes go into the latest release only. Please update before reporting, and test against `main` if you can.

| Version | Supported |
|---|---|
| Latest release (0.9.x) | Yes |
| Anything older | No |

## Reporting a vulnerability

**Please don't open a public issue.** Use GitHub's private vulnerability reporting instead:

1. Go to the repository's **Security** tab.
2. Click **Report a vulnerability**.
3. Describe what you found, how to reproduce it, and what an attacker gains.

You'll get a first reply within a week. This is a volunteer project, so fixes take as long as they take, but we'll tell you where things stand and credit you in the release notes if you'd like.
Please give us a reasonable time to fix a problem before you publish it.

**In scope:**

- Anything that lets someone who is *not* paired send to, read from, or learn about a PC.
- Bypassing the Private-network rule or the lockout.
- Writing outside the download folder.
- Replaying or forging a signed request.
- Extracting a phone's secret from the PC or from the QR flow.

## Threat model in short

The full description is in [docs/SECURITY-MODEL.md](../docs/SECURITY-MODEL.md); the protocol is in [docs/PROTOCOL.md](../docs/PROTOCOL.md).

- **Who is trusted:** phones you paired with a QR code and then confirmed with **Allow** on the PC.
  Nobody else can send or collect anything. Unsigned or bad requests all get the same bare `401`.
- **Where it listens:** only on networks Windows calls *Private* or *Domain* (or ones you
  explicitly chose with **Use here**), and only accepts connections from private address ranges.
  On a café or hotel network nothing is listening.
- **Secrets:** each phone has a 256-bit secret. It travels only inside the QR code, never over the network. On the PC it is encrypted with Windows DPAPI. On the phone it sits in a file in iCloud Drive, because Shortcuts has no keychain, so anyone who gets that file can act as the phone while on the same network as the PC, until you remove the phone in the panel.
- **Without encryption**, someone on your Wi-Fi who can see the traffic can read what you send. They still can't send to your PC or impersonate your phone, because requests are signed with a timestamp and a single-use nonce.
- **Not defended against:** a compromised PC or phone, malware running as your Windows user, or
  someone with your unlocked iCloud account.

## Encrypted mode is beta

**Settings → Encrypt transfers** turns the PC into a small certificate authority that can only vouch for itself (X.509 name constraints) and makes paired phones use HTTPS.
Its safety relies on iOS enforcing those name constraints for user-installed root certificates, and **that enforcement has not yet been verified on a real device**.
Until it is, treat the mode as beta and leave it off unless you want the added confidentiality and accept that risk.

Tested it on an iPhone, either way? Post results in a Discussion; report flaws privately.
See [PROTOCOL.md §7](../docs/PROTOCOL.md#7-optional-tls-beta).
