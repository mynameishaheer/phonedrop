# Ready-made iPhone shortcuts

Instead of building the six shortcuts by hand ([docs/SHORTCUTS.md](../docs/SHORTCUTS.md)), you
can import them. Apple only imports signed shortcuts, and signing needs a Mac.

| File | Does |
|---|---|
| `PhoneDrop Sign` | Helper the others call by name. Never run it yourself. |
| `PhoneDrop Setup` | Pairs the iPhone by scanning the PC's QR code. |
| `Send to PC` | Share sheet: photos, videos, files. Run without input, it lets you pick photos. |
| `Send Text to PC` | Share sheet: text and links. Run without input, it sends the clipboard. |
| `Send Latest Screenshot` | Sends your newest screenshot (for Back Tap / Action Button). |
| `Receive from PC` | Collects what the PC queued: photos to Photos, files to Files, text to the clipboard. |

## Install in 3 steps

1. **Sign on a Mac.** Copy this `shortcuts` folder to the Mac and run

   ```sh
   ./sign-all.sh
   ```

   It signs every file in `unsigned/` into `signed/` (`shortcuts sign --mode anyone`). The Mac
   must be signed in to iCloud and online.

2. **AirDrop** the files from `signed/` to the iPhone. Send `PhoneDrop Sign` first.

3. On the iPhone, tap **Add Shortcut** for each one.

   - If iOS refuses with a message about sharing, turn on
     **Settings → Apps → Shortcuts → Private Sharing** and try again.
   - **Delete your hand-built `PhoneDrop Sign` and `PhoneDrop Setup` first** (and any other
     PhoneDrop shortcuts with the same names). Otherwise the imported ones are named
     "PhoneDrop Sign 2" and so on, and the other shortcuts, which call **PhoneDrop Sign** by
     name, keep using your old one. You don't need to pair again: the pairing lives in
     iCloud Drive → Shortcuts → PhoneDrop → `config.json`, not in the shortcut.

After importing, open **PhoneDrop Sign** once in the editor and check that the first action
reads *Get file from **Shortcuts** at path PhoneDrop/config.json*. If the folder is empty, tap it
and pick iCloud Drive → **Shortcuts**. Then run **Send Text to PC** with something on the
clipboard to test.

Also turn on **Settings → Apps → Shortcuts → Advanced → Allow Sharing Large Amounts of Data**
for photos and videos.

## Rebuilding

`build.py` generates the unsigned files from scratch (Python 3, standard library only) and
checks them: every file must parse, every If/Repeat must be closed, every variable must be set
before use, and every variable placeholder must sit where the text says it does.

```sh
python3 -I build.py            # writes unsigned/*.shortcut and validates them
python3 -I build.py --check    # validates only
python3 -I build.py --xml /tmp/pd-xml   # also writes readable XML copies
```

The output is deterministic, so a rebuild without changes gives identical files.
