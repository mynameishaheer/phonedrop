# Setting up the iPhone shortcuts

PhoneDrop needs no iPhone app. It uses five shortcuts you build once in Apple's **Shortcuts** app:

| Shortcut | What it does | Build it in |
|---|---|---|
| **PhoneDrop Sign** | Helper used by the others (signs each request). You never run it yourself. | §1 |
| **PhoneDrop Setup** | Pairs the iPhone with your PC by scanning the QR code. Run once. | §2 |
| **Send to PC** | Share sheet: sends photos, videos, files, links and text. With no input it sends the clipboard. | §3 |
| **Send Latest Screenshot** | One tap (Back Tap, Action Button, Control Center) sends your newest screenshot. | §4 |
| **Receive from PC** | Fetches what the PC sent you: photos to Photos, files to Files, text to the clipboard. | §5 |

> Building takes about 15 minutes. Once you have them, you can share each one from the Shortcuts
> app as an iCloud link, so others install with one tap. Project releases will include those links.

**How to read the steps.** `Action name` is what you type in the search bar at the bottom of the
editor. **Bold** words are settings inside an action. "Variable *X*" means: long-press (or tap)
the field, choose **Select Variable**, and pick the output of the action named *X*. To rename an
action's output, tap it and choose **Rename**; the steps below name outputs so they're easy
to pick.

Before you start: in **Settings → Apps → Shortcuts → Advanced**, turn on **Allow Running
Scripts** and **Allow Sharing Large Amounts of Data**.

---

## 1. PhoneDrop Sign (helper)

Create a new shortcut named exactly **PhoneDrop Sign**. It reads the pairing file, then returns
the URL and headers for one request.

1. `Get File from Folder`: folder **Shortcuts**, path `PhoneDrop/config.json`, **Error If Not
   Found** on.
2. `Get Dictionary from Input` (input: *File* from step 1). Rename the output **Config**.
3. `Format Date`: date **Current Date**, **Date Format** = *ISO 8601*, **Include ISO 8601 Time**
   on. Rename **Time**.
4. `Random Number` from `100000000` to `999999999`. Rename **R1**.
5. `Random Number` from `100000000` to `999999999`. Rename **R2**.
6. `Text`: `R1` `R2` (the two variables next to each other, no space). Rename **Nonce**.
7. `Text`, all on one line, with `|` between items:

   `Config.secret`|`Shortcut Input.method`|`Shortcut Input.path`|`Time`|`Nonce`|`Shortcut Input.body`

   For `Config.secret`: insert variable *Config*, tap it, choose the key `secret`. For
   `Shortcut Input.method`: insert **Shortcut Input**, tap it, choose **Dictionary**, key
   `method`. Same for `path` and `body`.
8. `Generate Hash`: **SHA256** of the *Text* from step 7. Rename **Inner**.
9. `Text`: `Config.secret`|`Inner` (one line, one `|`).
10. `Generate Hash`: **SHA256** of the *Text* from step 9. Rename **Sig**.
11. `Text`: `http://` `Config.host` `:` `Config.port` `Shortcut Input.path`. Rename **URL**.
12. `Dictionary` with these Text keys:

    | Key | Value |
    |---|---|
    | `url` | *URL* |
    | `phone` | *Config.id* |
    | `time` | *Time* |
    | `nonce` | *Nonce* |
    | `body` | *Shortcut Input.body* |
    | `sig` | *Sig* |

13. `Stop and Output`: the *Dictionary*.

Open the shortcut's settings (ⓘ) and make sure **Show in Share Sheet** is off.

> **Check:** the PC's signature test vector is in [§7](#7-for-developers). Feed the same values
> through steps 7–10 and you must get the same `sig`.

## 2. PhoneDrop Setup

Create **PhoneDrop Setup**.

1. `Scan QR or Barcode`. Rename **QR**.
2. `Get Dictionary from Input` (input: *QR*). Rename **Pairing**.
3. `If` *Pairing.pd* **is** `1`. Inside the If:
   1. `Save File`: input *QR*, **Ask Where to Save** off, **Destination Path**
      `PhoneDrop/config.json`, **Overwrite If File Exists** on.
   2. `Get Device Details`: **Device Name**. Rename **Name**.
   3. `Generate Hash`: **SHA256** of *Name*. Rename **BodyHash**.
   4. `Dictionary`: `method` = `POST`, `path` = `/v1/pair`, `body` = *BodyHash*.
   5. `Run Shortcut`: **PhoneDrop Sign**, input: the *Dictionary*, **Show While Running** off.
      Rename **Req**.
   6. `Get Contents of URL`: URL *Req.url*. **Method** POST. **Headers**:

      | Header | Value |
      |---|---|
      | `X-PD-Phone` | *Req.phone* |
      | `X-PD-Time` | *Req.time* |
      | `X-PD-Nonce` | *Req.nonce* |
      | `X-PD-Body` | *Req.body* |
      | `X-PD-Sig` | *Req.sig* |

      **Request Body**: **File**, choose *Name*. Rename **Result**.
   7. `Get Dictionary from Input` (input: *Result*). Rename **Reply**.
   8. `If` *Result* **contains** `"ok":true` (type it exactly, with the quotes):
      `Show Alert`: "Paired with *Reply.laptop*. You're ready to send."
      **Otherwise**: `Show Alert`: "Pairing didn't work: *Result*".
      **End If**
4. **Otherwise** (the outer If): `Show Alert`: "That isn't a PhoneDrop code."
5. **End If**

To pair: on the PC, click the tray icon → **Pair**. On the iPhone, run **PhoneDrop Setup** and
scan the code. The PC asks *Allow "your iPhone" to send to this PC?*: click **Allow**.

## 3. Send to PC

*Coming with milestone 3.*

## 4. Send Latest Screenshot

*Coming with milestone 3.*

## 5. Receive from PC

*Coming with milestone 4.*

## 6. Troubleshooting

| You see | Meaning |
|---|---|
| `unauthorized` | The PC didn't accept the signature. Check §1 step 7 character by character (the `\|` separators, no spaces). If it happens right after pairing, rerun Setup. |
| *Could not connect to the server* | The PC isn't listening: is the tray panel saying *Ready to receive*? Is the phone on the same Wi-Fi? |
| *A server with the specified hostname could not be found* | `.local` names don't resolve on this network. Open `config.json` in Files (Shortcuts folder → PhoneDrop) and replace the `host` value with one of the `ips`. |
| `denied` | Someone clicked Deny on the PC, or nobody answered within 50 seconds. Run Setup again. |

## 7. For developers

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
`f14fe1f8e6dab993f61ab6229ca63ceaeb81a26f86a206d2b77149e51fc0c449`, which the PC accepts too.
