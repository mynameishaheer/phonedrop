# Packaging

Templates for Windows package managers. They're filled in for each release; nothing here is
built or run by CI.

## winget (do this at v1.0)

1. Tag a release (`git tag v1.0.0 && git push origin v1.0.0`). CI attaches `phonedrop.exe` and
   `SHA256SUMS.txt`.
2. Easiest: install [wingetcreate](https://github.com/microsoft/winget-create) and run
   `wingetcreate new https://github.com/mynameishaheer/phonedrop/releases/download/v1.0.0/phonedrop.exe`.
   It fills in the hash and opens a PR to `microsoft/winget-pkgs`. Use the files in `winget/`
   as the answers (identifier `mynameishaheer.PhoneDrop`, portable installer, description).
3. Later versions: `wingetcreate update mynameishaheer.PhoneDrop --version X.Y.Z --urls <exe url> --submit`.

Users then install with `winget install mynameishaheer.PhoneDrop`. A portable install doesn't
add a sign-in entry by itself: turn on **Settings → Start when I sign in** in the panel.

## Scoop

Scoop's main bucket needs an established project, so start with your own bucket: create a repo
`mynameishaheer/scoop-bucket`, put `scoop/phonedrop.json` in its `bucket/` folder with the
version and hash filled in, and users run:

```
scoop bucket add phonedrop https://github.com/mynameishaheer/scoop-bucket
scoop install phonedrop
```

`checkver` and `autoupdate` let Scoop's tools bump it for new releases automatically.
