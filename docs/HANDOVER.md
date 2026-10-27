# Handover

Everything a new agent or developer needs to pick up PhoneDrop exactly where it was left.
Last updated 27 Oct 2026. Read this whole page before changing anything.

## 1. What PhoneDrop is

AirDrop-style sharing between an iPhone and a Windows PC.

- **iPhone side:** Apple Shortcuts only, no app. Shortcuts run from the share sheet, Back Tap,
  the Action Button or Siri.
- **PC side:** one stdlib-only Go tray exe (`cmd/phonedrop/`), about 7 MB, no cgo, no runtime, no
  timers when idle.
- **Direct over the LAN.** No server or account in between.
- **Paired explicitly on both sides.** The PC shows a QR code, the phone scans it, and the PC asks
  Allow/Deny. Strangers get a bare `401`.

Repo: <https://github.com/mynameishaheer/phonedrop> (public, MIT). Owner: Shaheer Ahmed
(GitHub `mynameishaheer`). Owner's machines: a Windows laptop, an iPhone on iOS 27, and a borrowed
Mac for signing shortcuts. Owner's time zone: Pakistan (UTC+5).

## 2. How to work in this repo (owner's rules)

- **Commit straight to `main` and push.** The owner asked for this; there are no PRs or branches.
  Commit as the owner:
  `git -c user.name="Shaheer Ahmed" -c user.email="shaheer.ahmed12@gmail.com" commit ...`
  End commit messages with the attribution trailer your harness asks for. Never put a model name
  or ID in commits or files.
- **Stdlib only, no cgo, idle costs nothing.** These are hard rules (`.github/CONTRIBUTING.md`).
- **Before every commit:** `gofmt -l .` (must print nothing), `go vet ./...`,
  `GOOS=windows GOARCH=amd64 go vet ./...`, `go test ./...`, and `scripts/build.sh`.
- **Docs style:** short sentences, tables and numbered steps, no marketing adjectives, no emoji.
  The owner finds long text hard going and wants images where possible.
- **Owner preferences:** keep the repo root clean; non-technical users must be able to install
  without a terminal; ask before big product decisions (for example the rename); the owner tests
  on real devices and reports results with screenshots.
- **Agents:** the owner originally asked for an orchestrator that spawns sub-agents: Opus for
  hard tasks, Sonnet for normal ones.

## 3. Repo map

| Path | What |
|---|---|
| `cmd/phonedrop/` | All Go code, one `package main`. `*_windows.go` = UI and Win32; the rest is portable and tested on Linux. Also the embedded icon fonts, `rsrc_windows_amd64.syso` (icon/manifest/version) and `winres/` (its source). |
| `scripts/build.sh` | Runs tests, cross-compiles `phonedrop.exe` into the repo root (git-ignored). |
| `scripts/install.ps1` | Scripted/portable install, same target folder as the installer. Attached to releases. |
| `packaging/inno/phonedrop.iss` | The Inno Setup installer → `dist/PhoneDrop-Setup.exe`. Built by CI on Windows. |
| `packaging/winget/`, `packaging/scoop/` | Manifest templates (placeholders `VERSION`, `SHA256_OF_…`). Not submitted yet. |
| `shortcuts/` | `build.py` generates and validates the six `.shortcut` files into `unsigned/`; `sign-all.sh` signs them on a Mac; `probes.py`/`sign-probes.sh` find what Apple's signing server rejects. |
| `docs/` | `PROTOCOL.md` (wire spec), `DEV.md` (code map), `SHORTCUTS.md` (hand-build guide), `ROADMAP.md`, `HOW-IT-WAS-BUILT.md`, `SECURITY-MODEL.md`, `FAQ.md`, `media/` (screenshots), `launch/` (launch kit and research). |
| `.github/` | CI workflow, issue/PR templates, CONTRIBUTING, SECURITY, CODE_OF_CONDUCT. |

Code map per file: `docs/DEV.md`. Protocol details: `docs/PROTOCOL.md`.

## 4. Architecture in one screen

- **Server** (`server.go`): net/http on port 47474. Routes:
  - `POST /v1/pair`: QR pairing; plain-text or JSON phone name.
  - `POST /v1/items`: phone → PC uploads, streamed to hidden `.part` files and hash-verified.
  - `GET /v1/outbox` and `/v1/outbox/{id}`: PC → phone items.
  - `GET /v1/ping`.
  - signed `GET /v1/ca.mobileconfig`: profile for TLS mode.

  Other server behaviour:
  - Per-IP gate with a 5-minute penalty after repeated failures.
  - Connection caps: 32 per IP, 256 total.
  - Connections from outside private address ranges are dropped (`privatePeer`).
  - A refused request is logged with its reason (`auth: … refused: <why>`).
