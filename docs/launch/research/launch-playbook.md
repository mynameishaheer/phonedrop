# PhoneDrop launch and growth research

*Prepared 2026-10-25 for a solo maintainer with no existing audience. Repo state at time of writing: created 2026-10-24, 0 stars, v1.0 and one-tap iCloud shortcut links still on the roadmap (`docs/ROADMAP.md`), so the iPhone side currently takes about 20 minutes to build by hand (`docs/SHORTCUTS.md`).*

**Method and caveats.** Everything here comes from web search. Direct page fetches (news.ycombinator.com, reddit.com, arxiv.org and most mirrors) were blocked from this environment, so several subreddit rules could not be read from the sidebars themselves. Where a claim rests on a third-party summary or a vendor blog, it is labelled as such. Treat every subreddit rule below as "verify in the sidebar on the day you post".

---

## 0. The short version

1. **Don't launch until the iPhone side installs in under 2 minutes.** For every comparable tool, the funnel is: see the demo, then try it within 5 minutes. A 20-minute hand-built shortcut setup will kill conversion on HN, Reddit and especially short-form video. One-tap iCloud links (roadmap item 2) and a signed or at least reputation-friendly exe (roadmap item 6) are launch blockers, not nice-to-haves.
2. **Lead with the gesture.** "Double-tap the back of my iPhone and the screenshot is already on my PC" is the hook. Back Tap itself has gone viral before: one TikTok calling it a "secret button" got about 1M views in a day and was picked up by regional papers ([The Star](https://www.thestar.co.uk/read-this/there-is-a-secret-button-on-the-back-of-your-iphone-heres-what-it-does-3548700), [Scripps/TMJ4](https://www.tmj4.com/apple-icon-back-iphone-button/)).
3. **Be honest about the main competitor.** LocalSend has around 80k stars, an App Store app and HTTPS on by default ([Trendshift](https://trendshift.io/repositories/3735), [OpenAlternative](https://openalternative.co/compare/localsend/vs/pairdrop), [F-Droid](https://f-droid.org/packages/org.localsend.localsend_app)). PhoneDrop's real edge is that there's **no app to open on either side**: share sheet, Back Tap and Action Button on the phone, a right-click Send to on the PC. Say that plainly.
4. **Expect modest numbers.** A good front-page Show HN typically brings in low hundreds of stars over the first week (data below). Most Show HNs never reach the front page, and the 2026 Show HN queue is far more crowded than it used to be.
5. **Sequence:** warm up accounts and post build-in-public content for about 3 weeks, launch the week of **16 Nov 2026** (Show HN on Tue 17 Nov, before US Thanksgiving week), ship v1.1 within about 2 weeks, then run a second wave around the Christmas new-device season.

---

## 1. Launch channels: rules, norms and evidence

### 1.1 Show HN (Hacker News)

**Rules (official).**
- Show HN is for "something you've made that other people can play with": things people can run, not blog posts, sign-up pages or reading material. You must be around to discuss it, and early work is fine if people can try it ([Show HN guidelines](https://news.ycombinator.com/showhn.html)).
- General guidelines: no uppercase or exclamation marks in titles, no soliciting upvotes, and don't use HN primarily for promotion ([HN guidelines](https://news.ycombinator.com/newsguidelines.html)). Vote rings and multiple accounts are prohibited (restated by [Stackmatix](https://www.stackmatix.com/blog/launching-on-hacker-news)).
- Reposts: one or two are OK if the story got no significant attention. Moderators also run a **second-chance pool**, and you can email hn@ycombinator.com about a neglected post. They prefer suggestions of other people's work, but self-suggestions are allowed (dang, via [hn.svelte.dev mirror](https://hn.svelte.dev/item/26998309)).
- **2026 context:** in March 2026 the moderators were discussing throttling Show HN and new accounts amid a flood of AI-generated submissions ([keydiscussions](https://keydiscussions.com/2026/03/09/hacker-news-moves-toward-restricting-show-hn-posts-amid-the-ai-slop-wave/)). I couldn't confirm what was finally adopted. In practice, **create your HN account now and comment genuinely for 3+ weeks**, so you aren't a green (new) account on launch day ([HN FAQ](https://news.ycombinator.com/newsfaq.html)).

**What the data says.**

| Source | Finding |
|---|---|
| Kraishan, "Launch-Day Diffusion" (arXiv 2511.04453, 138 AI/LLM repo launches, 2024–25) | Average **+121 stars in 24 h, +189 in 48 h, +289 in a week** after HN exposure. Posting time mattered, and the "Show HN" tag had **no** statistical advantage once other factors were controlled ([arXiv](https://arxiv.org/abs/2511.04453v1)). |
| ICSME 2025 (2,195 HN stories, 1,814 repos) | Significant increases in stars, forks *and contributors* after HN ([ICSME](https://conf.researchr.org/details/icsme-2025/icsme-2025-papers/23/Social-Media-Reactions-to-Open-Source-Promotions-AI-Powered-GitHub-Projects-on-Hacke)). |
| BookStack (Show HN reached the top) | 13k site visitors that day vs. about 700 normally. **736 stars that month** vs. a prior monthly high of 244, and about 36% more traffic retained in later months ([BookStack blog](https://www.bookstackapp.com/blog/9000-stars-and-the-effects-of-hacker-news/)). |
| PostHog | 800+ stars in the launch week, and HN success put them on GitHub Trending, which brought more users ([PostHog](https://posthog.com/blog/after-the-hn-launch)). |
| Nebula Graph | #1 on the front page, about 300 stars in 24 h ([HackerNoon](https://www.hackernoon.com/how-hacker-news-home-page-got-us-300-stars-on-github-in-24-hours-mp153w7f)). |
| Onlook | First HN launch reached the front page, 1,000+ stars ([Onlook](https://onlook.substack.com/p/launching-on-hacker-news)). |
| Max Woolf 2014 analysis | About **16%** of Show HNs reached the front page. Timing helps slightly but "won't make-or-break" ([minimaxir](https://minimaxir.com/2014/07/show-hn/), [minimaxir](https://minimaxir.com/2014/02/hacking-hacker-news/)). |
| 2026 volume trackers (informal) | Monthly Show HN submissions rose about 6x since ChatGPT (0.9k to 5.8k by Feb 2026), but posts reaching 20+ points rose only about 1.6x (128 to 211). Time on page 1 is down to about 2.9 h at US peak ([feedbagel summary](https://feedbagel.com/post/analysis-shows-increasing-percentage-of-show-hn-posts-on-hacker-news-with-decrea), [hn.svelte.dev](https://hn.svelte.dev/item/49449648)). |

**Title format.** `Show HN: PhoneDrop – AirDrop-style sharing from iPhone to Windows, no iPhone app`. The common pattern is "Show HN: Name – plain description", with no adjectives or version numbers ([Lucas F. Costa](https://lucasfcosta.com/hn-launch), [causo playbook](https://hub.causo.ai/guides/show-hn-launch-playbook-technical-founders-2026)). Alternatives to test mentally:
- `Show HN: PhoneDrop – Send files from iPhone to a Windows PC with Apple Shortcuts`
- `Show HN: I built AirDrop-style iPhone→Windows sharing with Shortcuts and a 6 MB Go exe`

HN readers like "how it works" honesty, so the Shortcuts and Go angle is a feature here.

**Timing.** Advice converges on Tue–Thu, about 7–10am US Pacific ([Lucas F. Costa](https://lucasfcosta.com/hn-launch), [Indie Hackers 2019](https://www.indiehackers.com/post/how-to-promote-an-open-source-project-on-hacker-news-lessons-learned-from-our-failure-13f98cd6a0)). Weekends have less competition but less traffic ([Wiredcraft](https://wiredcraft.com/blog/how-to-post-on-hacker-news)). All of this is heuristic.

**What wins:**
- A link people can try right now (the GitHub repo is fine).
- A first comment from the maker explaining why, how it works, limits and the threat model.
- Replying to every comment for the first 2+ hours ([dev.to guide](https://dev.to/dfarrell/how-to-crush-your-hacker-news-launch-10jk)).

**What flops:**
- Marketing-speak.
- Landing pages and waitlists.
- An absent maker.
- Defensive replies.
- Asking friends to upvote (the voting-ring detector buries it).

You usually know within about 45 minutes whether a post will climb ([Indie Hackers](https://www.indiehackers.com/post/how-to-promote-an-open-source-project-on-hacker-news-lessons-learned-from-our-failure-13f98cd6a0)).

**Draft first comment (outline):**
1. The itch ("I have an iPhone and a Windows PC; AirDrop doesn't cross that line").
2. How it works: Shortcuts sign each request with nested SHA-256, and the exe is a Go stdlib HTTP server that listens only on Private networks.
3. What it isn't: not AWDL, same Wi-Fi only, no push.
4. The threat model in four lines, and that TLS is opt-in beta.
5. "Why not LocalSend" (§5.3).
6. What you'd like feedback on.

### 1.2 Reddit

**Site-wide norm.** The "9:1" (or 90/10) ratio is a long-standing *community heuristic*, not a formal rule. Subreddit rules are always stricter than site-wide ones ([tech.co on Reddiquette](https://tech.co/?p=151355), [Redship 2026 guide](https://redship.io/blog/reddit-self-promotion-rules-2026)). Many subs auto-remove posts from low-karma or new accounts; one guide says 500–1,000 karma is a common AutoModerator floor ([SaaSCity](https://saascity.io/blog/best-subreddits-promote-startup-2026)). That's unverified, but it's why you should start commenting now.

**Per-subreddit notes** (verify in each sidebar; I couldn't load reddit.com directly):

| Subreddit | Fit | What I could verify | How to post |
|---|---|---|---|
| **r/shortcuts** (~507k members) | Excellent: the core audience for the iPhone half | Self-promotion posts are a common, visible theme there ([GummySearch](https://gummysearch.com/r/shortcuts)). Rules text not retrieved. | Lead with the *shortcut* (iCloud links, a screenshot of the actions, how request signing works in Shortcuts). The PC app is the supporting act. |
| **r/selfhosted** | Weak fit (it isn't server software), but the audience overlaps | Projects **younger than 3 months may only be posted in the weekly "New Project Megathread"** (new thread every Friday, post any day). Standalone posts are removed. They ask for name, repo, description, deployment and **AI-involvement disclosure** ([rules wiki mirror](https://redlib.groet-infra.nl/r/selfhosted/wiki/rules), [megathread](https://redlib.groet-infra.nl/r/selfhosted/comments/1tqhzoo/new_project_megathread_week_of_28_may_2026/)) | Megathread entry only. |
| **r/opensource** | Good | Promotional posts need a **"Promotional" flair** and get the same scrutiny. Blog posts are mostly treated as spam (2022 mod post, [mirror](https://lr.sudovanilla.org/r/opensource/comments/xrw5s9/new_postflairs/iqiiii3/?context=3)) | Text post: what, why, license, how to contribute. |
| **r/SideProject** | Good for feedback, low conversion to users | Promotion-friendly, but bare link drops get removed. Story and context matter ([Indie Hackers](https://www.indiehackers.com/post/every-subreddit-has-different-unwritten-rules-and-heres-how-to-decode-them-before-posting-1wSck8RGykjDv4FAwS69), [Hive Index](https://thehiveindex.com/communities/side-project/)) | "I built X because Y; here's what I learned building on Shortcuts." |
| **r/Windows11** (~281k) | Good for the PC half | Self-promotion appears as a common post theme ([GummySearch](https://gummysearch.com/r/Windows11)). Rules not retrieved. | Demo GIF plus "tray app, 6 MB, no runtime, zero idle CPU". |
| **r/windows**, **r/software** | Medium | Not retrieved. Generic guides say developer subs vary from zero tolerance to showcase threads ([teract](https://www.teract.ai/resources/reddit-marketing-developer-tools-2026)) | Read the sidebar; message the mods first. |
| **r/iphone**, **r/apple** | Huge but strict | Not retrieved. Sibling Apple subs show the pattern: r/iOSProgramming allows app posts on Saturdays only, "one app, once per year" ([mirror](https://redlib.hackliberty.org/r/iOSProgramming/comments/1k4lasw/reminder_app_saturday)), and r/AppleWatch requires mod approval ([rules mirror](https://axebps-redlib.hf.space/r/AppleWatch/wiki/rules)) | **Ask the mods before posting.** If they say no, leave it to users to post organically. |

**Formats that work on Reddit:**
- Native video or GIF in the post (not a link).
- A title that describes the outcome ("I double-tap my iPhone and the screenshot lands on my Windows PC – free, open source").
- A top comment from you with the repo link, limitations and "I'm the developer".
- Disclose that you're the developer.
- Reply to everything.
- Space subs out over several days, not all in one hour: cross-posting the same link to many subs at once looks like spam.
- Moderators often approve a post if you ask first ([Indie Hackers](https://www.indiehackers.com/post/what-subreddits-have-you-found-that-actually-allow-and-encourage-self-promotion-864d4da1cd)).

**Evidence on yield:** I found no rigorous Reddit-to-stars data. BookStack's maintainer credits early reception on r/selfhosted ([BookStack 10k](https://bookstackapp.com/blog/10k-stars-and-a-look-back-at-first-sharing)). Treat Reddit as high-variance: most posts get a handful of upvotes, and a few hit the top of a sub and bring hundreds of visitors.

### 1.3 Product Hunt: worth it for OSS in 2025/26?

- **Featuring is scarce and manual.** Only about 25% of launches were featured after Jan 2024, and the number of daily featured products fell from 47 (Sep 2023) to 16 (Sep 2024) ([Indie Hackers](https://www.indiehackers.com/post/what-it-means-not-to-be-featured-on-product-hunt-23ad7dffe3), [awesome-directories via scour](https://scour.ing/p/https://awesome-directories.com/blog/product-hunt-launch-guide-2025-algorithm-changes)).
- **Visibility comes from outside.** PH's own CEO acknowledged that much of a launch's visibility comes from external reach rather than discovery on PH ([PH AMA](https://www.producthunt.com/p/producthunt/i-m-the-product-hunt-ceo-and-i-ve-launched-8-times-on-ph-ama-unfiltered)).
- **The Open Source category is growing:** up 143% in Q2 2026 ([PH State of Tech Discovery Q2 2026](https://www.producthunt.com/p/general/product-hunt-s-state-of-tech-discovery-q2-2026)). Monthly OSS launches are small (about 38–39 per month in 2025), so a decent launch can stand out ([hunted.space](https://hunted.space/top-products/2025/August/open-source)).
- **Sobering comparable:** *WinSend*, an iPhone→Windows clipboard tool built on Shortcuts and Back Tap, got **0 upvotes, #126 of the day** ([hunted.space](https://hunted.space/product/winsend)). Even #1-of-the-day OSS launches report hundreds, not thousands, of downloads (Meridian: 400 on day one, [PH](https://www.producthunt.com/p/meridian-16/we-got-400-downloads-on-day-one-can-meridian-top-the-month)).
- **Verdict:** low priority. Do it in the second wave (week 3–5), once you have an audience to tell, a polished gallery and the 60-second video. Expect a backlink and a little traffic, not a breakout.

### 1.4 GitHub Trending

- **How it ranks.** GitHub doesn't publish the algorithm. Trending reflects *star velocity* relative to a repo's usual activity, with undisclosed extra factors, in daily, weekly and monthly windows, per language ([dev.to thread](https://dev.to/yvonnickfrin/i-wonder-how-github-s-trending-algorithm-works-any-clue-3ebf), [layer5](https://layer5.io/blog/community/on-the-importance-of-github-stars), [OSS Insight](https://ossinsight.io/blog/introducing-trending-page)). There's no public stars-per-day threshold ([TechCrunch 2013](https://techcrunch.com/2013/08/13/github-adds-trending-page-to-filter-by-project-programming-languages-and-developers)).
- **What it means for you:** concentrate traffic into a 24–48 h window (HN plus one or two Reddit posts) to have a shot at **Trending → Go (daily)**, which is far easier than overall Trending. PostHog explicitly credits HN pushing them onto Trending and Trending bringing more users ([PostHog](https://posthog.com/blog/after-the-hn-launch)).
- **Don't buy or swap stars.** There are documented "star campaigns", and they damage credibility ([Hashnode/Liam](https://hashnode.com/tag/ospo)).
- **Repo hygiene that helps Trending visitors convert:** a demo GIF above the fold, a topics list (`airdrop-alternative`, `iphone`, `windows`, `apple-shortcuts`, `file-transfer`, `go`, `tray-app`), a social preview image, and a release with assets. The repo already has `docs/launch/repo-settings.md` for this.

### 1.5 Awesome lists and directories

| Listing | Criteria / process | Fit for PhoneDrop |
|---|---|---|
| **awesome-windows** (0PandaDEV) | One item per PR, alphabetical, `* [Name](link) - Description.` Its CONTRIBUTING warns: **"Vibecoded slop and tools that don't fall in the category of 'awesome' are not welcomed"** ([CONTRIBUTING](https://raw.githubusercontent.com/0PandaDEV/awesome-windows/main/CONTRIBUTING.md)). LocalSend is listed under System Utilities ([README](https://github.com/0PandaDEV/awesome-windows)). | Good, **after** v1.0 and some real users. Have a *user* submit it if possible. |
| **awesome-selfhosted** | First release **more than 4 months old**, actively maintained, working install docs; human submissions only (an "LLM-free" checkbox); merged about a week after approval ([PR template](https://github.com/awesome-selfhosted/awesome-selfhosted-data/blob/master/.github/PULL_REQUEST_TEMPLATE.md), [CONTRIBUTING](https://github.com/awesome-selfhosted/awesome-selfhosted-data/blob/master/CONTRIBUTING.md)). | **Poor fit.** It's a list of self-hosted *server* software, and you'd be ineligible until about March 2027 anyway. Skip it. |
| **awesome-shortcuts** | I couldn't find a maintained, canonical "awesome-shortcuts" list ([ecosyste.ms search](https://awesome.ecosyste.ms/projects/1144506)). | Instead: r/shortcuts, RoutineHub (community shortcut gallery), and Matthew Cassinelli's Shortcuts blog/newsletter ([site](https://matthewcassinelli.com/tag/youtube/)). |
| **AlternativeTo** | Sign up, then User menu → "Suggest new application". Approval follows verification ([FAQ](https://alternativeto.net/faq)). Third parties report a slow free queue and an optional ~$5 priority review (unverified) ([ideaproof](https://ideaproof.io/launch-directories/alternativeto)). | **Yes, do it now.** List as an alternative to AirDrop, LocalSend, Intel Unison and Phone Link. People search "alternative to Intel Unison" because Unison was shut down in 2025 ([Windows Latest](https://windowslatest.com/2025/04/03/intel-unison-support-ends-on-windows-11-in-june-killing-off-pc-android-ios-bridge-app), [AlternativeTo news](https://alternativeto.net/news/2025/4/intel-unison-app-to-be-discontinued-by-june-2025)). |
| **Slant, SaaSHub, OpenAlternative** | Self-serve submissions. | Low effort, long-tail SEO. Do them in week 2. |
| **winget / Scoop** | `wingetcreate new`, then an automatic PR to winget-pkgs ([Microsoft Learn](https://learn.microsoft.com/windows/package-manager/package/manifest), [Authoring guide](https://github.com/microsoft/winget-pkgs/blob/master/doc/Authoring.md)). | **Yes.** `winget install PhoneDrop` is also the best answer to "I'm not running a Bypass PowerShell script from a stranger". |

### 1.6 Press tip lines and newsletters

| Outlet | How | Angle that fits them |
|---|---|---|
| **9to5Mac** | tips@9to5mac.com or the anonymous form ([contact](https://9to5mac.com/contact/)) | "Free open-source tool brings AirDrop-style sharing to Windows using only Shortcuts and Back Tap." |
| **The Verge** | tips@theverge.com (editorial only) ([how to tip](https://kompas.wasmer.app/https_www_theverge_com/c/tech/22579076/how-to-tip-the-verge-email-signal-and-more)) | Long shot. Better pitched with an interop news hook: EU DMA AirDrop interoperability ([iClarified](https://www.iclarified.com/96765/eu-orders-apple-to-open-nine-ios-connectivity-features-for-interoperability)) or Quick Share↔AirDrop on Pixel 10 not covering Windows ([TechRadar](https://www.techradar.com/phones/google-pixel-phones/its-actually-happened-android-now-works-with-apple-airdrop-for-simple-file-sharing-starting-with-the-pixel-10)). |
| **XDA, How-To Geek, MakeUseOf** (all Valnet) | I couldn't find public tip addresses. Pitch individual writers who cover Windows utilities or file transfer (find them on author pages or Muck Rack ([example](https://muckrack.com/joe-fedewa))). These outlets publish "I found this open-source app" first-person pieces; LocalSend got exactly this treatment ([PCWorld](https://www.pcworld.com/article/2288153/localsend-a-free-super-fast-way-to-transfer-files-over-wi-fi.html), [Notebookcheck](https://www.notebookcheck.com/Dieses-Open-Source-Tool-uebertraegt-die-Daten-zwischen-meinen-Geraeten-Android-Mac-Windows.1166514.0.html)). | Short, personal email, a GIF, and "replaces X for iPhone+Windows users". |
| **Thurrott, Windows Central, Neowin, PCWorld** | Covered LocalSend and Phone Link iPhone features ([Thurrott](https://www.thurrott.com/?p=305519), [PCWorld](https://www.pcworld.com/article/2554929/sending-files-from-iphone-to-windows-just-got-a-lot-faster-and-easier.html)) | Windows-first angle. |
| **Console.dev** | Picks 2–3 tools a week. Criteria include built for developers, actively maintained, and no negative security or privacy impact ([selection criteria](https://console.dev/selection-criteria)). | Weak fit (consumer tool); maybe the Go/Shortcuts protocol angle. LocalSend's interview ran there ([Console #181](https://console.substack.com/p/console-181)). |
| **Changelog News** | Submit at changelog.com/submit; self-submissions encouraged; no commercial products or tutorials ([submit](https://changelog.com/submit)). | Yes, a free submission on launch day. |
| **TLDR** | I found only paid sponsorship info ([advertise.tldr.tech](https://advertise.tldr.tech/newsletter-advertising-best-practices/)). | Skip unless it's picked up organically. |

**Press kit** (put it in `docs/press/`):
- Logo (SVG and PNG, light and dark).
- 3 screenshots (tray panel, toast, iPhone share sheet).
- The 60-second video (MP4 plus a YouTube link).
- 3 GIFs.
- A one-paragraph description.
- A fact sheet: license, size, platforms, requirements, "not affiliated with Apple".
- Founder bio and contact.
- The threat-model summary.

### 1.7 YouTube creators

- **Shortcuts specialists:** Stephen Robles (frequent "15 new Shortcuts" videos, a Shortcuts membership business) ([Cassinelli](https://matthewcassinelli.com/stephen-robles-15-new-iphone-shortcuts-auto-reply-texts-wallpapers-and-transcribe-podcasts/)) and Matthew Cassinelli (ex-Workflow/Shortcuts team, blog and newsletter) ([site](https://matthewcassinelli.com/tag/youtube/)). They are the highest-fit pitch targets for the iPhone half.
- **Windows utility channels** (verify each on YouTube; I couldn't confirm specifics via search): channels that make "free apps you should install" and "Windows tools" roundups. Chris Titus Tech is prominent in open-source Windows tooling ([winutil coverage](https://www.elevenforum.com/t/windows-utility-by-chris-titus.27320/)).
- **iPhone tips creators** on TikTok and Reels (the format behind the "secret button" Back Tap virality).
- **Pitch:** a 3-line email, the GIF, "free and MIT, no sponsorship ask, happy to answer setup questions", plus ready iCloud links. Don't pay for coverage at this stage.

### 1.8 How comparable tools actually grew

| Tool | What drove early growth | Lesson for PhoneDrop |
|---|---|---|
| **LocalSend** (~80k stars) | Developer: a **Chinese website featured the app over the New Year**, then organic word of mouth as more people wrote about it ([Console #181](https://console.substack.com/p/console-181)). Wide press (PCWorld, Thurrott, LinuxToday, Xataka, iPhones.ru) ([LinuxToday](https://www.linuxtoday.com/developer/localsend), [iphones.ru](https://www.iphones.ru/iNotes/kak-obmenivatsya-faylami-mezhdu-ios-macos-windows-i-android-bez-tancev-s-bubnom-est-besplatnaya-utilita-otlichno-rabotaet)). | International, non-English tech sites matter; the iPhone+Windows pairing is huge in China, India and elsewhere. Localising the README (and the UI good-first-issue) is a growth lever. |
| **Snapdrop → PairDrop** | Snapdrop was acquired by LimeWire (Feb 2025), which dropped the open-source site and added paid hosting and crypto ([GitHub README](https://github.com/RobinLinus/snapdrop), [AlternativeTo](https://alternativeto.net/software/snapdrop/about), [Privacy Guides](https://discuss.privacyguides.net/t/pairdrop-and-snapdrop/28416)). PairDrop, an MIT fork, inherited the community. | Trust and "will this stay free?" are real concerns. Say clearly: MIT, no account, no server, no telemetry. |
| **KDE Connect** | Long-running KDE project; the iOS app appeared on the App Store quietly in 2022 ([It's FOSS](https://itsfoss.com/news/kde-connect-ios/)). | Being part of a trusted umbrella helps; you don't have one, so earn trust with transparency instead. |
| **ShareX** (~38k stars) | Years of press roundups (Guardian, Verge, Lifehacker) and a Microsoft Store award ([Wikipedia](https://en.wikipedia.org/wiki/ShareX)). | Press "best free tools" roundups compound over years. Get on AlternativeTo and winget early so writers find you. |
| **AirDropPlus** (Python + Shortcuts, Windows↔iOS) | ~451 stars, a direct conceptual competitor; uses "AirDrop" in its name ([gittrend](https://gittrend.io/repo/yeyt97/AirDropPlus)). | Proof the niche has demand, and that a Shortcuts-based tool tops out in the hundreds without a breakout channel. Also shows what *not* to name it. |
| **WinSend** (PH) | 0 upvotes on Product Hunt ([hunted.space](https://hunted.space/product/winsend)). | Same concept and a weak channel: channel fit matters more than the idea. |
| **Clipboard Bridge, ClipPy** | Small Shortcuts+LAN projects ([Clipboard Bridge](https://github.com/Mattboxx/Clipboard-Bridge/releases), [ClipPy](https://www.github.com/Cainor/ClipPy)). | You're not first. Your differentiators are polish, security (signed requests, Private-network-only) and the Windows-native UX. |

---

## 2. Short-form video and social

### 2.1 What the platforms reward (2025–26)

- **Instagram:**
  - Ranking signals: Mosseri has publicly named **watch time, likes per reach and sends per reach** as the key Reels signals. Sends (DM shares) matter most for reaching non-followers. These are widely reported second-hand ([Hootsuite](https://blog.hootsuite.com/instagram-algorithm/), [Stack Influence](https://stackinfluence.com/blog/how-does-the-algorithm-for-reels-work)).
  - Originality: Instagram favours original, non-watermarked content made for the platform; reposts and engagement bait get less reach ([onlinemarketing.de, Instagram's tips](https://onlinemarketing.de/social-media-marketing/instagram-10-offizielle-reels-tipps)).
  - Length: Reels up to 3 minutes can be recommended to non-followers (since Jan 2025), and Instagram launched **Edits** for watermark-free editing ([Instagram creators blog](https://creators.instagram.com/blog/the-latest-with-instagram?)).
  - **Hashtags are capped at 5 per post** since Dec 2025 ([Blog du Modérateur](https://www.blogdumoderateur.com/instagram-limite-hashtags-5-par-publication/), [Siècle Digital](https://siecledigital.fr/2025/12/22/instagram-enterre-les-hashtags-a-rallonge/)).
  - **Trial Reels** show a reel to non-followers first, with insights after 24 h ([TechCrunch](https://techcrunch.com/2024/05/31/instagram-tests-trial-reels-that-dont-display-to-a-creators-followers), [9to5Mac](https://9to5mac.com/2024/12/10/instagrams-new-trial-reels-feature-lets-creators-test-content-with-non-followers)). This is perfect for a brand-new account with 0 followers: A/B test hooks with it.
- **TikTok:**
  - Frequency: Buffer's study of 11M+ posts found **2–5 posts a week** gives the biggest per-post lift (about +17% vs. weekly). Gains flatten after that, and median views stay around 500 at any frequency. "TikTok is heavy-tailed. You only need one post to pop off" ([Buffer](https://buffer.com/resources/how-often-should-you-post-on-tiktok/), [Net Influencer](https://www.netinfluencer.com/tiktok-sweet-spot-new-study-shows-2-5-weekly-posts-maximize-viewing-efficiency/)).
  - Account type: business accounts are limited to the Commercial Sounds library ([Hootsuite](https://blog.hootsuite.com/tiktok-business-vs-personal)). A personal or creator account gives you trending audio.
- **YouTube Shorts:**
  - View counting: since 31 Mar 2025 a view counts on every start or replay; "Engaged views" holds the old metric ([YouTube Help](https://support.google.com/youtube/answer/10059070?hl=en-GB), [9to5Google](https://9to5google.com/2025/03/26/youtube-shorts-views-metric/)).
  - Length: Shorts can run up to 3 minutes.
  - Search: Shorts are search-indexed for the long-tail "how to send files from iPhone to PC" queries.
- **The hook window:** vendor data (unverified) says most drop-off happens in the first 2–3 seconds, and on-screen text plus motion in the first 2 seconds lifts 3-second retention ([NestScale](https://nestscale.com/?p=47200), [Shortimize](https://www.shortimize.com/blog/video-length-sweet-spots-tiktok-reels-shorts)). Treat the exact numbers as directional.

### 2.2 Formats that make utility apps spread

1. **"I didn't know your iPhone could do this"**: the hidden-feature reveal. The Back Tap "secret button" TikTok reached about 1M views in a day and was syndicated by UK regional papers and US TV sites ([The Star](https://www.thestar.co.uk/read-this/there-is-a-secret-button-on-the-back-of-your-iphone-heres-what-it-does-3548700), [WXYZ](https://www.wxyz.com/apple-icon-back-iphone-button/)). Your product *is* a hidden-feature reveal plus a payoff on the PC.
2. **Problem→solution in under 10 seconds**: show the pain (emailing yourself a photo, WhatsApp-to-self, a USB cable) for 2 seconds, then the tap and the toast on the PC.
3. **Split-screen before/after**: iPhone on the left, PC on the right, one continuous take, no cuts, so it reads as real.
4. **POV / relatable**: "POV: you have an iPhone and a Windows PC" (the ecosystem-tension meme).
5. **Plain demo, no polish**: the Dumbphone launcher reached 2.4M TikTok views and 20k downloads in 78 days with plain black-and-white demos ([Shortimize](https://shortimize.com/blog/20k-downloads-and-9k-mrr-in-78-days-how-this-indie-app-turned-minimalism-into-millions-of-views)). ClipTone, a ringtone maker, reports 55M views across 51 product-demo videos set to trending audio ([PlayKit](https://playkit.substack.com/p/ringtone-app)). Both are vendor-reported.
6. **"Stop doing X"**: "Stop emailing yourself photos."
7. **Reply-to-comment videos**: answer "does it work with Android?" or "is it safe?" with a video. These compound engagement.

**Why it's shareable:** almost everyone knows someone with an iPhone and a Windows PC. A "send this to your friend who has an iPhone and a Windows laptop" CTA directly targets *sends per reach*.

### 2.3 Brand account or personal account?

- **Evidence:** there's no rigorous study isolating founder accounts vs. brand accounts. Creator-led content outperformed brand-only content (+23% average engagement by views) in a vendor analysis ([Dash Social](https://www.dashsocial.com/blog/brand-vs-creator-campaign-performance)). The claim that business accounts get less organic reach is unconfirmed ([Shortimize](https://www.shortimize.com/blog/why-founders-are-using-personal-tiktok-and-reels-account-to-drive-business-growth)).
- **Recommendation:** run a **personal creator-style account with the product in the name** (for example `@shaheer.builds`, bio "I make PhoneDrop – free iPhone→Windows sharing"). Keep it a personal or creator account, not a business account, so you get full music access on TikTok.
  - Face and voice when comfortable. Personal "I built this" stories outperform faceless brand posts in this niche.
  - Reserve `@phonedrop` (or a close variant) on every platform now, to stop squatting, but don't run two accounts.
- **X, Bluesky, Threads:**
  - Bluesky (about 41M registered, about 3.5M DAU end of 2025) has fewer but deeper replies, a strong dev and open-source crowd, and is good for build-in-public threads.
  - Threads (about 400M MAU) has bigger, shallower reach ([Lovable comparison, vendor](https://lovable.dev/guides/bluesky-vs-threads)).
  - X still has the largest indie-dev "build in public" scene, but I found no hard data on which converts best for OSS.
  - Cross-post the same short clips natively to all three; don't invest in threads-only content until something shows traction.

### 2.4 Posting system

- **Cadence:**
  - 3–5 short videos a week, the same clip natively on Reels, TikTok and Shorts (no watermarks; export from CapCut or Edits).
  - 2–3 text posts a week on X, Bluesky and Threads.
  - Use Trial Reels for hook tests.
- **Length:** 7–20 s for demos and 30–60 s for "how I built it". Loopable endings help, because replays count.
- **On-screen text** in the first frame. Captions always (many people watch muted).
- **Hashtags (≤5):** `#iphonetips #windows11 #iphonehacks #techtips #shortcuts`. Swap in `#backtap`, `#opensource` or `#productivity` as relevant. Hashtags are a weak signal now; topic clarity and on-screen text matter more.
- **CTA:** "Link in bio – it's free and open source." The bio link goes to the GitHub Pages landing page, not the repo, with one-tap iCloud links and a Windows download.
- **Track:** profile visits → landing page visits (use `?utm_source=tiktok` etc.) → release downloads (the GitHub release download count) → stars.

### 2.5 30-day content calendar (15 posts)

Day 1 is the first content day. Suggested start: Mon 26 Oct 2026, so posts 1–8 are pre-launch (building the account) and posts 9–15 run through and after launch week. If the launch slips, keep the sequence and shift the dates.

| # | Day | Format | Hook line (on-screen text plus first spoken line) | Notes |
|---|---|---|---|---|
| 1 | D1 | Build-in-public, face to camera | "I have an iPhone and a Windows PC. AirDrop doesn't work. So I'm building it myself." | Sets the story; ask "what would you send first?" |
| 2 | D3 | Back Tap reveal | "There's a secret button on the back of your iPhone… and I made it send screenshots to my PC." | The core viral bet. Run as a Trial Reel with 2 hook variants. |
| 3 | D5 | Problem→solution (8 s) | "Stop emailing yourself photos." | Pain 2 s, tap, toast on PC. |
| 4 | D7 | Split-screen one-take | "Zero cuts. iPhone → Windows in one tap." | Proves it's real. |
| 5 | D9 | Dev story | "Apple Shortcuts can do crypto. Here's how my iPhone signs every request." | Nerd appeal; good for X/Bluesky too. |
| 6 | D11 | Clipboard demo | "Copy on iPhone. Paste on PC. No app." | Text/link to clipboard. |
| 7 | D13 | POV meme | "POV: you're the only one in your family with a Windows PC." | Relatable; trending audio. |
| 8 | D15 | Security explainer (30–45 s) | "Strangers on your Wi-Fi can't send you files. Here's why." | Pre-empts the trust objection. |
| 9 | D17 (launch day) | Launch announcement | "It's live. Free, open source, 6 MB. Link in bio." | Pin it. Same day as Show HN. |
| 10 | D19 | Reverse direction | "Now the other way: right-click any file on Windows → it's on your iPhone." | PC→phone. |
| 11 | D21 | Reply-to-comment | "'Does this work with Android?' Honest answer…" | Point Android users to LocalSend or KDE Connect; it builds trust. |
| 12 | D23 | Action Button / Control Center | "iPhone 15 Pro and later? Your Action Button can do this." | New-device angle. |
| 13 | D25 | Numbers / build-in-public | "I launched my free app 1 week ago. Here's what happened." | Real stars and downloads, even if small. Honesty performs. |
| 14 | D27 | Contributor call | "I'm one person. Here's how you can help build this (no coding needed for some)." | Translations, testing on iOS versions. |
| 15 | D29 | Setup in 60 s | "Setting this up takes 60 seconds. Watch." | Only once the iCloud links work; it answers "too complicated?" |

Back-up hooks to test via Trial Reels:
- "Your iPhone has a hidden gesture most people never turn on."
- "AirDrop for Windows doesn't exist. So I made the next best thing."
- "This is the most useful thing I've set up on my iPhone this year."
- "Send this to your friend with an iPhone and a Windows laptop."

**Expectation setting:** with a new account, most posts will get a few hundred views (Buffer's median is about 500 views per post). The plan works if 1 in 10–15 posts breaks out. Don't judge the channel before about 15 posts.

---

## 3. Turning users into contributors and maintainers

### 3.1 What the research says

- **Documentation is the #1 barrier.** In GitHub's 2017 Open Source Survey (5,500+ respondents), **93% said incomplete or confusing documentation is a pervasive problem**, and 60% rarely or never contribute docs themselves. **64%** say a license is very important when choosing a project ([SiliconANGLE](https://siliconangle.com/blog/2017/06/04/github-survey-highlights-poor-documentation-negative-interactions-big-problems/), [SD Times](https://sdtimes.com/github-releases-2017-open-source-survey)).
- **Negative interactions drive people away.** 18% had experienced one and 50% had witnessed one; rudeness was most common ([ADTmag](https://adtmag.com/Articles/2017/06/05/open-source-survey.aspx)). A Code of Conduct plus a kind, fast first response matters.
- **Speed of first response.** A Mozilla study cited in GitHub's Open Source Guides found contributors who got a code review **within 48 hours** had much higher rates of return and repeat contribution ([opensource.guide/building-community](https://opensource.guide/building-community/)). CHAOSS defines **Time to First Response** as a core health metric; one VMware team targeted two business days ([CHAOSS starter model](https://chaoss.community/starter-project-health-metrics-model/)). Independent analysis (not peer-reviewed) found contributors who left had a much longer tail of review waits at the 75th percentile: 117 h vs. 34 h ([via the MSR literature summary](https://conf.researchr.org/details/msr-2023/msr-2023-technical-papers/18/Understanding-the-Time-to-First-Response-In-GitHub-Pull-Requests)).
- **Good first issues work, with caveats.** Tan et al. (FSE 2020, 9,368 GFIs) found GFIs onboard some newcomers, but almost half of GFIs are not solved by newcomers, and they often take longer than ordinary issues ([FSE 2020](https://2020.esec-fse.org/details/fse-2020-papers/172/A-First-Look-at-Good-First-Issues-on-GitHub)). A 2026 preprint found **interaction features** (moderate maintainer participation in the issue thread) predicted retention more than the issue's intrinsic attributes ([arXiv 2604.27532](https://arxiv.org/abs/2604.27532)). GitHub surfaces `good first issue`-labelled issues in recommendations ([GitHub Docs](https://docs.github.com/en/communities/setting-up-your-project-for-healthy-contributions/encouraging-helpful-contributions-to-your-project-with-labels)).
- **Burnout is the base case.** Tidelift's 2024 survey (400+ maintainers) found about 60% are unpaid, about **60% have quit or considered quitting**, and about half feel underappreciated ([The Register](https://www.theregister.com/2024/09/18/open_source_maintainers_underpaid/), [SD Times](https://sdtimes.com/open-source/the-state-of-open-source-maintainers/)).
- **Bus factor:** CHAOSS's Contributor Absence Factor is the smallest number of people making 50% of contributions. Yours is 1, which is normal at launch; the goal for year one is 2 ([CHAOSS](https://chaoss.community/kbtopic/all-metrics/)).

### 3.2 Concrete playbook

Already in the repo: CONTRIBUTING, CODE_OF_CONDUCT, SECURITY, issue and PR templates, ROADMAP, 8 drafted good first issues. Next steps:

1. **Open the 8 good first issues before launch** and label them `good first issue` plus an area label (`shortcuts`, `windows-ui`, `docs`). For each, write the files to touch, how to test, and "comment to claim; I'll reply within 48 h". Keep 3–5 open at all times.
2. **Make non-code contributions first-class.** PhoneDrop's biggest needs are *testing on real devices* (iOS versions, iPhone models, Windows 10 vs 11, odd routers) and *translations*. Create a `help wanted: testing` label and a "Device test report" issue template. Encryption-out-of-beta needs iOS testers: make that a public call to action.
3. **Enable GitHub Discussions** with categories: Q&A, Show your setup, Ideas, Shortcut variations. Route "how do I…" questions there so issues stay actionable. Public over private: the Open Source Guides warn private DMs exhaust maintainers ([opensource.guide](https://opensource.guide/building-community/)).
4. **A 48-hour first-response promise,** written in CONTRIBUTING. Use saved replies. A quick "thanks, I'll look this weekend" counts.
5. **Recognise people:** add the [all-contributors](https://allcontributors.org) bot, which credits code, docs, testing, translation and ideas ([spec repo](https://github.com/all-contributors/all-contributors)). Thank contributors by handle in release notes, and on social ("v1.1 shipped thanks to @x and @y").
6. **A visible roadmap:** keep `docs/ROADMAP.md`, plus a GitHub Project board with "Up for grabs". Make the big items (relay, ntfy push) design Discussions; those attract senior contributors.
7. **The second-maintainer path:** after someone lands 3+ quality PRs, offer triage rights, then review rights. Write this ladder down in CONTRIBUTING.
8. **Hacktoberfest:**
   - **2026 is nearly over** (it runs in October).
   - Mechanics: maintainers opt in with the `hacktoberfest` topic, and PRs count when merged, approved or labelled `hacktoberfest-accepted`. Recent years required about 6 accepted PRs, and spam-labelled PRs disqualify participants ([Hacktoberfest participation](https://hacktoberfest.com/participation/), [Jenkins 2025](https://www.jenkins.io/blog/2025/09/24/hacktoberfest-2025-intro/)).
   - **Plan for Oct 2027:** add the topic on 1 Oct 2027 with 10+ curated issues.
   - Hacktoberfest's history includes waves of spam PRs ([WP Tavern](https://wptavern.com/hacktoberfest-adds-gitlab-support-updates-participation-requirements-to-combat-open-source-project-spam)), so only opt in with well-scoped issues.
9. **Sponsorship:**
   - GitHub Sponsors takes no fee on personal-account sponsorships (up to 6% from organisations) ([GitHub Docs](https://docs.github.com/en/sponsors/getting-started-with-github-sponsors/about-github-sponsors)).
   - **Check that you live in a supported payout region.** The current list has 103 regions; it includes India and the UAE but not, for example, Pakistan. If you're not eligible, use Ko-fi, Liberapay or Open Collective with a fiscal host.
   - Expect little money in year one. The point of the button is the signal and recognition.
10. **Spam and AI-slop PR defences.** GitHub data shows monthly PR volume rose from about 25M to 90M (early 2023 → Mar 2026). GitHub added repo-level controls to restrict PRs to collaborators or cap open PRs from outsiders (reported live 27 Aug 2026; single source) ([Crypto Briefing](https://cryptobriefing.com/github-restricts-pull-requests-ai-slop/), [InfoWorld](https://infoworld.com/article/4127156/github-eyes-restrictions-on-pull-requests-to-rein-in-ai-based-code-deluge-on-maintainers.html), [heise](https://heise.de/-11176690)). Recommended defences:
    - A PR template requiring "how I tested on a real iPhone/PC".
    - Close PRs that touch the protocol or auth without a linked issue.
    - A short AI-use disclosure line (r/selfhosted and awesome-selfhosted both ask for one).
    - Cap open PRs per outside contributor if floods appear.
11. **Burnout guardrails:**
    - Set public hours ("I triage on Tue/Sat").
    - Use a `wontfix` + "out of scope" policy anchored in the README's "What it isn't" section.
    - Don't promise Android or Mac.
    - Automate releases (CI build, checksums), so a release costs 10 minutes.

---

## 4. Launch sequencing for a solo dev

### 4.1 Pre-launch checklist (must be done before Show HN)

| Item | Why | Status / note |
|---|---|---|
| **One-tap iCloud links for all 6 shortcuts** | The #1 conversion blocker. 20 minutes by hand loses almost everyone. | Roadmap item 2; needs a Mac to sign. Borrow a Mac, a cloud Mac, or ask an early fan. Consider a single "PhoneDrop Installer" page listing the links in order. |
| **Test the import flow on current iOS** | Shortcuts import permission UX has changed across versions (the "Allow Untrusted Shortcuts" setting, greyed-out quirks) ([Apple guide](https://support.apple.com/en-ie/guide/shortcuts/apdf01f8c054/4.0/ios), [How-To Geek](https://www.howtogeek.com/699397/how-to-fix-allow-untrusted-shortcuts-grayed-out-on-iphone-or-ipad/)). | Record the exact taps; turn them into the 60-second setup video. |
| **v1.0 tagged release, CI-built, SHA-256 checksums, CHANGELOG** | Trust; HN will ask for reproducibility. | Roadmap item 1. |
| **Signed exe or winget** | Go binaries commonly trip Defender false positives (the "Wacatac" family), and signing reportedly fixes it ([Go forum](https://forum.golangbridge.org/t/my-compiled-exe-file-is-declared-as-a-virus/34038), [Bitdefender](https://community.bitdefender.com/en/discussion/84175/false-positive-when-running-go-lang-program)). SmartScreen trust builds over time even when signed ([Lyrion policy](https://lyrion.org/signing-policy/)). | Apply to the **SignPath Foundation** (free for OSI-licensed open source; requires automated CI builds from the public repo) ([signpath.org](https://signpath.org/), [SignPath OSS](https://about.signpath.io/product/open-source)), or Microsoft **Artifact Signing** (formerly Trusted Signing; Basic tier 5,000 signatures a month, about $9.99/mo per unverified reports) ([Azure pricing](https://azure.microsoft.com/pricing/details/artifact-signing/)). Also submit the exe to Microsoft's false-positive portal and link a VirusTotal scan in the release notes. |
| **Remove the "PowerShell -ExecutionPolicy Bypass" first impression** | Reddit and HN readers will flag "unsigned exe + Bypass script + SmartScreen Run anyway". | Ship via winget (`wingetcreate`), keep `install.ps1` as the advanced path, and explain what the script does in 3 lines. |
| **Demo GIF (≤10 s) at the top of the README** | GitHub visitors decide in seconds. | Split-screen: Back Tap → toast. |
| **60-second video** | Embedded on the landing page, YouTube, press kit. | Script: 0–5 s hook, 5–25 s three demos, 25–45 s setup, 45–60 s "free, open source, how it's secure". |
| **Landing page on GitHub Pages** | Bio links and press need a non-GitHub page with big "Add to iPhone" and "Download for Windows" buttons. | Detect iPhone vs PC and show the right button first. Add OpenGraph images. |
| **Press kit** (`docs/press/`) | See §1.6. | |
| **Threat model page** | HN will scrutinise it (§5.2). | Promote the README section to `docs/SECURITY-MODEL.md` with diagrams. |
| **FAQ: "Why not LocalSend / Phone Link / KDE Connect?"** | Pre-empts the top comment. | §5.3. |
| **Accounts warmed up** | HN green-account suspicion; Reddit karma floors. | Start today: comment genuinely on HN, r/shortcuts and r/Windows11 for 3 weeks. |
| **Social handles reserved** | Squatting. | IG, TikTok, YouTube, X, Bluesky, Threads. |
| **Analytics** | Know what worked. | GitHub Insights → Traffic (referrers, 14-day window, so screenshot it daily in launch week), release download counts, UTM links on the landing page (use a privacy-friendly counter, or none). |
| **Good first issues opened, Discussions on** | Convert launch-day enthusiasm into contributors. | §3.2. |

### 4.2 Launch-week timeline (proposed: week of 16 Nov 2026)

US Thanksgiving is Thu 26 Nov 2026, so the 23–27 Nov week has lower US traffic and press attention. If the iCloud links aren't ready by 13 Nov, slip to the week of **30 Nov**. Don't launch with the manual setup.

| Day | Channel | Why this order |
|---|---|---|
| **Fri 13 Nov** | Soft launch: personal network, the r/selfhosted Friday megathread entry, AlternativeTo submission (slow queue), Changelog submission draft | Catch install bugs with friendly users first; get directory queues moving. |
| **Mon 16 Nov** | **r/shortcuts** (morning US Eastern) + short video #9 teaser | The most forgiving, highest-fit audience; collects first stars and real-device bug reports a day before HN. |
| **Tue 17 Nov, ~8am PT** | **Show HN** + first comment; stay online 6+ hours. Same day: launch Reel/TikTok/Short, X/Bluesky/Threads post, Changelog submission | HN is the biggest single star spike; a Tuesday morning PT slot gets peak readership. Concentrating traffic gives a shot at Trending → Go. |
| **Wed 18 Nov** | **r/Windows11** (or r/windows, per the rules) with native video; winget PR | Windows audience; reuse anything HN taught you in the copy. |
| **Thu 19 Nov** | **r/opensource** (Promotional flair) + **r/SideProject** (build story) | Contributor-minded audiences, once you've fixed the launch-day bugs. |
| **Fri 20 Nov** | Press pitches (9to5Mac tips, 3–5 Windows/iPhone writers, Stephen Robles and Cassinelli), with real numbers ("X stars, Y downloads in 3 days") | Social proof makes pitches land; Friday emails get read the following week. |
| **Sat–Sun** | Reply to every issue; write "What I learned launching" notes; plan v1.1 | Don't post anywhere new; fix things. |

If Show HN doesn't take off, don't repost in the same week. After a quiet attempt, consider emailing hn@ycombinator.com once to ask about the second-chance pool, or wait for v1.1 and a genuinely new angle.

### 4.3 Post-launch (weeks 2–8)

- **v1.1 within about 2 weeks** (target early Dec): fix the top 3 reported issues and ship the most-requested small feature. Credit reporters by name. The release is itself content ("You asked, I shipped").
- **Second wave (mid-Dec to Jan):**
  - New iPhones and PCs at Christmas create a natural "set up your new iPhone" season, so post setup content and pitch "best free utilities" roundups.
  - Product Hunt launch in the second wave (week of 7 Dec or 11 Jan).
  - A dev-angle blog post ("Doing request signing in Apple Shortcuts") for HN as a normal (non-Show) submission, Lobsters (invite-only; strict self-promo limits ([antonz](https://antonz.org/self-promo))) and r/golang.
- **International wave:**
  - Translate the README and landing page (Chinese, Spanish, Hindi, Portuguese, German).
  - Pitch Xataka, iPhones.ru, Notebookcheck and Chinese tech sites (sspai/少数派 is the obvious target for Shortcuts content).
  - LocalSend's breakout came from a Chinese site's New Year feature ([Console #181](https://console.substack.com/p/console-181)). Chinese New Year 2027 is 6 Feb.
- **Monthly cadence after that:** one release, one dev-story post, 3 short videos a week, and an awesome-windows PR (once it's stable, ideally submitted by a user).

### 4.4 Realistic metrics

| Scenario | Show HN | Reddit (per good post) | Short-form | Month-1 stars | Month-1 downloads |
|---|---|---|---|---|---|
| **Quiet** (most likely single outcome) | <10 points, never on the front page (~80%+ of Show HNs historically) | 5–50 upvotes | 200–1,000 views per post | 20–100 | 50–300 |
| **Good** | Front page for a few hours, 50–150 points | Top of r/shortcuts for the day, 200–800 upvotes | One clip at 20k–100k views | 200–600 (matches the arXiv mean of ~289/week from HN alone) | 500–3,000 |
| **Breakout** (rare) | #1–5, 300+ points, GitHub Trending | Cross-sub virality | A 1M+ clip (the Back Tap precedent) | 1,000–3,000+ | 5,000–20,000+ |

Sources for anchors: [arXiv 2511.04453](https://arxiv.org/abs/2511.04453v1), [BookStack](https://www.bookstackapp.com/blog/9000-stars-and-the-effects-of-hacker-news/), [PostHog](https://posthog.com/blog/after-the-hn-launch), [minimaxir](https://minimaxir.com/2014/07/show-hn/), [Buffer](https://buffer.com/resources/how-often-should-you-post-on-tiktok/). The download estimates are my extrapolation (stars to downloads is often around 1:3–1:10 for consumer tools), not sourced data. Contributors: expect **0–3 outside PRs in month 1** even in the "good" case. Most will be docs or typos. That's success.

**Failure modes to plan for:**
1. The demo goes viral but the setup is too hard, leading to "doesn't work" comments and a reputation hit.
2. A Defender or SmartScreen scare thread ("this is malware").
3. A security takedown comment on HN with no prepared answer.
4. The top comment is "just use LocalSend" and you argue instead of agreeing where it's fair.
5. Launching everywhere on one day and being unable to answer everything.
6. Spammy cross-posting gets the account flagged on Reddit.
7. Trademark complaint over "AirDrop" in the name or visuals.
8. Burnout from issue volume in weeks 1–3.
9. iOS updates breaking a Shortcuts action (iCloud shortcut links have broken globally before ([9to5Mac 2021](https://9to5mac.com/2021/03/24/icloud-shortcut-sharing-broken/))).

---

## 5. Product-specific risks

### 5.1 Apple trademark and platform policy

- **AirDrop is a registered Apple trademark** (for a software feature) ([Apple trademark list](https://www.apple.com/legal/intellectual-property/trademark/appletmlist.html)).
- **What Apple's guidelines allow:** third parties may use an Apple word mark *referentially* (for example "works with iPhone") only when the Apple term is **not part of the product name**, is less prominent than your name, and implies no endorsement. Apple's own trademarks are adjectives, not verbs ("AirDrop it" is out) ([Apple guidelines for third parties](https://www.apple.com/legal/intellectual-property/guidelinesfor3rdparties.html)). Apple's HIG adds that Apple trademarks shouldn't appear in an app's name or imagery ([HIG](https://developer-rno.apple.com/design/human-interface-guidelines/technologies/homekit/editorial)).
- **Do:**
  - Use the name "PhoneDrop".
  - Describe it as "AirDrop-style" or "like AirDrop, for iPhone and Windows" in descriptions only.
  - Add the line "AirDrop, iPhone and Apple are trademarks of Apple Inc. PhoneDrop is not affiliated with or endorsed by Apple."
- **Don't:**
  - Use Apple's logo or the AirDrop icon.
  - Use iOS-identical icons in marketing.
  - Use "AirDrop for Windows" as a title or hashtag-handle.
  - Register phonedrop-airdrop domains.
- **Hashtags:** `#airdrop` is descriptive use in a post; acceptable but optional. Lead with `#iphonetips`.
- **"PhoneDrop" itself:** search the USPTO, EUIPO and app stores for existing "PhoneDrop" marks before investing in branding. I didn't run that check; do it before ordering a logo.
- **Store policy:** with no App Store app, there's no App Review exposure. Risks are instead:
  1. iCloud shortcut links depend on Apple's service.
  2. iOS changes to Shortcuts actions (Get Contents of URL, file access).
  3. The optional TLS mode installs a configuration profile and root CA. Apple support docs and mainstream users find profile installs scary, so keep it opt-in and explain it.
- **Microsoft:** SmartScreen and Defender, as above. The Microsoft Store is optional later (MSIX), which helps trust.
- **Platform-shift risk (be upfront in the FAQ):**
  - Microsoft Phone Link now does iPhone↔PC file sharing via the "Link to Windows" iOS app, in Insider/gradual rollout ([Microsoft Learn Q&A](https://learn.microsoft.com/en-us/answers/questions/2338274/(article), [PCWorld](https://www.pcworld.com/article/2554929/sending-files-from-iphone-to-windows-just-got-a-lot-faster-and-easier.html)).
  - The EU DMA requires Apple to enable third-party AirDrop-like close-range transfer (under appeal; Wi-Fi Aware added in iOS 26) ([heise](https://heise.de/-10962179), [Macwelt](https://www.macwelt.de/article/2824392/warum-airdrop-trotz-eu-verordnung-geschlossen-bleibt.html)).
  - Google's Quick Share↔AirDrop works on Pixel 10 only, not on Windows ([Android Authority](https://www.androidauthority.com/quick-share-works-with-airdrop-3618023)).
  - None of these remove PhoneDrop's niche today, but any could narrow it.

### 5.2 Security scrutiny on HN: have answers ready

Expect these questions. The README's threat model already answers most of them; condense it into a FAQ.

| Likely HN question | Honest answer |
|---|---|
| "It's plain HTTP on the LAN?" | Yes, by default. Every request is authenticated (HMAC-like nested SHA-256 with a 256-bit per-phone secret, ±90 s timestamp, single-use nonce), so nobody else can *send* or *collect*, but a passive sniffer on your Wi-Fi can *read* transfers. Opt-in TLS (beta) exists: a name-constrained per-PC CA. LocalSend encrypts by default; we don't yet, and here's why (Shortcuts can't pin certificates, and profile install is friction). |
| "Why nested SHA-256 instead of HMAC?" | Shortcuts has no HMAC action; explain the construction and that `SHA256(k‖SHA256(k‖m))`-style constructions are used to avoid length extension. Link `docs/PROTOCOL.md`. **Get a second pair of eyes on this before launch.** |
| "The secret sits in iCloud Drive?" | Yes, because Shortcuts has no keychain. It syncs to your Apple devices and iCloud.com; anyone with it can send to and collect from the PC *while on the same LAN*. Revoke in the panel. This is the weakest point; own it. |
| "Can a malicious file auto-run?" | No. Names are sanitised, nothing is overwritten, Mark-of-the-Web is applied, and executables are only ever revealed in Explorer, never opened from a toast. |
| "Is anything listening on café Wi-Fi?" | No. It listens only on networks Windows marks Private or Domain, and drops non-private source IPs. |
| "Name constraints on user-installed roots in iOS?" | Unverified on device, hence beta. Ask HN for testers; it turns criticism into contribution. |
| "Brute force / DoS?" | Uniform 401s, a 5-minute IP lockout after repeated failures, no UI shown. |
| "Supply chain?" | Go stdlib only, no cgo, CI-built releases, checksums, (ideally) SignPath-signed with provenance. |

Make `SECURITY.md` point to a private reporting channel (GitHub private vulnerability reporting) and thank reporters publicly.

### 5.3 "Why not LocalSend?" Answer it honestly

LocalSend is excellent: cross-platform (Windows, macOS, Linux, Android, iOS), HTTPS by default, no pairing, about 80k stars, and an App Store app ([F-Droid](https://f-droid.org/packages/org.localsend.localsend_app), [AlternativeTo](https://alternativeto.net/software/localsend/about)). Don't pretend otherwise.

**Suggested FAQ answer:**

> **Use LocalSend if** you need Android/Mac/Linux, want encryption on by default, or prefer a normal app. It's great.
> **PhoneDrop is for** one specific setup, iPhone + Windows, where you never want to open an app: share sheet → done, double-tap the back of the phone → the screenshot is on the PC, copy on the phone → it's on the PC clipboard, right-click → Send to phone. Nothing to launch on either side; the PC app is a 6 MB tray exe with zero idle CPU. Only your paired phone can send; strangers on the network see nothing. Trade-offs: iPhone only, same Wi-Fi, no push to the phone, TLS still beta.

Similar honest lines for the other alternatives:
- **Phone Link:** official, also does messages and calls; file sharing requires the Link to Windows app and rolled out gradually.
- **KDE Connect:** a broader feature set; its iOS app is more limited.
- **PairDrop:** browser-based, needs both sides open in a browser.
- **Intel Unison:** discontinued in 2025 ([Tom's Hardware](https://tomshardware.com/software/intel-discontinues-unison-app-for-connecting-pcs-and-smartphones)), so its users are looking.

---

## 6. Immediate next actions (this week)

1. Create the HN account; start commenting. Start participating in r/shortcuts and r/Windows11.
2. Reserve social handles; post content #1 (build in public).
3. Get the 6 shortcuts signed on a Mac and shared as iCloud links (the launch blocker).
4. Apply to SignPath Foundation; set up a CI release with checksums; prepare the winget manifest.
5. Record the demo GIF and 60-second video; build the GitHub Pages landing page.
6. Open the 8 good first issues, enable Discussions, install the all-contributors bot.
7. Write `docs/FAQ.md` (why not LocalSend, threat model summary, trademark notice).
8. Submit to AlternativeTo now (slow queue).
9. Set the launch date to Tue 17 Nov 2026 (fallback Tue 1 Dec), conditional on step 3.
