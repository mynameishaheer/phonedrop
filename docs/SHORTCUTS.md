# Setting up the iPhone shortcuts

PhoneDrop needs no iPhone app. It uses six shortcuts you build once in Apple's **Shortcuts** app:

| Shortcut | What it does | Section |
|---|---|---|
| **PhoneDrop Sign** | Helper used by the others (signs each request). You never run it yourself. | §1 |
| **PhoneDrop Setup** | Pairs the iPhone with your PC by scanning the QR code. Run once. | §2 |
| **Send to PC** | Share sheet: photos, videos and files. With no input it lets you pick photos. | §3 |
| **Send Text to PC** | Share sheet: text and links. With no input it sends the clipboard. | §4 |
| **Send Latest Screenshot** | One tap (Back Tap, Action Button, Control Center) sends your newest screenshot. | §5 |
| **Receive from PC** | Collects what the PC sent: photos to Photos, files to Files, text to the clipboard. | §6 |

> Building takes about 20 minutes, once. Afterwards you can share each shortcut from the
> Shortcuts app as an iCloud link, so other people install it with one tap.

**How to read the steps.** `Action name` is what you type in the search bar at the bottom of the
editor. **Bold** words are settings inside an action. When a step says "then `Set Variable`
**Name** to it", add a **Set Variable** action right after it, type the name, and set its input
to that action's output. Later, *Name* in italics means: tap the field, choose **Select
Variable**, and pick the variable *Name*. *Config.secret* means: insert the variable *Config*,
tap it, and choose the key `secret`. Type the `|` characters and keys exactly as shown, with no
spaces.

Before you start: in **Settings → Apps → Shortcuts → Advanced**, turn on **Allow Sharing Large
Amounts of Data**.

---

## 1. PhoneDrop Sign (helper)

Create a new shortcut named exactly **PhoneDrop Sign**. It reads the pairing file and returns
the URL and headers for one request. The other shortcuts call it with a *Dictionary* holding
`method`, `path` and `body`.

1. `Get File from Folder`: folder **Shortcuts**, path `PhoneDrop/config.json`, **Error If Not
   Found** on. Shortcuts may pre-fill the folder with *Shortcut Input*: tap it, **Clear**, then
   pick iCloud Drive → **Shortcuts**. Then `Set Variable` **ConfigFile** to it.
2. `Get Dictionary from Input` (input: *ConfigFile*). Then `Set Variable` **Config** to it.
3. `Format Date`: date **Current Date**, **Date Format** *ISO 8601*, **Include ISO 8601 Time**
   on. Then `Set Variable` **Time** to it.
4. `Random Number` from `100000000` to `999999999`. Then `Set Variable` **R1** to it.
5. `Random Number` from `100000000` to `999999999`. Then `Set Variable` **R2** to it.
6. `Text`: *R1* *R2* *Time*. Then `Generate Hash` **SHA256** of that *Text*, then
   `Set Variable` **Nonce** to it.
7. `Text` (one line):
   *Config.secret*`|`*Shortcut Input.method*`|`*Shortcut Input.path*`|`*Time*`|`*Nonce*`|`*Shortcut Input.body*

   For *Shortcut Input.method*, insert **Shortcut Input**, tap it, choose **Dictionary**, key
   `method`. Do the same for `path` and `body`.
8. `Generate Hash` **SHA256** of the *Text* from step 7. Then `Set Variable` **Inner** to it.
9. `Text`: *Config.secret*`|`*Inner*.
10. `Generate Hash` **SHA256** of the *Text* from step 9. Then `Set Variable` **Sig** to it.
11. `Text`: type `http://`, then insert *Config* (key `base`) and, right after it with no space,
    **Shortcut Input** → **Dictionary** → key `path`. It reads
    `http://` [Config: base] [Shortcut Input: path]. Then `Set Variable` **URL** to it.

    *Only if you turn on **Encrypt transfers** on the PC (§7):* this Text must say `https://`
    for everything except Setup's two requests (which pass `plain`). See §7 for the extra steps.
12. `Dictionary`, with these keys (type **Text**):

    | Key | Value |
    |---|---|
    | `url` | *URL* |
    | `phone` | *Config.id* |
    | `time` | *Time* |
    | `nonce` | *Nonce* |
    | `body` | *Shortcut Input.body* |
    | `sig` | *Sig* |

13. `Stop and Output`: the *Dictionary*.

In the shortcut's settings (ⓘ), make sure **Show in Share Sheet** is off.