- **Auth** (`auth.go`):
  - `sig = hex(SHA256(secret|inner))`, where `inner = hex(SHA256(secret|METHOD|PATH|TIME|NONCE|BODYHASH))`.
  - Headers: `X-PD-Phone`, `X-PD-Time`, `X-PD-Nonce`, `X-PD-Body`, `X-PD-Sig`.
  - ±90 s clock skew, a nonce cache, constant-time compare.
  - Every failure gets the same generic 401.
  - **Response bodies:** error bodies never contain the word `true` and success bodies always do,
    because the shortcuts check "contains true".
- **Pairing** (`pairing.go`, `pair_windows.go`):
  - QR JSON: `{"pd":1,"name","host","ips","port","base","id","secret"[,"tls":true,"ca":<sha256 of profile>]}`.
  - The code is single-use and valid for 2 minutes, then the PC shows an Allow/Deny prompt.
  - Phones are stored with DPAPI (`phones.go`, `dpapi_windows.go`).
- **Networks** (`netwatch_windows.go`, `trust_windows.go`):
  - Listens only when the network is Private or Domain (Network List Manager COM) or a Public
    network the user trusted with **Use here** (`settings.Trusted`).
  - **Use here** can add a Public-profile firewall rule through elevated PowerShell. The rule is
    named `PhoneDrop (trusted Public networks)`.
  - Change detection: `NotifyAddrChange` and `RegNotifyChangeKeyValue`.
- **TLS beta** (`tls.go`, `mobileconfig.go`):
  - A per-PC CA with name constraints: its DNS host plus the private IP ranges, serverAuth only,
    email and URI names constrained to "invalid".
  - A TLS-sniffing listener on the same port.
  - A deterministic mobileconfig whose hash is pinned in the QR code.
- **UI:**
  - GDI+ layered windows (`gdip_windows.go`, `ui_windows.go`), per-monitor DPI v2, tray icon (NOTIFYICON v4).
  - Toasts (`toast_windows.go`). Their **Open** button uses an allow-list; anything else is only shown in Explorer.
  - Explorer **Send to** reaches the running instance through WM_COPYDATA (`forwardToRunning`).
  - Hotkey Ctrl+Alt+P sends the clipboard; IFileOpenDialog picks files.
- **Received files** (`files*.go`): sanitized names, never overwritten, Mark-of-the-Web.
- **Data folder:** `%LOCALAPPDATA%\PhoneDrop`. It holds `settings.json`, the phones file,
  history and `phonedrop.log`, which is recreated on each run.
- **Program folder:** `%LOCALAPPDATA%\Programs\PhoneDrop`, used by both the installer and `install.ps1`.
  Before 27 Oct, `install.ps1` put the exe in the data folder; both installers now delete it from there.
- **Start at sign-in:** `HKCU\…\CurrentVersion\Run`, value `PhoneDrop`. The Run key is the source of
  truth that the panel toggle reads.
- **Version:** `version = "0.9.2"` in `cmd/phonedrop/settings.go`. Keep
  `cmd/phonedrop/winres/winres.json` in step, then regenerate the `.syso` (command in `scripts/build.sh`).
- **Preview mode:** `phonedrop.exe -preview <dir>` renders every panel state to PNGs. It runs under
  Wine here: `wine phonedrop.exe -preview preview/`. Wine can check layout only; it can't test
  networking.

## 5. Current state (27 Oct 2026)

**Works on the owner's real devices:**
- Pairing.
- Send Text to PC, Send Latest Screenshot.
- Big file uploads: 216 MB in 19 s.
- **Use here** on a Public network.

**CI** (`.github/workflows/build.yml`):
- Job `build` (Ubuntu): gofmt, vet (also for Windows), tests with `-race`, build.
- Job `installer` (Windows): Inno Setup through choco if missing, builds `PhoneDrop-Setup.exe`,
  then **smoke-tests** it: silent install, check the files and the Run key, silent uninstall,
  check it all went.
- Job `release`: on a push to `main` whose version has no release yet (or on a pushed `v*` tag).
  Writes `SHA256SUMS.txt`, adds provenance attestations for both exes, and creates the tag and
  release with notes from CHANGELOG.md.

