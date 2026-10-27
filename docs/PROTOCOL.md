# PhoneDrop protocol (v1)

The iPhone side is Apple Shortcuts, so every rule here is chosen to be doable with stock actions:
*Generate Hash* (SHA-256, hex), *Text* (concatenation), *Get Contents of URL* (method, headers,
raw file body), *Format Date* (ISO 8601) and *Random Number*. Shortcuts has no HMAC, no
encryption, no try/catch, and doesn't throw on HTTP errors, so responses are JSON the shortcut can
read.

All values below are case-sensitive unless noted. "hex" means lowercase hexadecimal.

## 1. Transport

- The laptop listens on TCP port `47474` (configurable) on its IPv4 addresses on networks Windows
  classifies as **Private** or **Domain**. Public networks are skipped unless the user
  chose **Use here** on that network in the panel. That choice is remembered by the network's
  Windows id (not its name), and PhoneDrop adds a Public-profile firewall rule for itself once,
  with the user's UAC consent.
- Peers outside private ranges (RFC 1918, link-local, loopback, IPv6 ULA) are dropped at accept,
  as are connections beyond 32 open per peer IP or 256 in all.
- The phone addresses the laptop as `http://<hostname>.local:47474` (mDNS). The QR code also
  carries the current IPs as a fallback.
- Optional TLS (§7): the same port also speaks HTTPS; the server tells them apart by the first
  byte of the connection.

## 2. Pairing QR code

The tray panel's **Pair** button opens a 2-minute pairing window and shows a QR code containing
this JSON (UTF-8, byte mode):

```json
{"pd":1,"name":"DESKTOP-AB12CD","host":"desktop-ab12cd.local","ips":["192.168.1.20"],
 "port":47474,"base":"desktop-ab12cd.local:47474","id":"9f2c1a7e5b3d4c60","secret":"<64 hex chars>"}
```

| Field | Meaning |
|---|---|
| `pd` | protocol version, `1` |
| `name` | the laptop's display name (its computer name) |
| `host` | mDNS name, lower case |
| `ips` | current private IPv4 addresses, fallback only |
| `port` | listening port |
| `base` | `"<host>:<port>"` as a string, so a shortcut builds URLs without formatting a number (which Shortcuts may group by locale, `47,474`) |
| `id` | the **phone id** this QR code creates: 16 hex chars (64 random bits) |
| `secret` | the **phone secret**: 64 hex chars (256 random bits) |
| `tls` | only while encryption is on (§7): `true` |
| `ca` | only with `tls`: hex SHA-256 of the exact bytes of `GET /v1/ca.mobileconfig` |

The secret travels only inside the QR code, never over the network. Each QR code is single use:
the pending pairing is discarded after one pairing attempt (allowed, denied or timed out), when
the window closes, or when a new QR code is shown.

## 3. Signed requests

Every request carries these headers:

| Header | Value |
|---|---|
| `X-PD-Phone` | phone id |
| `X-PD-Time` | current time, ISO 8601 with offset (`2026-10-24T22:15:24+05:00`) **or** Unix seconds |
| `X-PD-Nonce` | random, 8–128 chars from `[A-Za-z0-9._:-]`, never reused |
| `X-PD-Body` | hex SHA-256 of the exact request body, or `-` for a request with no body |
| `X-PD-Sig` | the signature below |

```
inner = hex( SHA256( secret "|" METHOD "|" PATH "|" TIME "|" NONCE "|" BODY ) )
sig   = hex( SHA256( secret "|" inner ) )
```

- `secret` is the 64-char hex string itself, hashed as text (UTF-8).
- `METHOD` is upper case (`GET`, `POST`). `PATH` is the URL path without query, e.g. `/v1/items`.
- `TIME`, `NONCE` and `BODY` are the header values exactly as sent.
- The nested hash keeps it safe from length extension without needing HMAC.
- The signature doesn't cover `X-PD-Name` or `X-PD-Kind`. Without TLS (§7), someone on the
  network can rename an upload or change its kind, but never change its content.

The laptop accepts the request only if all of these hold:

