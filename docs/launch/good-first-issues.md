# Good first issues: ready to paste

For the repo owner. Create each as a GitHub issue with the label `good first issue` (and the
extra label named in brackets), copying the title and body. Delete this file once they exist.

Every body ends with the same pointer to [CONTRIBUTING.md](../../.github/CONTRIBUTING.md); keep it.

---

## 1. Rename a paired phone in the panel  `[enhancement]`

**Title:** Let me rename a paired phone in the panel

**Body:**

A phone gets its name at pairing time (for example "iPhone") and the panel can only remove it. If
you pair two phones with the same name you can't tell them apart.

**What to do**

- Add a **Rename** action next to **Remove** in the paired-phones list (`ui_windows.go`).
- Add a rename method to the phone store (`phones.go`; it already writes atomically to
  `phones.json`) and keep the secret untouched.
- Add a portable unit test in `phones_test.go`.

**Done when:** renaming survives a restart, empty and over-long names are handled, and the toast
and Recent list show the new name.

**Hints:** the Windows-only UI can be checked with `phonedrop.exe -preview <folder>` (works under
Wine). Start with the store and test, which build and run on Linux.

Read [CONTRIBUTING.md](../../.github/CONTRIBUTING.md) first. Comment here to claim this.

---

## 2. Drag files onto the panel to send them  `[enhancement]`

**Title:** Drag and drop files onto the panel to queue them for the phone

**Body:**

Today you queue files for the phone with **Send to → PhoneDrop (to phone)**, the **Files…**
button or Ctrl+Alt+P. Dropping files onto the open panel would be the obvious fourth way.

**What to do**

- Register the panel window for file drops (`DragAcceptFiles` / `WM_DROPFILES` through
  `syscall`, like the other Win32 calls in `win_windows.go`).
- Feed the dropped paths into the same code path as **Files…** (see `send_windows.go`).

**Done when:** dropping one or several files queues them and the outbox card updates; dropping a
folder or a missing file is handled without a crash.

**Constraints:** standard library only, no cgo, and no timers or polling while the panel is hidden.

Read [CONTRIBUTING.md](../../.github/CONTRIBUTING.md) first. Needs a Windows machine to test.

---

## 3. Measure and document idle memory  `[documentation]`

**Title:** Measure PhoneDrop's idle memory use and document it

**Body:**

The README says the target is "under 15 MB private working set" at idle, but nobody has measured
it on real Windows. This one needs no Go.

**What to do**

- Run the latest release on Windows 10 and/or 11 with the panel hidden and no phone connected;
  after 10 minutes, record *Private working set* (Task Manager → Details, add the column; or
  `Get-Process phonedrop | Select PrivateMemorySize64, WorkingSet64`).
- Repeat with the panel open and after receiving a large file.
- Put the numbers and how to reproduce them in the README's "Resource use" section.

**Done when:** the README states measured figures with the Windows version and PhoneDrop version
they came from. If you find it exceeds the target, open a separate issue with what you saw.

Read [CONTRIBUTING.md](../../.github/CONTRIBUTING.md) first.

---

## 4. Shortcuts generator: fix the Get File folder  `[shortcuts]`

**Title:** Generated shortcuts need the Get File folder re-picked by hand

**Body:**

After importing the generated shortcuts, users must open **PhoneDrop Sign** and re-select the
folder in the first action (*Get file from Shortcuts at path PhoneDrop/config.json*), or it reads
empty. See the "After importing" note in `shortcuts/README.md`.

**What to do**

- In `shortcuts/build.py`, find how the *Get File* action's folder is serialized.
- Work out what an iPhone-authored shortcut stores for "iCloud Drive → Shortcuts" (export one
  from the Shortcuts app, or open a signed file's plist) and generate the same.
- If it can't be done reliably, document exactly why in `shortcuts/README.md`.

**Done when:** `python3 -I shortcuts/build.py --check` passes and an imported **PhoneDrop Sign**
reads `config.json` without any manual fix. Signing needs a Mac; say in the pull request what you
could test.

Read [CONTRIBUTING.md](../../.github/CONTRIBUTING.md) first (see "The iPhone shortcuts").

---

## 5. Recognise more file types when a name is missing  `[enhancement]`

**Title:** Add more file signatures to `sniffExt`

**Body:**

When an upload arrives without a usable extension, `sniffExt` in `files.go` guesses one from the
first bytes of the file. It covers the common cases; many are missing.

**What to do**

- Add signatures for formats people really share from an iPhone or want on the PC (for example
  WebP, HEIC/HEIF variants, MKV/WebM, GIF variants, Office documents, ZIP-based formats,
  text with a BOM).
- Add table-driven cases to `files_test.go` with real, tiny sample headers.

**Done when:** `go test -race ./...` passes and each new type has a test. This is portable code,
so any OS works.

**Constraint:** standard library only. Never read more of the file than the header the function
already receives.

Read [CONTRIBUTING.md](../../.github/CONTRIBUTING.md) first.

---

## 6. Move UI strings into one table (localisation groundwork)  `[enhancement]`

**Title:** Collect user-visible strings in one place so they can be translated

**Body:**

Panel text, buttons and toast messages are string literals spread through `ui_windows.go`,
`toast_windows.go`, `pair_windows.go` and `trust_windows.go`.

**What to do**

- Introduce a small `tr(key)`-style lookup with the current English strings as the default table.
- Convert the strings in one file as a first pull request (say, `toast_windows.go`), then the
  rest.
- Choose the language from the Windows UI language with a fallback to English. Don't add a
  dependency.

**Done when:** English output is byte-for-byte unchanged (compare the PNGs from
`phonedrop.exe -preview`), and adding a second language means adding one table.

Discuss the approach in the issue before a large change. Read
[CONTRIBUTING.md](../../.github/CONTRIBUTING.md) first.

---

## 7. Show the version in the panel  `[enhancement]`

**Title:** Show the PhoneDrop version in the panel

**Body:**

Bug reports ask for the PhoneDrop version, but the panel doesn't show it; people have to open the
exe's file properties. The version is the `version` constant in `settings.go`.

**What to do**

- Add a small, low-contrast "PhoneDrop 0.9.2" line to the settings view in `ui_windows.go`,
  using the existing text helpers and theme colours.
- Check light and dark themes with `phonedrop.exe -preview <folder>` and attach the PNGs.

**Done when:** the version shows in both themes and doesn't shift the existing layout.

Read [CONTRIBUTING.md](../../.github/CONTRIBUTING.md) first.