**CI history:**
- `11a88c3` (the folder move) failed: the Checksums step still pointed at `install.ps1` in the root.
- `83228f6` fixed that and added the installer. Build passed, Inno Setup compiled, but the smoke
  test hung: pwsh `Start-Process -Wait` also waits for child processes, and setup leaves
  PhoneDrop running. Cancelled.
- The commit that added this page fixes the waits (`WaitForExit`, 20-minute job timeout). **Check
  its run first**: `gh api "repos/mynameishaheer/phonedrop/actions/runs?head_sha=<sha>"`, then
  `/actions/runs/<id>/jobs` for the failed step. `gh run view --log` is blocked in the cloud
  sandbox; use the jobs API and `check-runs/<job id>/annotations`.

**No release has been published yet.** CI on `main` is green (build, installer compile, silent install/uninstall smoke test, run 40562636453). Releases are published automatically; see §6 item 3.

**Docs rewrite: committed** with this page. A Sonnet agent shortened every doc and added the
screenshots. Leftovers it flagged:
- `docs/media/panel-full.png` shows "Not connected" at the top. Re-render it with a connected
  state if it's ever used as a hero.
- `docs/media/settings.png` isn't used anywhere.
- iPhone screenshot placeholders are HTML comments in README, SHORTCUTS.md and shortcuts/README.md.
  They need real phone screenshots from the owner.

## 6. Open items, in priority order

| # | Item | Owner | Notes |
|---|---|---|---|
| 1 | Make sure CI is green on `main` | Agent | See §5. The build job, checkout@v5, setup-go@v6, Inno Setup and the compile all work. Only the smoke test is unproven. |
| 2 | Re-check the docs | Agent | Optional: README is 160 lines; the owner wants it short. |
| 3 | Check the `v0.9.2` release | Agent | Releases are now automatic. On every push to `main`, the `release` job checks whether `v<version>` (from `cmd/phonedrop/settings.go`) has a release yet. If not, it creates the tag and the release, titled `PhoneDrop vX.Y.Z`. The notes are that version's CHANGELOG section plus install instructions. The files are PhoneDrop-Setup.exe, phonedrop.exe, install.ps1 and SHA256SUMS.txt, with attestations. Releases are never marked pre-release: GitHub hides those from the sidebar and from `/releases/latest`. The commit that made this change should have published v0.9.2; check it on the Releases page. **To release a new version:** bump `version` in settings.go and `winres.json` (regenerate the .syso), add a CHANGELOG `## [X.Y.Z]` section, push. The cloud sandbox itself can't push tags, which is why the job doesn't rely on one. |
| 4 | Owner tests PhoneDrop-Setup.exe on the laptop | Owner | **Not tested on real Windows yet.** Not yet seen on a real machine: the firewall task (it runs `netsh` through `runas`, one UAC prompt), upgrading over an old `install.ps1` install, the finish-page text, SmartScreen behaviour, and Send to working from the new path. |
| 5 | Owner signs and tests the shortcuts | Owner | Regenerate with `python3 -I shortcuts/build.py`, zip the `shortcuts/` folder, and the owner runs `sign-all.sh` on the Mac. Needs testing on the device: the regenerated **Send to PC** and **Receive from PC**, and the If-value fix. **Send Latest Screenshot** failed to sign earlier with a 500; the fix was retries (Apple fails at random). If it still fails, bisect with `probes.py`. |
| 6 | **Rename the product** | Owner decides | At least 7 GitHub repos are called PhoneDrop, including ImSammyTTV/PhoneDrop, an iPhone↔Windows tray app released in Oct 2026. Candidates: **TossPad** (recommended), HopSend, PocketPipe, SlingPC, PocketBeam. The owner must first run the USPTO/EUIPO and app-store name checks. The rename touches the module path, `appName`, the data folder (needs a migration so paired phones survive), the installer's AppName (keep the same AppId GUID), the shortcuts (folder `PhoneDrop/` in iCloud, shortcut names), all docs and the repo name. |
| 7 | One-tap iCloud links for the shortcuts | Owner | The #1 launch blocker. After signing, share each shortcut from the Shortcuts app and put the links in the README. |
| 8 | Code signing via SignPath Foundation | Owner applies, agent wires CI | Free for OSS. Add a CODE_SIGNING.md modelled on LocalSend's. |
| 9 | winget submission | Agent prepares, owner submits | `packaging/README.md` has the steps. The manifest targets the Inno installer, user scope. |
| 10 | Demo GIF and videos | Owner records | The placeholder is commented out at the top of README.md; save it as `docs/media/demo.gif` (under 3 MB, 8–12 s). |
| 11 | Landing page (GitHub Pages) and press kit (`docs/press/`) | Agent | See the marketing plan. |
| 12 | Repo settings | Owner | `docs/launch/repo-settings.md`: Discussions, private vulnerability reporting, topics, branch protection (careful: it would block direct pushes to main), social preview. Also replace `[INSERT CONTACT]` in `.github/CODE_OF_CONDUCT.md`. |
| 13 | Create the good first issues | Owner or agent with a token | 7 texts in `docs/launch/good-first-issues.md`. The uninstall-entry issue was dropped, because the installer solves it. |
| 14 | One human security review | Owner | Before marketing the security story. |
| 15 | Verify the TLS beta on an iPhone | Owner | Does iOS enforce X.509 name constraints on user-installed roots? |
| 16 | Node 20 deprecation in CI | Agent | `upload-artifact`/`download-artifact@v4` still run on Node 20; bump when Node 24 versions are confirmed. |