**Every request below** uses `Get Contents of URL` with URL *Req.url*, the method given, and these
five **Headers**, where *Req* is what *PhoneDrop Sign* returned (insert *Req*, tap it, pick the key):

| Header | Value |
|---|---|
| `X-PD-Phone` | *Req.phone* |
| `X-PD-Time` | *Req.time* |
| `X-PD-Nonce` | *Req.nonce* |
| `X-PD-Body` | *Req.body* |
| `X-PD-Sig` | *Req.sig* |

The PC's answers always contain the word `true` when something worked, and never when it didn't.
That's what the shortcuts check for. *Get Contents of URL* hands back its answer as a file, and
`If` only offers **contains** for text, so each check first puts the answer in a `Text` action.

## 2. PhoneDrop Setup

Create **PhoneDrop Setup**.

1. `Scan QR or Barcode`. Then `Set Variable` **QR** to it.
2. `Get Dictionary from Input` (input: *QR*). Then `Set Variable` **Pairing** to it.
3. `If` *QR* **contains** `secret`. Inside the If:
   1. `Set Name` of *QR* to `config.json` (otherwise Shortcuts may save it as `config.txt`).
      Then `Save File`: input *Renamed Item*, **Ask Where to Save** off, **Destination Path**
      `PhoneDrop/config.json`, **Overwrite If File Exists** on.
   2. `Get Device Details`: **Device Name**. Then `Set Variable` **Name** to it.
   3. `Generate Hash` **SHA256** of *Name*. Then `Set Variable` **BodyHash** to it.
   4. `Dictionary`: `method` = `POST`, `path` = `/v1/pair`, `body` = *BodyHash*, `plain` = `1`.
   5. `Run Shortcut` **PhoneDrop Sign** with the *Dictionary* (**Show While Running** off).
      Then `Set Variable` **Req** to it.
   6. `Get Contents of URL`: URL *Req.url*, **POST**, the five headers from *Req*, **Request Body** **File** = *Name*.
      Then `Set Variable` **Result** to it.
   7. `Text`: *Result* (just the variable). Then `If` that *Text* **contains** `true`: `Show Alert` "Paired! You're ready to send."
      **Otherwise**: `Show Alert` "Pairing didn't work: *Result*", then `Stop This Shortcut`.
      **End If**.
   8. *Only if you use **Encrypt transfers** on the PC (§7); otherwise skip.*
      `Get Dictionary Value` key `tls` in *Pairing*. `If` *Dictionary Value* **has any value**:
      1. `Dictionary`: `method` = `GET`, `path` = `/v1/ca.mobileconfig`, `body` = `-`,
         `plain` = `1`. `Run Shortcut` **PhoneDrop Sign** with it. Then `Set Variable` **ProfileReq** to it.
      2. `Get Contents of URL`: URL *ProfileReq.url*, **GET**, the five headers from
         *ProfileReq*. Then `Set Variable` **Profile** to it.
      3. `Generate Hash` **SHA256** of *Profile*, then `Change Case` to **lowercase**.
         Then `Set Variable` **ProfileHash** to it.
      4. `If` *ProfileHash* **is** *Pairing.ca*:
         `Set Name` of *Profile* to `PhoneDrop.mobileconfig`, then `Save File` *Renamed Item*,
         **Ask Where to Save** off, **Destination Path**
         `PhoneDrop/PhoneDrop.mobileconfig`, **Overwrite If File Exists** on.
         `Show Alert`: "Last step: in the Files app open iCloud Drive → Shortcuts → PhoneDrop
         and tap PhoneDrop.mobileconfig. Then Settings → Profile Downloaded → Install, and
         Settings → General → About → Certificate Trust Settings → turn on PhoneDrop."
         **Otherwise**: `Show Alert`: "The certificate didn't match the QR code. Don't install
         anything; pair again on your home network."
         **End If**.
      **End If**.
4. **Otherwise**: `Show Alert` "That isn't a PhoneDrop code." **End If**.

To pair: on the PC, click the tray icon → **Pair**. On the iPhone, run **PhoneDrop Setup** and
scan the code. Then on the PC click **Allow**.

## 3. Send to PC (photos, videos, files)

Create **Send to PC**. In its settings (ⓘ) turn on **Show in Share Sheet**. In the first action,
set **Receive** to *Images, Media, Files* and **If there's no input** to **Ask For → Photos**
(or add `Select Photos` with **Select Multiple** on).

