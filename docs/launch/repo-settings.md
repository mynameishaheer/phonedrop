# Repository settings checklist

For the repo owner (the lead may apply some of these through the API). Delete this file when done.

## About box

**Description** (under 350 characters):

> AirDrop-style sharing between an iPhone and a Windows PC. No iPhone app: uses Apple Shortcuts. One small tray exe (Go, standard library only) that uses no CPU while idle. Photos, files, text and links over your own Wi-Fi, paired once with a QR code, signed requests, no cloud or account.

(287 characters.)

**Website:** leave empty until there is one (or point at the latest release).

**Topics** (GitHub allows 20; lower-case, hyphens):

```
airdrop
iphone
ios
windows
ios-shortcuts
apple-shortcuts
file-transfer
file-sharing
local-network
lan
golang
tray-app
system-tray
clipboard-sync
privacy
open-source
qr-code
no-cloud
```

## Social preview image

Settings → General → Social preview → Upload an image.

- Size **1280 × 640 px** (2:1), PNG or JPG, under 1 MB. Keep important content about 40 px inside
  the edges; some sites crop.
- Suggested content: the app icon (`winres/icon.png`), the name "PhoneDrop", the one-line tagline
  ("AirDrop-style sharing between iPhone and Windows. No iPhone app."), and a simple phone → PC
  arrow. Plain background, large text; it is shown small.
- Don't include screenshots with real file names, network names or IP addresses.

## Settings to enable

- [ ] **Issues**: on (Settings → General → Features).
- [ ] **Discussions**: on. Create categories: *Q&A*, *Ideas*, *Show and tell*. The issue chooser links here.
- [ ] **Private vulnerability reporting**: Settings → Code security → *Private vulnerability
      reporting* → Enable. `SECURITY.md` tells people to use it.
- [ ] **Dependabot alerts / security updates**: on (there are no Go dependencies, but it is free;
      add `github-actions` version updates if you like).
- [ ] **Branch protection on `main`** (Settings → Branches → Add rule for `main`):
  - [ ] Require a pull request before merging.
  - [ ] Require status checks to pass: select **build** (the job from `.github/workflows/build.yml`;
        it only appears in the list after it has run once) and require branches to be up to date.
  - [ ] Block force pushes and deletion.
  - [ ] Optional: require linear history; allow yourself to bypass while you are the only maintainer.
- [ ] **Merge options**: allow squash merging (default to it); disable merge commits if you want a linear history.
- [ ] **Delete branch on merge**: on.
- [ ] **Actions → General → Workflow permissions**: the workflow asks for `contents: write` itself, so
      the default read-only setting is fine.
- [ ] **Releases**: when ready, `git tag v1.0.0 && git push origin v1.0.0`; the workflow builds and
      attaches `phonedrop.exe` and `install.ps1`. (The README and CHANGELOG already link `v0.9.2`;
      tag that first if you want those links to work.)

## Labels

Create (or rename the defaults to): `bug`, `enhancement`, `documentation`, `shortcuts`,
`good first issue` (colour `#7057ff`), `help wanted`, `windows`, `ios`, `security-adjacent`
(for issues that touch auth or the network surface but aren't vulnerabilities). The issue forms
apply `bug` and `enhancement` automatically, and `CONTRIBUTING.md` links to `good first issue`.

## Before announcing

- [ ] Fill in the `[INSERT CONTACT]` placeholder in `CODE_OF_CONDUCT.md`.
- [ ] Record `docs/media/demo.gif` and uncomment it in the README (see the comment there).
- [ ] Create the good first issues from `docs/launch/good-first-issues.md`.
- [ ] Tag a release so the "latest release" badge and install links work.