**Roadmap after v1.0** (`docs/ROADMAP.md`): optional ntfy push, a relay for sending off the LAN,
encryption out of beta, a Microsoft Store (MSIX) listing.

## 7. Hard-won lessons (don't relearn these)

**Apple Shortcuts files** (`shortcuts/build.py`):
- Apple's signing server (`shortcuts sign --mode anyone`) returns HTTP 500 for:
  - an If on Extension Input,
  - feeding text straight into Open URLs (put a URL action in between),
  - dictionary-key aggrandizements on Repeat Item (use Get Dictionary Value, then Text).
- It also fails at random, so `sign-all.sh` retries 3 times.
- `WFConditionalActionString` must be a WFTextTokenString (`tok()`); a plain string imports blank.
- An If only offers "is / begins with" when its input is Text, so convert dictionary values to
  Text before comparing.
- A "no input" check must be count-based.

**Hand-built shortcuts** (`docs/SHORTCUTS.md`):
- Spaces around `|` in the string to sign break every signature.
- Use Set Name before saving, or `config.json` saves as `config.txt`.
- Use Set Variable to name results; Rename doesn't work for this.
- Write the URL out in every Get Contents step.
- The Get File folder must be re-picked by hand (iCloud Drive → Shortcuts).
- If `.local` doesn't resolve, replace the host in `base` with one of the `ips` in `config.json`.

**Windows:**
- Most new Wi-Fi networks are *Public*, which is why **Use here** exists.
- Every pointer passed to Win32 goes inline (`uintptr(unsafe.Pointer(&x))`) into helpers marked
  `//go:uintptrescapes`. Never stash a uintptr first.

**Sandbox limits (cloud session):**
- GitHub GraphQL is blocked; use REST via `gh api`.
- `gh run view --log` is blocked; use the jobs and annotations APIs.
- jrsoftware.org, reddit.com, news.ycombinator.com, USPTO and the app stores are unreachable.
- Wine works for `-preview` only.

## 8. Launch and marketing

- **Plan:** "PhoneDrop launch and marketing plan", a Claude Docs doc at
  <https://claude.ai/artifact/3hRS4CSVLPPcekw8t4nzTj>. It is private to the owner and may not open
  for another account.
- **Raw research:** `docs/launch/research/` holds the competition and name check, the launch
  playbook, and the case studies of famous repos, READMEs and the AI angle.
- **Key decisions in the plan:**
  - Rename first (§6).
  - Don't launch until both installs take under 2 minutes: iCloud links, a signed exe or winget,
    and the installer.
  - Show HN on **Tue 17 Nov 2026, 8am PT**, one channel per day around it. If the iCloud links
    aren't live by 13 Nov, the fallback is the week of 30 Nov.
  - Lead with the product, not the AI story; disclose the AI process honestly
    (`docs/HOW-IT-WAS-BUILT.md`).
  - Post from a personal creator account on Instagram and TikTok, following a 15-post calendar
    starting 1 Nov.
  - Pitch the Shortcuts writers: Viticci, Stephen Robles, Matthew Cassinelli.
  - Position as the replacement for Intel Unison, which was discontinued in 2025.
  - GitHub Sponsors may not pay out in Pakistan; use Ko-fi or Liberapay.
- **Realistic month one:** 20–100 stars if quiet, 200–600 if good.

## 9. Files outside the repo

The container is temporary, so assume these are gone. All of them can be regenerated:

- The release zip: `scripts/build.sh`.
- The shortcut zips: `python3 -I shortcuts/build.py`, then zip `shortcuts/`.
- The panel screenshots: the `-preview` command above.