1. `Repeat with Each` item in **Shortcut Input**. Inside:
   1. `Get Name` of *Repeat Item*. Then `URL Encode` it. Then `Set Variable` **EncodedName** to it.
   2. `Generate Hash` **SHA256** of *Repeat Item*. Then `Set Variable` **BodyHash** to it.
   3. `Dictionary`: `method` = `POST`, `path` = `/v1/items`, `body` = *BodyHash*.
      `Run Shortcut` **PhoneDrop Sign** with it. Then `Set Variable` **Req** to it.
   4. `Get Contents of URL`: URL *Req.url*, **POST**, the five headers from *Req* plus `X-PD-Name` = *EncodedName*,
      **Request Body** **File** = *Repeat Item*. Then `Set Variable` **Result** to it.
   5. `Text`: *Result*. Then `If` that *Text* **does not contain** `true`: `Show Alert` "PhoneDrop: *Result*",
      `Stop This Shortcut`. **End If**.
2. **End Repeat**. Optionally `Show Notification` "Sent to PC".

> **Videos:** if a video fails with *"produced no file representation"*, add `Encode Media`
> (**Size** *Passthrough*) as the first action inside the Repeat, and use its output instead of
> *Repeat Item* in steps 2 and 4. Passthrough doesn't re-compress, so it's quick.

On the PC, photos and files are saved (default: Downloads) and a notification offers **Open** and
**Show in folder**.

## 4. Send Text to PC (text, links, clipboard)

Create **Send Text to PC**. **Show in Share Sheet** on, **Receive** *Text, URLs, Rich Text,
Safari web pages*, **If there's no input** → **Get Clipboard**.

1. `Text`: **Shortcut Input**. Then `Set Variable` **Body** to it. This turns links and pages into plain text, so
   the hash and the upload are the same bytes.
2. `Generate Hash` **SHA256** of *Body*. Then `Set Variable` **BodyHash** to it.
3. `Dictionary`: `method` = `POST`, `path` = `/v1/items`, `body` = *BodyHash*.
   `Run Shortcut` **PhoneDrop Sign** with it. Then `Set Variable` **Req** to it.
4. `Get Contents of URL`: URL *Req.url*, **POST**, the five headers from *Req* (no `X-PD-Name`), **Request Body**
   **File** = *Body*. Then `Set Variable` **Result** to it.
5. `Text`: *Result*. Then `If` that *Text* **does not contain** `true`: `Show Alert` "PhoneDrop: *Result*".
   **End If**.

The PC copies text to its clipboard (if *Copy received text* is on) and shows links with an
**Open link** button.

## 5. Send Latest Screenshot

Duplicate **Send to PC** and name it **Send Latest Screenshot**. Turn **Show in Share Sheet** off,
put `Get Latest Screenshots` (**1**) at the top, and make the Repeat loop over *Latest
Screenshots* instead of *Shortcut Input*.

### One gesture

| Trigger | Where to set it |
|---|---|
| **Back Tap** (double/triple tap the back of the phone) | Settings → Accessibility → Touch → Back Tap |
| **Action Button** (iPhone 15 Pro and later) | Settings → Action Button → Shortcut |
| **Control Center / Lock Screen** (iOS 18+) | Edit Control Center → Add a Control → Shortcuts |
| **Siri** | Say the shortcut's name |
| **Share sheet** | Share → **Send to PC** / **Send Text to PC** |

A good set: Back Tap double → **Send Latest Screenshot**, triple → **Send Text to PC** (sends
the clipboard), and a Control Center button → **Receive from PC**.

## 6. Receive from PC

On the PC you queue things for the phone in three ways: right-click files in Explorer → **Send to
→ PhoneDrop (to phone)**; the panel's **Files…** and **Clipboard** buttons; or **Ctrl+Alt+P**,
which sends the clipboard. The phone collects them with **Receive from PC**.

1. `Dictionary`: `method` = `GET`, `path` = `/v1/outbox`, `body` = `-` (one hyphen).
   `Run Shortcut` **PhoneDrop Sign** with it. Then `Set Variable` **Req** to it.
2. `Get Contents of URL`: URL *Req.url*, **GET**, the five headers from *Req*. Then `Set Variable` **List** to it.
3. `Text`: *List*. Then `If` that *Text* **does not contain** `true`: `Show Alert` "PhoneDrop: *List*",
   `Stop This Shortcut`. **End If**.
4. `Get Dictionary Value` key `items` in *List*. Then `Set Variable` **Items** to it. `Count` *Items*; if it
   **is** `0`, `Show Notification` "Nothing waiting on the PC" and `Stop This Shortcut`.
