# How PhoneDrop was built

PhoneDrop was built in October 2026 by one person working with [Claude Code](https://claude.com/claude-code) AI agents.
This page says how, what was checked, what went wrong and what is still unverified, so you can decide how much to trust it.

## Who did what

| Role | Who |
|---|---|
| Product owner, decisions, testing on a real iPhone and Windows laptop | The maintainer (a person) |
| Lead developer and reviewer of every change | Claude (Claude Code), coordinating the agents below |
| QR-code encoder (written from the ISO spec, no libraries) | A Claude sub-agent, checked against an independent decoder (zxing-cpp): 203 round-trip cases |
| Request signing, phone store, pairing | A Claude sub-agent |
| Upload endpoint, outbox, file-name sanitising | Claude sub-agents |
| Optional TLS mode | A Claude sub-agent |
| **Release review**: read every file looking for bugs and security holes | A separate Claude agent whose only job was to break it |
| Fixes for the review findings | Claude sub-agents, then reviewed again by the lead |

Everything is in the public git history, commit by commit.

## What keeps it honest

- **Standard library only.** No third-party Go modules, so there's no dependency supply chain.
- **A written protocol and threat model**, weaknesses included: [PROTOCOL.md](PROTOCOL.md) and [SECURITY-MODEL.md](SECURITY-MODEL.md).
- **Tests run in CI with the race detector**: signatures (with a published test vector), replay protection, pairing, uploads, path traversal, the outbox, TLS name constraints and connection limits.
- **Release builds come from CI**, with SHA-256 checksums and GitHub build-provenance attestations, so you can check the exe was built from this repository.

## What the release review found (and what was fixed)

Before the first public build, the review agent reported about 30 issues. All were fixed. The most serious:

| Severity | Finding | Fix |
|---|---|---|
| High | Pointers passed to Windows APIs could be moved or freed by Go's runtime while the call was in progress | Every call now converts pointers inline, with `go:uintptrescapes` on the call helpers |
| High | Every redraw leaked a GDI bitmap; after days, drawing would fail | The canvas now releases its bitmap correctly |
| Medium | The optional TLS certificate authority could, if its key were stolen, still sign email/URI certificates | The CA is limited to server certificates and constrained for email and URI names too |
| Medium | The "Open" button in notifications used a deny-list of dangerous file types | It now uses an allow-list; anything else is only shown in Explorer |
| Medium | Unlimited connections per IP | Capped per IP and in total; silent connections count as failures |
| Medium | Profile for TLS mode downloaded over plain HTTP could be swapped on the network | The QR code carries the profile's hash; the phone checks it before installing |

## What went wrong along the way

Real-device testing found things no test could:

| Finding | Result |
|---|---|
| Windows marks most new Wi-Fi networks *Public*, where PhoneDrop deliberately doesn't listen. | The **Use here** button. |
| Building shortcuts by hand was far too fiddly. A single extra space around a `|` broke every signature. | The shortcut generator in `shortcuts/` and the plan for one-tap links. |
| Apple's shortcut-signing service rejects some valid-looking shortcut files. | We found which patterns by signing tiny one-action test shortcuts (`shortcuts/probes.py`). |

## Still unverified

- **Encrypted mode (beta)** relies on iOS enforcing X.509 name constraints on user-installed
  certificates. That hasn't been checked on a real iPhone yet.
- **No independent human security review yet.** If you know your way around this, please [try to break it](../.github/SECURITY.md). Reviewers will be credited.
