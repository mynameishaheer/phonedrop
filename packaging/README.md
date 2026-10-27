# Packaging

| Folder | What | Built by |
|---|---|---|
| `inno/` | `PhoneDrop-Setup.exe`, the installer most people use | CI, on every push (attached to tagged releases) |
| `winget/` | winget manifest templates | You, once per release |
| `scoop/` | Scoop manifest for your own bucket | You, once |

## Releasing

```sh
git tag v0.9.3 && git push origin v0.9.3
```

CI builds `PhoneDrop-Setup.exe`, the portable `phonedrop.exe`, `install.ps1` and `SHA256SUMS.txt`,
attests both exes and publishes the release (`v0.*` tags are marked pre-release). Keep the
`version` constant in `cmd/phonedrop/settings.go` and `cmd/phonedrop/winres/winres.json` in step
with the tag.

## winget (do this at v1.0)

1. Install [wingetcreate](https://github.com/microsoft/winget-create) and run
   `wingetcreate new https://github.com/mynameishaheer/phonedrop/releases/download/v1.0.0/PhoneDrop-Setup.exe`.
   It fills in the hash and opens a PR to `microsoft/winget-pkgs`. Use the files in `winget/` as
   the answers (identifier `mynameishaheer.PhoneDrop`, Inno installer, user scope).
2. Later versions: `wingetcreate update mynameishaheer.PhoneDrop --version X.Y.Z --urls <setup url> --submit`.

Users then install with `winget install mynameishaheer.PhoneDrop`.

## Scoop

Scoop's main bucket needs an established project, so start with your own bucket: create a repo
`mynameishaheer/scoop-bucket`, put `scoop/phonedrop.json` in its `bucket/` folder with the
version and hash filled in, and users run:

```
scoop bucket add phonedrop https://github.com/mynameishaheer/scoop-bucket
scoop install phonedrop
```

Scoop uses the portable `phonedrop.exe`. `checkver` and `autoupdate` let Scoop's tools bump it for
new releases automatically.