1. `X-PD-Phone` is a paired phone (or, for `/v1/pair` only, the pending pairing's id).
2. `X-PD-Time` is within ±90 s of the laptop's clock.
3. `X-PD-Nonce` is well-formed and hasn't been seen for this phone in the last 5 minutes.
4. `X-PD-Sig` matches, compared in constant time. Hex case of `X-PD-Sig` is ignored, and `inner`
   may be upper- or lower-case hex (some Shortcuts versions print upper case).
5. While the body streams in, its SHA-256 matches `X-PD-Body` (case ignored). Bodies are written
   to a temporary file and only kept if the hash matches. `-` requires an empty body.

Failures 1–4 get **the same response for everything**: `401` with body `unauthorized`, no UI, and
the IP counts towards a penalty (more than 10 failures a minute: its connections are dropped for
5 minutes). While TLS is on, a connection that sends nothing for 10 s counts as a failure too. A
request that passes 1–4 is from a paired phone, so it gets a useful JSON error.

## 4. Responses

Authenticated responses are JSON with `Content-Type: application/json`:

```json
{"ok": true, ...}
{"ok": false, "error": "body hash mismatch"}
```

A shortcut can test for success with *If Result contains `true`*, which survives Smart
Punctuation turning `"` into curly quotes: every JSON success contains `"ok":true`, and no error
body (the JSON errors or the `401` `unauthorized`) ever contains the text `true`.

## 5. Endpoints

### `POST /v1/pair`: finish pairing

Signed with the pending pairing's id and secret from the QR code. Body (≤ 4 KB): the phone's
name as plain UTF-8 text (`Shaheer's iPhone`), or JSON `{"name": "Shaheer's iPhone"}` if it
starts with `{`. Plain text is what the Setup shortcut sends, since building JSON by
concatenation breaks on names with quotes.

The laptop shows **Allow "Shaheer's iPhone" to send to this PC?** with Allow / Deny and holds
the request open until the user answers (at most 50 s).

- Allowed: `200 {"ok":true,"laptop":"DESKTOP-AB12CD"}`. The phone is now paired; it keeps the id
  and secret from the QR code. The laptop stores the secret DPAPI-protected.
- Denied or no answer: `403 {"ok":false,"error":"denied"}`.

### `GET /v1/ping`

`200 {"ok":true,"laptop":"DESKTOP-AB12CD","version":"0.x.y","tls":false}`. Used to test a setup.

### `POST /v1/items`: phone → laptop

One item per request; the body is the raw item.

| Header | Value |
|---|---|
| `X-PD-Name` | optional. File name, percent-encoded (Shortcuts' *URL Encode*). Omit for text and links. |
| `X-PD-Kind` | optional. `file`, `text` or `url`. Inferred when absent (below). |

Inference when `X-PD-Kind` is absent, chosen so a shortcut can send every share-sheet item the
same way (Shortcuts names a text item something like `Text`, with no extension):

1. A name with a file extension: a `file`.
2. Otherwise a body of at most 1 MB that is valid UTF-8: `text`, or a `url` if the whole body
   is a single `http(s)://` link.
3. Otherwise a `file`. A missing extension comes from the content's magic bytes (`IMG_2041` →
   `IMG_2041.heic`), and a missing name becomes `Received 2026-10-24 22.15.24.<ext>`.

- Files are saved to the chosen folder (default Downloads) with a sanitized, de-duplicated name
  and the Mark-of-the-Web (`Zone.Identifier`, `ZoneId=3`). A name without an extension gets one
  from the content's magic bytes when recognisable.
- Text is copied to the clipboard if the *auto-copy* setting is on; links are offered with an
  Open button and never opened automatically.
- `200 {"ok":true,"kind":"file","saved":"IMG_2041.HEIC"}`. Errors: `413` too large, `400` body
  hash mismatch.

### `GET /v1/outbox`: laptop → phone, list

`200 {"ok":true,"items":[...]}`, oldest first. Each item:

```json
{"id":"a1b2c3d4","kind":"image","name":"Screenshot.png","size":123456,"path":"/v1/outbox/a1b2c3d4"}
{"id":"e5f6a7b8","kind":"text","text":"hello"}
{"id":"c9d0e1f2","kind":"url","text":"https://example.com"}
```

`kind` is `image`, `video`, `file`, `text` or `url`. `image` and `video` are only used for
formats the Photos app imports (jpg, jpeg, png, heic, heif, gif, webp, tif, tiff, bmp; mov, mp4,
m4v, 3gp), so a shortcut can save them to the photo library; other media (mkv, webm, avi, avif...)
is a `file`. Text and link items carry their content inline and are removed from the outbox once
listed. File items stay until downloaded.

### `GET /v1/outbox/{id}`: download one item

The raw bytes, with `Content-Type` and `Content-Disposition: attachment; filename*=UTF-8''…`.
Signed with `X-PD-Body: -`. After the whole body has been sent, the item leaves the outbox.

Items expire 24 hours after being added.

## 6. Phone-side storage

The Setup shortcut saves the QR code's JSON to `iCloud Drive/Shortcuts/PhoneDrop/config.json`,
and every other shortcut reads it from there. Shortcuts has no keychain, so any shortcut can read
it, and iCloud Drive syncs it to the user's other devices and to iCloud.com. The secret is per
phone and revocable from the laptop. On its own, and only from the same LAN, it lets the holder
send files to this laptop and download the outbox (queued files and clipboard text).

## 7. Optional TLS (beta)

When the user turns on **Encrypt transfers** on the laptop:

- The laptop creates a private CA (ECDSA P-256, its key DPAPI-protected) limited to server
  certificates (extended key usage `serverAuth`) with **X.509 name constraints**: it may only
  certify `<host>` (the `.local` name) and the private IP ranges (10/8, 172.16/12, 192.168/16,
  169.254/16, fc00::/7, fe80::/10), and no email address or URI (both are constrained to the
  reserved `invalid` domain). A CA made without these is replaced. A leaf certificate for
  `<host>` and the current IPs inside those ranges is re-issued whenever the IPs change. TLS 1.2
  or later.
- `GET /v1/ca.mobileconfig`, signed by a paired phone like any request (`X-PD-Body: -`), returns
  the CA as an iOS configuration profile (`application/x-apple-aspen-config`), or
  `404 {"ok":false,"error":"encryption is off"}`. There is no unauthenticated endpoint. The
  profile is deterministic for a given CA and laptop name, so the phone checks the SHA-256 of the
  downloaded bytes against the QR code's `ca` (§2) before opening it; that catches a profile
  swapped in transit, since pairing and this download may be plain HTTP. The user then installs
  it and turns on full trust (Settings → General → About → Certificate Trust Settings).
- The phone's config then uses `https://`. Once a phone has made one signed HTTPS request, the
  laptop refuses that phone's plain-HTTP requests (`403 {"ok":false,"error":"use https"}`) while
  encryption is on. The profile download is exempt, so a phone can fetch a new CA (after a reset
  or a rename) over plain HTTP.
- Turning encryption off accepts plain HTTP from every phone again, but phones paired while it
  was on still use `https://` (their config says `"tls":true`) and fail until they are paired
  again.

Name constraints are what make installing the CA safe: without them, a stolen CA key could
impersonate any website to that phone. **Check on a real iPhone that iOS enforces them before
relying on this mode** (see [SECURITY-MODEL.md](SECURITY-MODEL.md)).
