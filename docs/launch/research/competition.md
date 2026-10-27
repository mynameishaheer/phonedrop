# PhoneDrop market research (as of 2026-10-25)

## Method and limits (read first)
- Tools: WebSearch (summarising search tool) + WebFetch. The sandbox egress allowlist blocked reddit.com (both tools), news.ycombinator.com, hn.algolia.com, learn.microsoft.com, support.microsoft.com, alternativeto.net, localsend.org, apps.apple.com/itunes API. github.com, Vercel's domain-registry lookup and WebSearch worked.
- Consequences: **no Reddit threads could be read** (demand evidence below is HN titles/snippets, Microsoft Q&A titles, GitHub issues, press). HN comment text is only what the search tool surfaced (first line of each comment); HN dates are inferred from item-ID ranges and flagged "approx". Star counts come from WebFetch of GitHub HTML (could be cached; treat +/-5%).
- Nothing about trademarks could be confirmed via USPTO/EUIPO (not reachable). No store (App Store / Microsoft Store) search was possible directly; only web-search inference.

---
## 1. Competitive landscape

| Tool | Price | iPhone app needed? | Setup friction | Key limits | Recency / source |
|---|---|---|---|---|---|
| **Microsoft Phone Link + "Link to Windows" (iOS)** | Free | Yes (Link to Windows, App Store) | Pair via Bluetooth/QR, needs recent Phone Link on Win 10/11, iOS 16+ | Historically no photo browsing/ no file transfer; file sharing announced Dec 2024 as Insider-only, **GA status still unclear** (Learn Q&A copy "updated 2026" still says gradual rollout / Insider; one guide ~78 days old says Insider no longer required). Users report "File failed to send" iPhone->PC while PC->iPhone works; Local Network permission + keep app foreground. Local files only (OneDrive files can't be shared). Slower than AirDrop (Tom's Guide hands-on). No group msgs/media in messaging. | Dec 2024 announcement: https://blogs.windows.com/windows-insider/2024/12/11/sharing-files-between-your-iphone-and-windows-pc-rolling-out-to-windows-insiders/ ; Q&A copy (2026 edit): https://learn.microsoft.com/en-us/answers/questions/2338274/article-sharing-your-files-between-your-iphone-and ; failure threads: https://learn.microsoft.com/en-in/answers/questions/5899713/file-transfer-using-from-iphone-to-pc-fails-works , https://learn.microsoft.com/en-sg/answers/questions/5941182/unable-to-send-files-from-iphone-to-pc ; Tom's Guide: https://www.tomsguide.com/computing/windows-operating-systems/airdrop-for-windows-is-finally-here-microsoft-announces-new-way-to-easily-share-files-between-your-iphone-and-pc ; Wikipedia lists Phone Link Win 1.26072.255 (11 Sep 2026), iOS app 1.26082.6 (8 Sep 2026): https://en.wikipedia.org/wiki/Phone_Link |
| **Intel Unison** | Was free | Yes | n/a | **Dead.** Intel ended service for most platforms 30 Jun 2025 (store removal ~1 Jul 2025); Lenovo Aura kept it through 2025; Intel's own site says end of support 1 Jan 2026. Unison was the only credible iOS bridge pre-Phone Link; its users are orphaned. | https://9to5google.com/2025/04/03/intel-unison-app-shutting-down/ , https://www.pcworld.com/article/2836520/intels-excellent-unison-pc-to-phone-app-shuts-down-for-good.html , https://unison.intel.com (Jan 2026 date, via search summary) |
| **LocalSend** | Free, Apache-2.0 | **Yes, on both ends** (iOS App Store) | Install on each device; same LAN | GitHub ~95.5k stars, 5.4k forks (WebFetch, Oct 2026); site claims 93k+ stars / 35M+ downloads (self-reported). Distribution: winget, Scoop, Chocolatey, EXE, portable ZIP, Homebrew, App Store, Play, F-Droid, Flathub, Snap, AUR... (no Microsoft Store; no auto-update on Windows). iOS limits: app must be foreground; device vanishes from list when backgrounded (issue #1468, open since 9 Jul 2024); iOS Local Network permission/ firewall rule pitfalls (#1156). HN users: large batches/files can fail, no resume (approx 2025-26 threads). A lower-quality review says single maintainer; one listing says v1.18.2 on 2026-08-21 (unverified). | https://github.com/localsend/localsend ; https://github.com/localsend/localsend/issues/1468 ; https://github.com/localsend/localsend/issues/1156 ; https://www.pcgamer.com/software/localsend-is-the-most-useful-app-ive-come-across-in-a-very-long-time-letting-me-share-files-across-devices-with-a-couple-clicks/ |
| **PairDrop / Snapdrop** | Free, open source (licence not checked), self-hostable | No (browser/PWA) | Open web page on both devices, same network (or pair via code over WebRTC/TURN) | PairDrop ~11.7k stars (WebFetch). Snapdrop was **acquired by LimeWire (11 Feb 2025 per PitchBook)**; site now redirects/ads; community points to PairDrop. Browser transfers: no share-sheet send for arbitrary apps, no background, no clipboard->PC integration. | https://github.com/schlagmichdoch/PairDrop ; https://github.com/RobinLinus/snapdrop ; https://pitchbook.com/profiles/company/752589-55 |
| **KDE Connect (+ iOS)** | Free, OSS | Yes (official iOS app on App Store, v0.4.3 Jan 2025 per Wikipedia - no newer found) | Install on both, pair, same LAN; Windows client = KDE Connect (Microsoft Store/ installer) | iOS is feature-limited (no notification sync; foreground-oriented; clipboard PC->iPhone works but not reverse per Dec 2025 KDE Discuss report). Repo ~426 stars. Vendor blogs say it freezes with many files (weak source). | https://en.wikipedia.org/wiki/KDE_Connect ; https://github.com/KDE/kdeconnect-ios ; https://itsfoss.com/news/kde-connect-ios/ |
| **Google Quick Share + AirDrop interop** | Free | n/a (it's Android<->iPhone) | iPhone must be "Everyone for 10 minutes" | Launched 20 Nov 2025 on Pixel 10; expanded to Galaxy S26/S25/S24 etc. (One UI 8.5) and announced for OnePlus/Oppo/Vivo/Xiaomi/Honor (May-Jun 2026). Android-only; **no Windows<->iPhone path found**. HN: Windows has no Wi-Fi Aware stack, so no obvious port. | https://www.androidauthority.com/quick-share-works-with-airdrop-3618023 ; https://9to5google.com/2026/05/12/google-bringing-airdrop-support-most-android-phones-later-this-year/ ; https://www.macrumors.com/2026/05/13/every-android-phone-getting-airdrop-support/ ; HN https://news.ycombinator.com/item?id=45994854 (Nov 2025) |
| **Apple's own moves** | - | - | - | No AirDrop for Windows. EU DMA: Apple added Wi-Fi Aware in iOS 26 (global). **Microsoft filed a DMA interoperability request 25 Mar 2026 for iPhone<->Windows clipboard sync; Apple approved plan 26 Jun 2026; engineering done ~fall 2027, public ~early 2028, EU-only at first** (pair PC as accessory via AccessorySetupKit). Third-party AirDrop-alternatives must get share-sheet parity in EU by end-2026 (per heise). iCloud for Windows remains the sanctioned photo path. No Apple Windows file-sharing announcement found. | https://www.ghacks.net/2026/08/10/apple-confirms-iphone-to-windows-clipboard-sync-after-microsofts-eu-interoperability-request/ ; https://www.techrepublic.com/article/news-apple-iphone-windows-clipboard-sharing/ ; https://heise.de/-10446649 |
| **Shortcuts-based OSS (direct analogues)** | Free, MIT | **No** | Python/Node server + hand-built or imported Shortcut | **AirDropPlus** (yeyt97): ~491 stars, MIT, Python, files+clipboard both ways, ~8 manual setup steps. **Orange Share**: ~60 stars, MIT, Python, Windows installer+tray, README warns Shortcuts flakiness, plain HTTP + Basic Auth. **iOS-ShareEasy**: 14 stars. **ios-shortcuts-files-to-pc**/bariseksi: C# listener, stops when screen locks. Last-update ages (AirDropPlus ~673 days, ShareEasy ~376 days) are from search summary, undated. | https://github.com/yeyt97/AirDropPlus ; https://github.com/Yannis4444/Orange-Share ; https://github.com/spectreDeveloper/iOS-ShareEasy ; https://github.com/drshajul/ios-shortcuts-files-to-pc |
| **WinSend** | Freemium, ~$13 one-time (AlternativeTo listing, vendor copy) | Yes (iPhone app, but triggerable via Shortcuts/Back Tap) | Same Wi-Fi, no account | Clipboard + photos. Only seen via AlternativeTo (not independently verified). | https://alternativeto.net/software/winsend/ |
| **Cloud/other** | iCloud 5 GB free; SHAREit/Xender/Send Anywhere/AirDroid freemium, ads | mostly yes | - | Spammy SEO "AirDrop for Windows" guides (AOMEI, Wondershare, EaseUS, CopyTrans Filey) dominate search results - vendor-driven, not independent. | https://www.copytrans.net/blog/airdrop-to-pc/ |

### NEW and critical: same-name, near-same-concept project, shipped days ago
- **ImSammyTTV/PhoneDrop** - "AirDrop-style tray app to send files between an iPhone or iPad and a Windows PC over Wi-Fi". MIT, Electron, repo created **10 Oct 2026**, v2.0.0 (Mac + clipboard + Tailscale + AES-256 passphrase E2E) and v2.1.0 on **23 Oct 2026**. iPhone side = Safari/home-screen web app (no native app, **no Shortcuts**), manual cert trust step. 0 stars at fetch time. Sources: https://github.com/ImSammyTTV/PhoneDrop , https://github.com/ImSammyTTV/PhoneDrop/releases , https://github.com/ImSammyTTV/PhoneDrop/commits/main
- Also **dhairyakhetan/PhoneDrop** (Windows app, QR, phone browser, Node bundled; Android+iPhone): https://github.com/dhairyakhetan/PhoneDrop
- Takeaway: independent developers are converging on this exact idea in the same fortnight - a demand signal, but also a discoverability and branding problem (see section 3).

### PhoneDrop honest positioning
"A tiny, no-account, no-iPhone-app receiver for people who already use Shortcuts/Back Tap/Action Button, with pairing and request signing done right." It is a **power-user / tinkerer tool today**, not a mainstream AirDrop replacement. Mainstream users will pick LocalSend (95k stars, app-store install) or whatever Phone Link finally ships.

### 3 sharpest differentiators
1. **Zero install on the phone and native triggers**: share sheet, Back Tap, Action Button, Control Center, plus one-tap "send latest screenshot". LocalSend, KDE Connect, Phone Link and WinSend all need an iOS app; LocalSend's own iOS pain (#1468) is the app-must-be-foreground problem a Shortcuts push model avoids. (Verify: how does PC->iPhone work? iOS cannot accept inbound connections without a resident app, so presumably the phone pulls via a Shortcut. State that plainly in the README.)
2. **Footprint and trust**: single ~7 MB Go exe, no runtime, zero idle CPU, vs Electron (ImSammyTTV), Python (AirDropPlus, Orange Share), Node (dhairyakhetan). Pair-once QR, signed requests, 401 to strangers, trusted-networks-only, vs Orange Share's plain HTTP/Basic Auth and ImSammyTTV's manual self-signed cert install.
3. **Right-sized scope**: text/links straight to the Windows clipboard, screenshots, both directions, no cloud/account/Microsoft login - a gap left by Unison's death and Phone Link's still-unclear, Insider-gated and flaky iPhone transfer.

### Weaknesses (be upfront)
- **Setup**: README says ~20 minutes to hand-build Shortcuts (docs/SHORTCUTS.md). Competitors import via QR/iCloud link (Orange Share, AirDropPlus). This is the biggest adoption blocker; shipping importable .shortcut files (iCloud share links) is the top fix.
- **Same Wi-Fi only** (HN complaint #1 about every LAN tool; AirDrop wins ad hoc/ stranger sharing). ImSammyTTV already offers Tailscale remote.
- **Windows-only**; competitors cover Mac/Linux.
- Unsigned exe (SmartScreen warning), 0 stars, brand-new, no store listing (LocalSend also lacks Microsoft Store but has winget/Scoop/Choco - add those).
- Shortcuts fragility (Orange Share README documents iOS Shortcuts regressions; the iOS "Allow Untrusted Shortcuts" toggle step).
- Phone Link could become "good enough" if GA arrives; Apple/Microsoft clipboard sync (EU first, ~2028) would erode the clipboard feature.
- Large-file reliability on iOS Shortcuts (screen lock stops transfers in a comparable project): test and document.

---
## 2. Demand signals
Reddit was inaccessible; the evidence below is HN (item IDs given, dates approximate from ID ranges), Microsoft Q&A, GitHub issues, press.

| # | Pain point (quote = verbatim snippet surfaced by search) | Link | Date |
|---|---|---|---|
| 1 | "My problem is that all these alternatives require the devices to be on the same ..." (LAN-only objection to LocalSend/PairDrop/KDE Connect; replies: AirDrop's value is ad hoc/ strangers) | https://news.ycombinator.com/item?id=47933813 (thread: "Localsend: An open-source cross-platform alternative to AirDrop", https://news.ycombinator.com/item?id=47933208) | 2026, approx |
| 2 | "I love LocalSend for quick transfers between your own devices, just werks on eve..." (iPhone/Windows user; notes both ends need the app + same network) | https://news.ycombinator.com/item?id=44705067 | mid-2025, approx |
| 3 | Pixel-10/AirDrop thread: commenter notes Windows "hasn't deployed Wi-Fi Aware in Windows. With no public plans to do so" -> Windows is the gap in the AirDrop-interop story (paraphrase/ snippet) | https://news.ycombinator.com/item?id=45994854 | 20-21 Nov 2025 |
| 4 | "Ask HN: How to export photos from iPhone to non Mac computers" | https://news.ycombinator.com/item?id=41619536 | Sep 2024, approx |
| 5 | "I hope ability to get photos off iPhones is improved. I can NEVER get it to full..." | https://news.ycombinator.com/item?id=39303002 | Feb 2024, approx |
| 6 | "I am a person outside of apple ecosystem that has to use iPhone and occasionally..." | https://news.ycombinator.com/item?id=38804922 | Jan 2024, approx |
| 7 | Windows import "corrupted" photos after import-and-delete; "Apple corrupted images on my iPhone where I can't import them to my PC via photo..." | https://news.ycombinator.com/item?id=45274805 | Sep 2025, approx |
| 8 | MS Q&A title: "File transfer using from iPhone to PC fails, works PC to iPhone." and "Unable to send files from iPhone to PC." (error "File failed to send"; answers: Local Network permission, keep app foreground) | https://learn.microsoft.com/en-in/answers/questions/5899713/file-transfer-using-from-iphone-to-pc-fails-works ; https://learn.microsoft.com/en-sg/answers/questions/5941182/unable-to-send-files-from-iphone-to-pc | likely 2026 (inferred from question IDs; page dates not readable) |
| 9 | LocalSend #1156 "iOS can't find Windows though Windows can find iOS and transmit." ; #1468 "[iOS/iPadOS] Run in background" (open since 9 Jul 2024, 11 comments) | https://github.com/localsend/localsend/issues/1156 ; https://github.com/localsend/localsend/issues/1468 | 2024- |
| 10 | Tom's Guide headline: "I paired my iPhone 16 with a Windows PC - and Microsoft Phone Link can't compete with a MacBook" | https://www.tomsguide.com/computing/i-paired-my-iphone-16-with-a-windows-pc-and-microsoft-phone-link-cant-compete-with-a-macbook | undated (iPhone 16 era, late 2024-2025) |

Size/recurrence estimate (qualitative, no search-volume tool available):
- Evergreen: "AirDrop for Windows" is a perennial query; page-one results in 2026 are all SEO vendor guides (updated July/Aug 2026), indicating sustained search demand and low-quality supply.
- Proof of appetite: LocalSend ~95k stars/ claimed 35M downloads; PairDrop ~12k; AirDropPlus ~490 stars for an ugly 8-step Shortcuts install; at least 2 new same-concept "PhoneDrop" repos in Oct 2026.
- Active supply-side vacuum: Unison dead (2025), Snapdrop sold to LimeWire (Feb 2025), Phone Link iPhone file sharing unreliable/Insider-labelled, Quick Share-AirDrop skips Windows, Apple clipboard sync EU-only ~2028.
- Realistic TAM for PhoneDrop as designed: small-to-mid niche (Windows desktop + iPhone + comfortable with Shortcuts). Hundreds to low thousands of stars plausible with a good demo and importable shortcuts; mass adoption unlikely without a native app.

---
## 3. Name check: "PhoneDrop"

### Findings
| Check | Result | Confidence |
|---|---|---|
| GitHub | `github.com/search?q=phonedrop` shows **22 repos**; at least 7 are same/similar-concept: ImSammyTTV/PhoneDrop (iPhone<->Windows, near-identical pitch, v2.1.0 on 23 Oct 2026), dhairyakhetan/PhoneDrop (Windows <-> Android/iPhone), willmanduran/phonedrop (22 stars, QR over LAN), osbendes/phonedrop (Android->Mac AirDrop-style), Metamorfolog/phonedrop (Windows->Android), subsite/phonedrop, caffeinepub/phonedrop | high (WebFetch of GitHub) |
| Chrome Web Store | "PhoneDrop - Send to Phone via QR" extension exists | medium (search result) |
| App Store / Microsoft Store | No listing found by web search; **not directly verifiable** (store sites blocked) | low |
| Domains (Vercel registry lookup, 25 Oct 2026) | phonedrop.app, .dev, .io, getphonedrop.com **available** (9.99 yr1; renew 15 / 13 / 46 for .io); phonedrop.com, .net, .org **taken** (owners/ usage not checked) | high for availability flag |
| Trademark | No USPTO record for PHONEDROP surfaced; USPTO/EUIPO/WIPO not reachable so **not verified**. Neighbours seen: PODDROP (reg. 6925853), PHONE DRS (Class 11). | low |
| Apple sensitivity | Apple lists **AirDrop** and **NameDrop** as its trademarks (https://www.apple.com/legal/trademark/appletmlist.html). "PhoneDrop" is not "AirDrop", but "[Noun]Drop" for a share feature sits in the same naming family as AirDrop/NameDrop. The risk is mostly in marketing copy: never use "AirDrop" in the name, domain, app/store title or logo; use "AirDrop-style"/"like AirDrop" descriptively and with attribution. Not legal advice. | medium |
| Microsoft | Edge has a feature called "Drop" (file/notes sync) - minor descriptive overlap | medium |

### Risk assessment
- **Legal risk: low-moderate (unverified)**. **Practical/brand risk: high.** An identical name already ships near-identical functionality (Windows tray app + iPhone, MIT, 2 weeks old, Electron). Users, search and GitHub topics will conflate them; whoever gets traction first owns "PhoneDrop". Because ImSammyTTV has live releases and PhoneDrop-the-name is spread over 22 repos, **renaming before launch is wise** (cost is near zero today: 0 stars, 0 users).

### 5 alternative names (domain check via Vercel registry, 25 Oct 2026; GitHub name-search via WebFetch)
| Name | .app | .dev | .io | .com | GitHub search | Notes |
|---|---|---|---|---|---|---|
| **TossPad** | free | free | free | not checked | 0 repos | Short, verb-y; no "Drop"/Air; best clean sweep |
| **SlingPC** | free | free | - | **free** | 0 repos | States platform; .com available (limits future Mac port) |
| **HopSend** | free | free | free | not checked | 1 unrelated repo (leo1394/hopsend, 0 stars) | Descriptive, low collision |
| **PocketBeam** | free | free | free | taken | 3 repos; moneysiva1/PocketBeam is a phone->Android-TV sender (1 star) = adjacent concept | Nice but crowded-ish; "Beam" has Android Beam baggage |
| **PocketPipe** | free | free | free | not checked | 2 unrelated repos (Telegram link saver; bioinformatics) | Metaphor fits Shortcuts "pipe to PC" |
Also checked/rejected: SendLane (.app/.dev taken), BeamHome (.io taken). Domain "available" = registry flag only; no premium check, no trademark search, no App Store/Microsoft Store name search done for any alternative.

Recommendation: pick one of TossPad / HopSend / PocketPipe, grab .app + .dev (~$20/yr), run a real USPTO TESS/TSDR + EUIPO search and App Store/Microsoft Store name search (not possible in this sandbox) before announcing. Keep "AirDrop-style" only as a descriptive tagline.

---
## Bottom line
1. Concept is validated (persistent demand, vacated incumbents, copycats), but the product is a niche power-user tool and the manual 20-min Shortcuts setup is the main barrier - ship importable shortcuts first.
2. Do not lead with "AirDrop for Windows" vs LocalSend; lead with "no phone app, one tap from the share sheet, 7 MB, signed + private".
3. Rename now: PhoneDrop collides with ImSammyTTV/PhoneDrop (Oct 2026) and ~20 other repos.