5. `Repeat with Each` item in *Items*. Inside:
   1. `Get Dictionary Value` key `kind` in *Repeat Item*. Then `Set Variable` **Kind** to it.
   2. `If` *Kind* **is** `text`: `Copy to Clipboard` *Repeat Item.text*. **End If**.
   3. `If` *Kind* **is** `url`: `If` *Repeat Item.text* **begins with** `http`:
      `Open URLs` *Repeat Item.text*. **Otherwise**: `Copy to Clipboard` it. **End If**.
      **End If**.
   4. `If` *Kind* **is** `image`, and again for `video` and `file` (three Ifs, or `Otherwise`s):
      1. `Dictionary`: `method` = `GET`, `path` = *Repeat Item.path*, `body` = `-`.
         `Run Shortcut` **PhoneDrop Sign** with it. Then `Set Variable` **FileReq** to it.
      2. `Get Contents of URL`: URL *FileReq.url*, **GET**, the five headers from *FileReq*.
         Then `Set Variable` **Download** to it.
      3. For `image` and `video`: `Save to Photo Album` *Download*. For `file`: `Save File`
         *Download* (**Ask Where to Save** on).
6. **End Repeat**. `Show Notification` "Received *Count* from the PC".

Text and links leave the PC once listed; files once downloaded. Anything not collected within 24
hours is dropped.

## 7. Encryption (optional, beta)

With **Settings → Encrypt transfers** on, the PC acts as a tiny certificate authority that may only
vouch for itself (its own `.local` name and private IP addresses). The Setup shortcut downloads
it, checks it against the hash in the QR code so nobody on the network can swap it, and saves it
for you to install. From then on the phone talks to the PC over HTTPS only.

- **Change PhoneDrop Sign** so it picks https: replace step 11's `http://` with a *Scheme*
  variable. Before step 11 add `Text` `http` → `Set Variable` **Scheme**; then
  `Get Dictionary Value` key `tls` in *Config*, and `If` it **has any value**:
  `Get Dictionary Value` key `plain` in **Shortcut Input**, `If` that **does not have any value**:
  `Text` `https` → `Set Variable` **Scheme**. **End If**, **End If**. In step 11 use *Scheme*
  followed by `://` instead of `http://`.
- Turned encryption **on** after pairing? Remove the phone in the PC's panel and pair again, so
  the new QR code carries the certificate.
- Turned it **off**? Phones paired with it on keep trying HTTPS, so pair them again too.
- **Beta:** the PC relies on iOS enforcing the certificate's name constraints, which hasn't been
  checked on a real iPhone yet. See the README.

## 8. Troubleshooting

| You see | Meaning |
|---|---|
| `unauthorized` | The PC didn't accept the signature. Check §1 steps 6–10 character by character (the `\|` separators, no spaces). Right after pairing, rerun Setup. |
| *Could not connect to the server* | The PC isn't listening: does the tray panel say *Ready to receive*? Is the phone on the same Wi-Fi? |
| *A server with the specified hostname could not be found* | `.local` names don't resolve on this network. In Files, open Shortcuts → PhoneDrop → `config.json` and replace the host in `base` with one of the `ips` (keep `:47474`). |
| `denied` | Someone clicked Deny on the PC, or nobody answered within 50 seconds. Run Setup again. |
| `body hash mismatch` | What was sent isn't what was hashed. For text and links use **Send Text to PC**; for videos see the note in §3. |
| `use https` / *certificate is invalid* | Encryption settings changed since pairing (§7). Pair again. |
| Your old pairing stopped working after a failed Setup | Setup saves the new code before the PC says yes. Run Setup again and click Allow. |

## 9. For developers

Test vector for the signature (see [PROTOCOL.md §3](PROTOCOL.md#3-signed-requests)):

| Input | Value |
|---|---|
| secret | `9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08` |
| method, path | `POST`, `/v1/items` |
| time | `2026-10-24T22:15:24+05:00` |
| nonce | `n0nce-7f3a9c21` |
| body (`hello`) | `2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824` |
| **inner** | `f795c82b4e578020b2393c09a9ca484d8e006738bf732af151ef43a93dff74ad` |
| **sig** | `936b011316a67ab2656939ab738b1786fa40ca55380faeec5111d0693056caaf` |

If *Generate Hash* prints upper-case hex, the inner hash is upper case and the signature is
`f14fe1f8e6dab993f61ab6229ca63ceaeb81a26f86a206d2b77149e51fc0c449`, which the PC also accepts.
