# How open-source projects got famous, and what PhoneDrop can copy

Research date: **2026-10-25**. Prepared for the launch of PhoneDrop (https://github.com/mynameishaheer/phonedrop).

## How this was researched, and its limits

- **Star counts.** Every "stars now" figure was read from the project's GitHub page with WebFetch on 2026-10-25. GitHub shows rounded figures such as "95.5k". The GitHub API, OSS Insight, star-history.com, Trendshift, Wikipedia, arXiv, Hacker News and Substack all failed in this sandbox, either from DNS or a proxy 403.
- **Everything else came from web search snippets.** That covers growth curves, inflection moments and the HN studies. Each claim below is linked to the page the search returned. Where only aggregator or blog sources existed, the claim is marked *(secondary)*.
- **The search quota ran out** after about 45 searches (the per-turn limit is shared across agents). A few areas are therefore thinner than planned: Instagram and TikTok accounts, newsletter submission processes, LinkedIn performance data and SmartScreen reputation mechanics. Those gaps are marked **[gap]**. A follow-up session could fill them.
- Points marked *(from prior knowledge, not re-verified)* are things I know but couldn't confirm with a source this session.
- **Nothing here promises virality.** The research shows that most launches, even good ones, get a few hundred stars. The averages are pulled up by a few outliers (see §1.0).

---

## 1. Case studies

### 1.0 Base rates first (read before the case studies)

- **Hacker News exposure.** One study tracked 138 HN launches of AI/LLM tools in 2024–25. Repos gained an **average of 121 stars in 24 h, 189 in 48 h and 289 in a week**. Medians were much lower. Posting at **12–17 UTC** was worth about 200 stars more than the worst slots, and the "Show HN" tag gave **no statistical advantage** once other factors were controlled. (Kraishan, "Launch-Day Diffusion", arXiv 2511.04453, Nov 2025: https://arxiv.org/abs/2511.04453; repo https://github.com/obadaKraishan/Launch-Day-Diffusion)
- **An older study of HN.** It found the median repo went from 74 stars in the 3 days before an HN appearance to 138 in the 3 days after. (Borges and Valente, "How do Developers Promote Open Source Projects?", arXiv 1908.04219, 2019: https://arxiv.org/pdf/1908.04219)
- **A larger 2025 study.** It covered 2,195 HN stories and 1,814 AI repos and found significant rises in stars, forks *and contributors* in the month after a post. At least 19% of the AI developers sampled promoted their own repo on HN. (ICSME 2025, arXiv 2506.12643: https://arxiv.org/abs/2506.12643v1)
- **Fake stars are common, and they don't work for long.** About 6 million suspected fake stars were found across 18,617 repos (2019 to Dec 2024), and the effect lasts **less than two months**. (ICSE 2026, arXiv 2412.13459: https://arxiv.org/abs/2412.13459; https://conf.researchr.org/details/icse-2026/icse-2026-research-track/14/Six-Million-Suspected-Fake-Stars-on-GitHub-A-Growing-Spiral-of-Popularity-Contests). **Don't buy stars.**
- **What is trending now (GitHub Trending, monthly, fetched 2026-10-25).** All 21 entries are AI or agent tooling, for example hyperframes (+18.8k this month), VoiceStudio (+49.8k), cloudflare/security-audit-skill (+22.2k) and paperclip (+25.7k). Even the monthly Go list is led by agent tools and infrastructure. (https://github.com/trending?since=monthly, https://github.com/trending/go?since=monthly)
  - **What this means for PhoneDrop:** a non-AI Windows utility won't ride GitHub Trending the way agent tools do. Its audience lives elsewhere: Shortcuts fans, Windows users and people whose "AirDrop doesn't work with my PC" search led them to the repo.

### 1.1 LocalSend: the closest analog

- **Stars:** 95.5k, 5.4k forks (https://github.com/localsend/localsend, 2026-10-25).
- **One-liner:** "An open-source cross-platform alternative to AirDrop".
- **Origin.** Tien Do Nam wanted to send uncompressed photos between siblings on different operating systems. Snapdrop was down at the time. He shipped a minimal app to the App Store and Play Store after 3 weeks, with Windows soon after. (Console #181 interview: https://console.substack.com/p/console-181, as summarised by search, 2026-10-25)
- **Inflection.** There was no single viral post. **A Chinese website featured it over New Year**, and after that it grew "organically as more and more people wrote about it" (same interview).
  - Evidence of that press wave, across many languages:
    - Genbeta, Spain (https://www.genbeta.com/a-fondo/esta-app-parecido-a-airdrop-que-conozco-para-windows-permite-enviar-recibir-archivos-ios-android-macos-windows)
    - Dr. Windows, Germany (https://www.drwindows.de/news/software-vorstellungen/localsend-daten-mit-geraeten-ueber-lokales-netzwerk-teilen)
    - Antyweb, Poland (https://antyweb.pl/localsend-jak-dziala-multiplatformowy-airdrop)
    - iPhoned, Netherlands (https://www.iphoned.nl/apps/localsend-airdrop-windows/)
    - It's FOSS (https://news.itsfoss.com/localsend)
    - Habr (https://habr.com/en/companies/ruvds/articles/802939/)
    - TechnoTim (https://docs.technotim.com/posts/localsend/)
- **SEO effect.** AlternativeTo ranks it the top AirDrop alternative (https://alternativeto.net/software/airdrop).
- **README.**
  - The hero has a title, 3 badges (CI, Weblate translations, packaging status), links to the homepage, Discord and Codeberg, and an EN/中文 switch.
  - Then a one-paragraph intro, a table of contents, About, Sponsors (BrowserStack), **two static screenshots (no GIF)**, Download, Setup and firewall notes, How it works (protocol), CLI, Contributing (Weblate), a Troubleshooting table, Building and Contributors.
  - It says to "download the app either from an app store or from a package manager" because there's no auto-update.
  - Windows channels: Winget, Scoop, Chocolatey, EXE installer and portable ZIP. (https://github.com/localsend/localsend/blob/main/README.md, 2026-10-25)
- **Code signing.** "Windows binaries are signed" for free through **SignPath Foundation**, built by GitHub Actions, and "Every signing request is manually approved" by @Tienisto. (https://github.com/localsend/localsend/blob/main/CODE_SIGNING.md)
- **Contributors and governance.** It's essentially a BDFL project: Tienisto is the sole committer and approver in the signing policy. Translations are outsourced to the community through Weblate, and there's a documented open protocol.
- **Channel data.**
  - Flathub shows about 440k installs (https://flathub.org/apps/org.localsend.localsend_app, date not shown).
  - Chocolatey snapshots show only about 7k–19k downloads (https://community.chocolatey.org/packages/localsend, cached snapshots).
  - So Chocolatey is a minor channel even for a 95k-star app.
- **What a solo dev can copy:**
  - Own the "X alternative" phrase.
  - Get covered by tech blogs in many languages.
  - Use package managers rather than raw downloads.
  - Get free signing through SignPath.
  - Document the protocol.
  - Put a troubleshooting table in the README, mainly for firewall and network problems.

### 1.2 Snapdrop → PairDrop: web-based AirDrop clones

- **Stars:**
  - Snapdrop: 19.7k (https://github.com/SnapDrop/snapdrop).
  - PairDrop: 11.7k, 708 forks (https://github.com/schlagmichdoch/PairDrop). Both read 2026-10-25.
- **Snapdrop's history.**
  - Robin Linus built it as a progressive web app (PWA) using WebRTC (https://noted.lol/snapdrop-a-self-hosted-airdrop-alternative/).
  - It was **acquired by LimeWire on 11 Feb 2025** (PitchBook: https://pitchbook.com/profiles/company/752589-55; Korean summary: https://news.hada.io/topic?id=19496). Local sharing was dropped in favour of cloud uploads (https://alternativeto.net/software/snapdrop/about).
  - Its README now says "Snapdrop is now LimeWire".
- **PairDrop's growth.**
  - PairDrop is a fork that added paired devices, public rooms and CLI and share-menu sending, and it lists those "Differences to the Snapdrop it is based on" in its README.
  - Its growth comes partly from being the open-source refuge after the acquisition. A GitHub issue is titled "SnapDrop.net has been replaced by PairDrop.net" (https://discuss.privacyguides.net/t/pairdrop-and-snapdrop/28416).
- **README.** Logo, then "_Send it_, with PairDrop.", a one-liner, a features list, **a screenshot GIF**, a differences section, Weblate and Buy-me-a-coffee badges, and "PairDrop is libre, and always will be."
- **What a solo dev can copy:**
  - **Replacement moments drive adoption** when a beloved tool dies or "sells out". PhoneDrop has one: **Intel Unison ended in June 2025, with end of support on 1 Jan 2026** (https://windowslatest.com/2025/04/03/intel-unison-support-ends-on-windows-11-in-june-killing-off-pc-android-ios-bridge-app; https://alternativeto.net/news/2025/4/intel-unison-app-to-be-discontinued-by-june-2025; https://www.techradar.com/computing/windows/intel-is-discontinuing-an-app-youve-never-heard-of-but-sadly-that-alternative-to-windows-11s-phone-link-was-a-useful-tool-you-might-regret-never-using).
  - Also copy the "Differences to X" section.

### 1.3 Excalidraw

- **Stars:** 134.5k, 15.8k forks (https://github.com/excalidraw/excalidraw, 2026-10-25).
- **Inflection.** On 1 Jan 2020 Christopher Chedeau (vjeux) **tweeted about a small drawing app** he'd started. Features were added daily in public, and it was named on 3 Jan. Early contributor lipis found it through that tweet. (Google web.dev case study: https://web.dev/articles/excalidraw-and-fugu; Wikipedia: https://en.wikipedia.org/wiki/Excalidraw)
- **Strategy.**
  - Build in public, one feature a day, from an author with an existing following.
  - It was instantly usable at a URL with no account.
  - It was easy to embed (npm package), so integrations spread it.
- **README.**
  - Logo hero, then "An open source virtual hand-drawn style whiteboard."
  - Badges for license, npm downloads, PRs welcome, Discord, DeepWiki and X.
  - Features, quick start, then "Who's integrating Excalidraw", then sponsors.
  - Translations through Crowdin under Contributing.
- **Governance.** The community grew around the daily public progress. It later formed a company (Excalidraw+) to fund the open-source project *(from prior knowledge, not re-verified)*.
- **What a solo dev can copy:** daily build-in-public posts that show visible progress, each with a GIF. Excalidraw's leverage came from vjeux's existing following, which PhoneDrop doesn't have.

### 1.4 Microsoft PowerToys: the distribution baseline for Windows utilities

- **Stars:** 139.9k, 8.6k forks (https://github.com/microsoft/PowerToys, 2026-10-25).
- **Inflection.** It was relaunched as open source on GitHub in **May 2019**, with wide press coverage (PCWorld: https://www.pcworld.com/article/3394340/microsoft-revives-powertoys-windows-10-github.html; BleepingComputer: https://www.bleepingcomputer.com/news/microsoft/microsoft-is-relaunching-power-toys-for-windows-10-users/). Microsoft asked the public which candidate tools to build first.
- **README install.**
  - GitHub releases.
  - **Microsoft Store badge**.
  - **`winget install Microsoft.PowerToys -s winget`**.
  - Chocolatey and Scoop listed as community options.
- **What a solo dev can copy:** the order and format of install options. The one-line winget command sits right at the top, and the Store badge comes second. The scale isn't replicable, since it came from Microsoft's brand.

### 1.5 ShareX

- **Stars:** 40.4k (https://github.com/ShareX/ShareX, 2026-10-25).
- **README.** Banner, badges (build, license, release, **downloads**, Discord, Twitter), screenshot, "Why ShareX?", grouped feature lists.
- **Distribution:** website, GitHub, **Microsoft Store and Steam**. Steam is an unusual but real discovery channel for free Windows utilities.
- **What a solo dev can copy:** a "Why X?" section, and a Store or Steam presence for discovery.

### 1.6 Mac menu-bar and utility trio: Ice, Rectangle, Maccy, AltTab

These are the "replacement moment" and "do one thing well" pattern.

**Ice**
- **Stars:** 29.9k (https://github.com/jordanbaird/Ice).
- **Grew in a replacement moment.** In June 2024, **Bartender was quietly sold** to Applause. The update had a new signing certificate and added Amplitude analytics, and MacUpdater warned users. (Six Colors: https://sixcolors.com/link/2024/06/bartender-has-a-new-owner/; 512 Pixels: https://512pixels.net/2024/06/bartender-sold/; Podfeet: https://www.podfeet.com/blog/2024/06/bartender-kerfuffle/)
- Ice became the default free open-source recommendation (https://www.podfeet.com/blog/wp-json/wp/v2/posts/31248, https://talk.macpowerusers.com/t/ice-bartender-replacement-offering-secondary-bar/37510).
- **README:** one-liner, **`brew install --cask jordanbaird-ice`**, a feature checklist that doubles as a public roadmap, FAQ, screenshots and a Buy Me a Coffee button.

**Rectangle**
- **Stars:** 30.2k (https://github.com/rxhanson/Rectangle).
- "A window management app for macOS **based on Spectacle**". It inherited Spectacle's users when Spectacle went unmaintained (https://alternativeto.net/software/spectacle; https://medium.com/@skzwksk/how-to-use-rectangle-app-which-move-and-resize-windows-on-macos-a576d322e30, Dec 2019).
- It has a "Differences with Spectacle" section, a `brew` one-liner and a Known Issues section.
- It sustains itself through a paid **Rectangle Pro** built on top of the free app.

**Maccy**
- **Stars:** 22.1k (https://github.com/p0deje/Maccy).
- "Lightweight clipboard manager for macOS". Free on GitHub and Homebrew, and **$9.99 on the Mac App Store "to support the development"** (https://apps.apple.com/app/maccy/id1527619437).
- The README warns about malware look-alike domains. Once you're popular, impersonators appear, so publish your official URLs early.

**AltTab**
- **Stars:** 16.4k (https://github.com/lwouis/alt-tab-macos).
- "Windows alt-tab on macOS". JetBrains sponsorship; AltTab Pro.

**What a solo dev can copy:**
- A one-liner that names the familiar thing ("Windows alt-tab on macOS", "based on Spectacle").
- A one-command package-manager install.
- A paid store copy as an optional way to support the project.
- Be present when the incumbent stumbles.
- PhoneDrop's incumbents are **Intel Unison (dead)**, **Phone Link (limited for iPhone)** and the many **commercial "AirDrop for Windows" apps** that dominate SEO (e.g. https://bishare.app/airdrop-for-windows, https://www.copytrans.net/blog/airdrop-to-pc/).

### 1.7 Ollama: AI era, infrastructure

- **Stars:** 184.7k, 18.6k forks (https://github.com/ollama/ollama, 2026-10-25).
- **Curve.**
  - Repo created 26 Jun 2023. **Show HN "Ollama, Run LLMs on your Mac" on 20 Jul 2023 got about 284 points**, and it led with Llama 2, two days after Meta released Llama 2 *(secondary: https://brevis-research.duckdns.org/examples/ollama.html)*.
  - It first reached #1 on GitHub Trending on 29 Sep 2023 *(secondary: Trendshift via search, https://trendshift.io/repositories/7239)*.
  - Each big open-model release since then (Llama, Mistral, DeepSeek, Qwen, gpt-oss and others) has brought a wave of "how to run X locally" tutorials that say "install Ollama".
  - The company raised a $65M Series B and reports about 8.9M monthly users *(secondary: https://feedbagel.com/post/save-what-matters-1783603265)*.
- **README.**
  - Tagline, then **`curl -fsSL https://ollama.com/install.sh | sh`**, plus `irm https://ollama.com/install.ps1 | iex` for Windows.
  - `ollama run gemma4` is the quickstart.
  - Then REST API docs and a **very long "Community Integrations" list**. Every integration links back, so it acts as a backlink engine.
- **What a solo dev can copy:**
  - Launch alongside a news event (Llama 2 → Ollama; for PhoneDrop, something like "Unison is dead" or a new iOS Shortcuts feature).
  - A one-command install.
  - A protocol others can build on, such as an Android client or a Raycast extension.

### 1.8 Open WebUI

- **Stars:** 156.7k, 22.8k forks (https://github.com/open-webui/open-webui, 2026-10-25).
- **Curve.**
  - It started as "Ollama WebUI" (first release about 25 Sep 2023) and rode Ollama's growth.
  - It was renamed in early 2024 after a trademark-confusion contact from Ollama. A rename commit is dated 16 Feb 2024. *(secondary: https://aiwiki.ai/wiki/open_webui/raw; PyPI "Formerly Ollama WebUI": https://pypi.org/project/open-webui/0.3.1)*
- **README.**
  - "Open WebUI 👋", then a big **badge wall** (stars, forks, watchers, repo size, last commit, Discord, Sponsor).
  - One-liner, an enterprise callout, **a demo GIF**, Key Features, `pip install open-webui` or `docker run`, a sponsors thank-you, **a star-history chart** and Discord.
- **What a solo dev can copy:**
  - **Be the "missing UI or bridge" for something already popular.** A companion to a popular ecosystem gets free distribution. For PhoneDrop that ecosystem is Apple Shortcuts and its creators.
  - Pick a name with no trademark risk. "PhoneDrop" is generic, and that's a good thing.

### 1.9 browser-use

- **Stars:** 119.1k, 13.2k forks (https://github.com/browser-use/browser-use, 2026-10-25).
- **Curve.**
  - It launched as a **Hacker News experiment and reached #1 there**.
  - The big inflection came in **March 2025**, when **Manus** turned out to use it. An X post about this got over 2.4M views. Daily downloads went from about 5k (3 Mar) to 28k (10 Mar), and it became the top trending repo. (TechCrunch, 12 Mar 2025: https://techcrunch.com/2025/03/12/browser-use-one-of-the-tools-powering-manus-is-also-going-viral/)
- **README.**
  - Light/dark logo, badge row (Demos, Docs, Blog, Merch, Stars, Twitter, Discord, Cloud).
  - "The open-source browser agent."
  - **One demo GIF** of the agent booking a driving test.
  - "Which Browser Use do I need?" chooser, then `uv add browser-use`.
- **What a solo dev can copy:**
  - Show HN first.
  - Then be "the thing inside the thing people are talking about". The third-party virality wasn't planned.
  - A single concrete demo GIF of a real-life task.

### 1.10 screenshot-to-code

- **Stars:** 80.7k (https://github.com/abi/screenshot-to-code, 2026-10-25).
- **Curve.** It went **#1 on GitHub Trending on 19 Nov 2023** (Trendshift via search: https://trendshift.io/repositories/5047). That was about two weeks after OpenAI shipped the GPT-4 Vision API. It was a timely demo of a brand-new capability, launched by the author on X *(the specific tweet wasn't found [gap])*.
- **README.** The first line is the outcome ("Convert screenshots, mockups, Figma designs … into clean, functional code"), followed immediately by a **demo video and GIF**, supported stacks, and both local and hosted paths.
- **What a solo dev can copy:**
  - Make the demo the README.
  - Ship within days of a platform change.
  - Monetise with a hosted version (screenshottocode.com).

### 1.11 AutoGPT: the cautionary curve

- **Stars:** 187.7k, 45.9k forks (https://github.com/Significant-Gravitas/AutoGPT, 2026-10-25).
- **Curve.**
  - Released 30 Mar 2023 by Toran Bruce Richards (@SigGravitas). It was #1 trending by 3 Apr.
  - **Karpathy quote-tweeted it on 2 Apr 2023**. It had about 30k stars by 12 Apr and later passed 100k.
  - It was widely called the fastest-growing repo at the time. (https://en.wikipedia.org/wiki/AutoGPT; https://aiwiki.ai/wiki/autogpt)
- **Lesson.** Stars came from novelty and one influencer quote. The product then had to pivot to a hosted platform, now "Your team of AI Experts" with a Polyform Shield licence on the platform part. **Stars aren't users.** For a utility, the useful metrics are installs and retention, not the star spike.

### 1.12 OpenHands (ex-OpenDevin) and bolt.diy: "open version of the hyped closed thing"

**OpenHands**
- **Stars:** 91.8k (https://github.com/All-Hands-AI/OpenHands, 2026-10-25).
- Started **12 Mar 2024**, days after Cognition's Devin demo, as "OpenDevin" (https://www.aiwiki.ai/wiki/openhands; https://www.openhands.dev/blog/one-year-of-openhands-a-journey-of-open-source-ai-development).
- Renamed and turned into a company (All Hands AI) in Aug–Sep 2024.

**bolt.diy**
- **Stars:** 20.0k, 10.6k forks (https://github.com/stackblitz-labs/bolt.diy).
- Started by **Cole Medin as a showcase on his YouTube channel** (as "oTToDev"). It "quickly grew into a massive community effort". The maintainers formed a team, and it was adopted as the official open-source Bolt. (https://huggingface.co/spaces/kloudux/kodex/blob/main/FAQ.md; https://devtoollab.com/ai-tools/bolt-diy)

**What a solo dev can copy:**
- A **YouTube creator owning the project** gives it a built-in audience.
- "Open version of X" framing works when X is hyped and closed. For PhoneDrop, "AirDrop, for Windows, without an app" is that framing.

### 1.13 OpenClaw (Clawdbot → Moltbot → OpenClaw): the 2026 phenomenon

- **Stars:** about 393k, 82.6k forks (https://github.com/openclaw/openclaw, 2026-10-25). It's the most-starred software project on GitHub.
- **Curve.**
  - Peter Steinberger (PSPDFKit founder) reportedly **vibe-coded it in about 10 days in Nov 2025** *(secondary: https://blog.wenhaofree.com/en/posts/articles/burnout-restart-viral-clawdbot-creator-peter-steinbergers-35-minute-interview-transcript/)*.
  - **Day 1: 19 stars. Week 1: 232.** Then it went viral in late January 2026 and **added 150k stars in 24 days**, passing 200k on 19 Feb 2026, 88 days after creation. (star-history blog: https://www.star-history.com/blog/deepseek-harness-beats-openclaw-fastest-200k-stars)
- **Triggers.**
  - A weekend X wave on 24–25 Jan 2026, with early viral posts by individual users (https://www.eweek.com/news/clawdbot-self-hosted-ai-security-risks-neuron/; https://medium.com/@ocwholesaleauto/the-clawdbot-saga-how-one-developers-hobby-project-broke-the-internet-got-sued-and-changed-ai-f567ddff3122).
  - **Federico Viticci's MacStories essay**, linked by Daring Fireball on 26 Jan 2026 (https://daringfireball.net/linked/2026/01/26/viticci-on-clawdbot).
  - A Mac-mini-buying meme (https://mlq.ai/news/viral-clawdbot-ai-agent-sparks-surge-in-mac-mini-purchases/).
  - Trademark-driven renames on 27 and 30 Jan, which only added drama (https://hackernoon.com/from-clawdbot-to-moltbot-to-openclaw-the-chaotic-story-of-the-trending-jarvis-ai-assistant).
- **The security cost of hype.** The adjacent **Moltbook** (an agent social network) was found by Wiz on 31 Jan 2026 to expose its whole database (about 1.5M API tokens) through a client-side Supabase key without row-level security. Its founder had said he "didn't write one line of code", and Wiz's cofounder called it "a textbook result of vibe coding". (https://www.infosecurity-magazine.com/news/moltbook-exposes-user-data-api/; https://www.govinfosecurity.com/moltbook-gave-everyone-control-every-ai-agent-a-30710)
- **Contrast.** DeepSeek Harness gained **74,578 stars on its creation day** (Aug 2026), but that was a big-lab launch and isn't replicable (same star-history post).
- **What a solo dev can copy:**
  - The flat start is normal. One trusted writer's endorsement (Viticci, an Apple and **Shortcuts** authority) can be the inflection.
  - **Viticci and MacStories are directly relevant to PhoneDrop's Shortcuts angle.**
  - Moltbook shows how badly "AI wrote it" plays when security goes wrong.

### 1.14 VoiceStudio (2026): "local, free alternative to a paid product"

- **Stars:** 85.9k, +49.8k this month, the top monthly gainer (https://github.com/debpalash/VoiceStudio; https://github.com/trending?since=monthly, 2026-10-25).
- **README.**
  - Logo plus a **Trendshift badge**, then a one-liner listing everything it does, links, translations (中文/日本語) and badges.
  - **Animated GIF tour**, feature columns and screenshots.
  - **One-command install for each OS**, plus an **"install with your coding agent" prompt** and a hardware support table.
  - The repo ships CLAUDE.md and AGENTS.md. It uses AI tooling openly without making that the headline.
- **What a solo dev can copy:**
  - The "open-source alternative to [paid brand]" topic tag.
  - A GIF tour.
  - A one-liner install.
  - A paste-into-your-agent install prompt, which is a 2026 pattern.

### 1.15 Cross-case patterns

| Pattern | Seen in | Applies to PhoneDrop? |
|---|---|---|
| "Alternative to / X for Y" one-liner | LocalSend, Rectangle, AltTab, VoiceStudio, OpenDevin | Yes: "AirDrop between iPhone and Windows, no app" |
| Replacement moment (incumbent dies or sells out) | PairDrop (Snapdrop→LimeWire), Ice (Bartender), Rectangle (Spectacle) | Yes: Intel Unison is dead, Phone Link is weak on iPhone |
| Timed to a platform event | Ollama (Llama 2), screenshot-to-code (GPT-4V), OpenDevin (Devin) | Possible: iOS releases, Shortcuts features, Unison anniversary |
| One trusted writer or creator is the inflection | Excalidraw (vjeux), AutoGPT (Karpathy), OpenClaw (Viticci), bolt.diy (Cole Medin) | Yes: Shortcuts creators (Viticci, Stephen Robles, Matthew Cassinelli) |
| One-command install | Ollama, OpenClaw, VoiceStudio, Ice, Rectangle, PowerToys (winget) | **Missing in PhoneDrop on both sides** |
| Hero GIF or video | PairDrop, Open WebUI, browser-use, screenshot-to-code, VoiceStudio | **Missing (commented out)** |
| Multi-language press and translations | LocalSend, Excalidraw, Rectangle, VoiceStudio | Later |
| Paid store copy or Pro tier funds the free version | Maccy, Rectangle Pro, AltTab Pro, ShareX (Steam/Store) | Optional |

---

## 2. README anatomy

### 2.1 What the top READMEs do

All fetched 2026-10-25 from GitHub: LocalSend, PairDrop, Excalidraw, PowerToys, ShareX, Ice, Rectangle, Maccy, Ollama, Open WebUI, ComfyUI, browser-use, screenshot-to-code, Aider, OpenClaw and VoiceStudio.

1. **Hero (first screen).**
   - Logo or banner, then a **single bold one-liner**, then 3–8 badges, then a link row (Website · Download · Docs · Discord).
   - Most name a familiar reference point ("alternative to AirDrop", "Windows alt-tab on macOS").
   - Aider's badges are themselves proof points: installs, tokens per week, and a "Singularity" percentage of code written by Aider (https://github.com/Aider-AI/aider).
   - Open WebUI uses a big badge wall. LocalSend uses only 3 badges.
2. **The demo comes before any text wall.**
   - GIF: PairDrop, Open WebUI, browser-use, VoiceStudio.
   - Video plus GIF: screenshot-to-code.
   - Static screenshots: LocalSend, Ice, Rectangle, OpenHands, Open Interpreter.
   - Guidance converges on **5–15 s**, under about 2–5 MB, about 10 fps (https://agentgif.com/guides/embed-gif-github-readme/; https://dev.to/voxel51/elevate-your-github-readme-game-49mg).
   - GitHub caps images and GIFs at 10 MB. Videos are capped at 10 MB on free plans, and .mp4/.mov/.webm are accepted (https://docs.github.com/en/get-started/writing-on-github/working-with-advanced-formatting/attaching-files).
   - Video doesn't autoplay, so put a GIF in the hero and the MP4 below it.
3. **Install friction is minimised.**
   - A one-line command for each OS (Ollama, OpenClaw, VoiceStudio, Open Interpreter).
   - Or a package manager (`brew install --cask …`, `winget install …`).
   - PowerToys and LocalSend list the **Store, winget, Scoop and Chocolatey** in that order.
4. **A "why" or "differences" section.**
   - "Why ShareX?", PairDrop's "Differences to Snapdrop", Rectangle's "Differences with Spectacle", and AutoGPT's hosted vs self-host table.
5. **Features** as a scannable list or icon grid (Aider, Excalidraw, ShareX, VoiceStudio).
6. **How it works and the protocol** (LocalSend's REST/HTTPS summary and protocol link).
7. **Troubleshooting and FAQ** (LocalSend table, Rectangle Known Issues, Ice FAQ, Maccy FAQ).
8. **Social proof.**
   - Testimonials (Aider's "Kind Words From Users"), "Who's integrating" (Excalidraw), community integrations (Ollama).
   - Star-history charts appear only on big projects (Open WebUI; OpenClaw links one).
   - **Star-history at 0 stars looks bad. Add it after about 1k.**
9. **Community.** Discord badge (most), Contributing, Weblate or Crowdin translations (LocalSend, PairDrop, Excalidraw), and a contributors graph.
10. **Sponsors and support.** GitHub Sponsors, Buy Me a Coffee (PairDrop, Ice), corporate sponsors (LocalSend/BrowserStack, AltTab/JetBrains).
11. **Translations of the README itself** (LocalSend 中文, Rectangle 简体中文, Open Interpreter ES/中文/日本語, AutoGPT 8 languages, VoiceStudio 中文/日本語).
12. **Trust signals.** Signing policy (LocalSend), security policy, Maccy's malware look-alike warning, and Open Interpreter's sandbox and approvals doc.

### 2.2 What PhoneDrop's current README has

Read at `/home/user/phonedrop/README.md` on 2026-10-25. It's already stronger than most launch READMEs:

- A bold one-liner and 4 badges.
- Benefit bullets and a "From the phone / On the PC" table.
- An honest "What it isn't".
- **"Why PhoneDrop?" with a comparison table** against Phone Link, LocalSend, Intel Unison and PairDrop.
- Install steps, a Security table plus threat model, measured performance (216 MB in 19 s), Build, FAQ, Contributing, Roadmap and Licence.

### 2.3 What's missing, in priority order

1. **The demo GIF.** It's a commented-out placeholder, and `docs/media/` is empty. This is the single most important README asset.
2. **No screenshot of the tray panel or the iPhone share sheet.** `phonedrop.exe -preview` can already render panel PNGs.
3. **iPhone install is the biggest drop-off.**
   - Today it says "iCloud links coming with v1.0". You either build 6 shortcuts by hand (about 20 minutes) or sign them on a Mac.
   - Every consumer channel (Reddit, TikTok, YouTube, MacStories) needs a **one-tap "Get shortcut" link**.
   - **Don't do a broad launch before this exists.**
4. **PC install is scary for a security tool.**
   - It's two files, `powershell -ExecutionPolicy Bypass`, and an unsigned exe that triggers a SmartScreen "Run anyway".
   - What's missing: a `winget install` line, a signed binary, SHA-256 checksums and a **build-provenance attestation** command (`gh attestation verify phonedrop.exe -R mynameishaheer/phonedrop`) to show the exe came from CI (https://github.com/actions/attest-build-provenance).
5. **No "How this was built" or AI-disclosure section.** See §3.6. Don't hide it; put it in a linked doc.
6. **No official-URL notice** (Maccy-style) and no landing-page link.
7. **No community link** (Discussions is mentioned; there's no Discord, which is fine for now) and no sponsor link.
8. **No troubleshooting table.** Firewall and Public-network problems will be the top support issue, as they are for LocalSend.
9. **Not yet needed:** translations, a star-history chart, a contributors graph, testimonials. Add these after launch.

### 2.4 Template for PhoneDrop's README

```markdown
<p align="center"><img src="docs/media/logo.png" width="96" alt=""></p>
<h1 align="center">PhoneDrop</h1>
<p align="center"><b>AirDrop between your iPhone and your Windows PC. No iPhone app. One tiny tray exe.</b></p>
<p align="center">[build] [release] [MIT] [downloads] [signed: SignPath]  </p>
<p align="center"><a href="https://phonedrop.app">Website</a> · <a href="#install">Install</a> · <a href="docs/SHORTCUTS.md">Shortcuts</a> · <a href="SECURITY.md">Security</a> · <a href="…/discussions">Discussions</a></p>

<p align="center"><img src="docs/media/demo.gif" width="760" alt="iPhone share sheet → Send to PC → Windows toast, in 8 seconds"></p>
<!-- 8–12 s, side by side: iPhone (left) and Windows (right), no cuts, <3 MB; MP4 version below for detail -->

## Install (2 minutes)
**PC:** `winget install PhoneDrop.PhoneDrop`  ·  or [download the signed exe](releases/latest) (SHA-256 + `gh attestation verify` below)
**iPhone:** tap **[Get the PhoneDrop shortcuts](icloud-link)** → open PhoneDrop on the PC → **Pair** → scan the QR.

## What you can do        (the existing From the phone / On the PC table + PC → phone paragraph)
## Why PhoneDrop?         (existing comparison table; add a row "Needs an account: No")
## What it isn't          (existing)
## Security at a glance   (existing table; link to THREAT_MODEL.md and "How it was built & reviewed")
## Troubleshooting        (table: "Paused on this network" → Use here; firewall; iPhone can't find PC; SmartScreen)
## FAQ                    (existing)
## How it was built       (3 lines + link to docs/HOW-IT-WAS-BUILT.md: AI agents, human review, tests, what was found and fixed)
## Build from source      (existing)
## Contributing · Roadmap · Support the project (Sponsors / Buy me a coffee)
## Official sources       (only github.com/mynameishaheer/phonedrop and <site>; anything else is not us)
## Licence
<!-- after ~1k stars: star-history chart, contributors graph, translations (README.zh-CN.md etc.) -->
```

---

## 3. The AI-era playbook

### 3.1 Why AI-adjacent repos go viral now

- **The audience on X is AI-native.** It's the AI timeline: researchers, founders and "AI news" accounts. They post demos daily, and quote-tweets from a few large accounts move repos. Karpathy did this for AutoGPT (https://en.wikipedia.org/wiki/AutoGPT), and an anonymous X post about Manus did it for browser-use (TechCrunch, above).
- **News events come every week** (model releases, closed-product demos), and "open version of X" repos ride each one (OpenDevin, bolt.diy, Ollama).
- **GitHub Trending is now almost entirely agent tooling** (see §1.0). That creates a feedback loop: trending, then aggregator posts and YouTube recaps, then more stars. Non-AI utilities mostly don't enter it.
- **Supply has exploded too.**
  - HN users report Show HN submissions "tripled and now mostly share the same vibe-coded look", and Show HN is now over 12% of HN links (https://news.ycombinator.com/item?id=47864393; https://news.ycombinator.com/item?id=49712005).
  - On 8 Mar 2026, the thread "Ask HN: Please restrict new accounts from posting" got 600+ upvotes, and moderators said they're discussing throttling Show HN (https://keydiscussions.com/2026/03/09/hacker-news-moves-toward-restricting-show-hn-posts-amid-the-ai-slop-wave/).
  - **Use an established HN account and expect stricter scrutiny.**

### 3.2 Channels, with evidence

| Channel | What works | Evidence | PhoneDrop fit |
|---|---|---|---|
| **Hacker News (Show HN)** | Technical, honest, "here's a clever constraint" story; post 12–17 UTC; answer every comment | avg 121/189/289 stars at 24h/48h/1w, medians lower (arXiv 2511.04453) | **High.** The "HMAC in Apple Shortcuts" and "X.509 name-constrained CA for a LAN" details are classic HN material |
| **X/Twitter** | Demo video in the post itself; thread with the build story; tag relevant creators. X says the link penalty was removed, but Buffer data still shows link posts from non-Premium accounts getting near-zero median engagement since Mar 2025, so **put the link in the first reply** | https://buffer.com/resources/links-on-x/; https://ppc.land/x-drops-year-old-link-penalty-musk-tells-zuckerberg-on-platform/ | Medium. The AI-build story fits here; the product fits Apple-Shortcuts X |
| **Reddit** | Problem-first post, demo, honest limits, no corporate tone; obey each sub's rules. r/selfhosted only allows projects under 3 months old on **Fridays** | https://redlib.groet-infra.nl/r/selfhosted/comments/1rmt39o/rules_update_new_project_friday_here_to_stay/; https://www.indiehackers.com/post/every-subreddit-has-different-unwritten-rules-and-heres-how-to-decode-them-before-posting-1wSck8RGykjDv4FAwS69 | **High** for r/shortcuts, r/iphone, r/Windows11, r/software, r/opensource, r/SideProject |
| **Tech press and blogs** | A short pitch with a GIF; "free alternative to dead Intel Unison" angle | LocalSend's growth came from press "writing about it" (Console #181) across ES/DE/PL/NL/RU outlets (§1.1) | **High.** Neowin, gHacks, XDA, How-To Geek, MakeUseOf, Windows Central, MacStories, iMore, 9to5Mac; international: Genbeta, Dr. Windows, iphoned.nl |
| **Apple Shortcuts creators** | Send a ready-made iCloud link plus a 15 s clip | Viticci/MacStories was a key OpenClaw inflection (https://daringfireball.net/linked/2026/01/26/viticci-on-clawdbot); Stephen Robles (about 185k subs) makes Action Button and Shortcuts videos (https://sponsorradar.com/channels/beardfm) | **Very high.** PhoneDrop *is* a Shortcuts showcase |
| **YouTube "GitHub repo" channels** | They pick from Trending and HN | Github Awesome about 58.7k subs, daily; GithubTrends about 2.3k (https://developereducators.com/channel/githubawesome/, Aug 2026) | Medium; follows HN or Trending success |
| **Instagram/TikTok "free tools you didn't know"** | Creators like @digitaljoegeorge run "hidden websites" series (https://www.tiktok.com/discover/websites-you-didnt-know-existed). These accounts mostly feature **zero-setup** tools | **[gap]** No data found on how to get featured. Inference: DM a 10–15 s vertical clip and a one-tap link; zero-setup matters | Low until the iCloud links and winget exist; then medium. "AirDrop to Windows" is a strong hook for this audience |
| **Product Hunt** | One-day spike; the Open Source category grew 143% (PH's own Q2 2026 report) | https://www.producthunt.com/p/general/product-hunt-s-state-of-tech-discovery-q2-2026; no evidence of lasting stars | Low–medium; cheap to do once |
| **LinkedIn** | "What I learned building with AI agents" posts | **[gap]** No quantitative data found | Medium for career reasons, low for installs |
| **Newsletters** | Curators take submissions (Golang Weekly accepts reader links, https://golangweekly.com/issues/307); selfh.st weekly (https://selfh.st/newsletter-signup/); cron.weekly (https://mattiasgeniar.be/cronweekly) | **[gap]** Submission processes for Console.dev, TLDR and Changelog weren't verified | Medium. Golang Weekly (pure stdlib Go, no cgo, a GDI+ UI in Go is interesting), MacStories Weekly, Changelog News |

### 3.3 "I built X with Claude/Cursor in a weekend" posts

- **How they perform.** These posts do well on X and LinkedIn with AI-curious audiences; the story is the product. They do **badly with technical gatekeepers in 2026**:
  - HN moderators are moving to throttle Show HN amid "AI slop" (above).
  - r/selfhosted had to push all new projects to Fridays because of weekend-built AI floods (above).
  - **awesome-windows (3.1k stars), where LocalSend is listed, says: "Vibecoded slop and tools that don't fall in the category of 'awesome' are not welcomed"** (https://github.com/0PandaDEV/awesome-windows, 2026-10-25).
  - A January 2026 commentary describes hostile comment sections on weekend AI projects (https://digitalbiztalk.com/article/vibe-code-friday-how-self-hosting-communities-are-fighting-ai-spam).
- **Developer trust in AI is falling.** In the 2025 Stack Overflow survey, 46% distrust AI accuracy and 33% trust it; only 3% highly trust it; favourability fell from 72% to 60% (https://shiftmag.dev/stack-overflow-survey-2025-ai-5653/; https://www.techrepublic.com/article/news-developer-trust-in-ai-declines/).
- **Security reviewers expect AI code to be insecure.** In Veracode's 2025 GenAI Code Security Report, models chose the insecure option in 45% of 80 curated tasks across 100+ LLMs (https://www.veracode.com/press-release/ai-generated-code-poses-major-security-risks-in-nearly-half-of-all-development-tasks-veracode-research-reveals/, 30 Jul 2025). It's a vendor study with curated tasks, but it's widely cited.

### 3.4 Honest assessment: does "built by AI agents in 2 days" help or hurt?

**For a security-sensitive LAN file receiver, leading with it hurts. Disclosing it transparently, backed by evidence, is neutral to positive.**

- **Why it hurts as a headline.**
  - PhoneDrop opens a listening port and accepts files and clipboard content. The first question from HN and r/netsec-adjacent readers is "can a stranger on my Wi-Fi drop a payload?"
  - "Two days, written by AI" invites the Moltbook comparison (§1.13). It also gets the project filtered out of curated lists (awesome-windows) and subreddits.
- **Why it helps if framed well.**
  - It's a genuine story people want to read (OpenClaw's creator openly "vibe-coded" it).
  - The *process* PhoneDrop used is the opposite of the Moltbook pattern:
    - A lead agent spawned specialist agents for the QR encoder, crypto and auth, TLS, UI and **a dedicated release review**.
    - The git history shows review findings being fixed ("Fix release-review findings", "Fix server-side review findings; version 0.9.0").
    - There's a documented protocol, a written threat model that admits weaknesses (iCloud-synced pairing file, TLS beta not verified on a device), **10 Go test files run with `-race` in CI**, standard library only (no dependency supply chain), and CI-built release binaries.
  - That's a credible "AI-assisted, human-directed, review-gated" story.

**How to frame it:**

1. **Lead with the product. Put the process second.** The Show HN title and README hero say nothing about AI. The README's "How it was built" section and a linked `docs/HOW-IT-WAS-BUILT.md` say it plainly.
2. **Replace "in 2 days" with what was done to earn trust.** For example: "Built with Claude Code agents over a weekend, then reviewed: threat model, N tests, release review that found and fixed K issues." Publish the review findings and their fixes. Transparency about defects *raises* trust.
3. **Get one human security review before pushing the security story.** Ask in r/netsec, a Go security person, or a paid hour from a pentester. Then link it. Also add a "security reviewers welcome" issue and a credits section in SECURITY.md.
4. **Make claims verifiable.**
   - A signed binary (SignPath, as LocalSend does) and provenance attestations (`actions/attest-build-provenance`).
   - SHA-256 hashes.
   - A short "attack surface" list: one port, Private networks only, HMAC on every request, 401 for everything else, rate-limiting.
5. **Never overclaim.** Keep "Encrypt transfers (beta) … hasn't been verified on a device". This honesty is a strength on HN.
6. **Use the AI story on the channels that reward it** (X, LinkedIn, r/ClaudeAI, r/ChatGPTCoding, Anthropic and Claude community spaces). Keep the *product* story on the channels that reward that (r/shortcuts, r/iphone, r/Windows11, MacStories, Windows press).

### 3.5 Post drafts

**X thread outline** (post the build-story thread a few days *after* the product launch, once there are real users and issues; video in post 1, link in the first reply)

1. *(15 s side-by-side video)* "AirDrop from my iPhone to my Windows PC. No iPhone app: it's Apple Shortcuts talking to a 7 MB Go tray app. Free, MIT. How I built it with a team of Claude Code agents, and why I didn't trust them 🧵"
2. "The constraint: Shortcuts has no crypto API and no keychain. So the auth is nested SHA-256 HMAC you can compute with Shortcuts' built-in Hash action, plus a timestamp and a single-use nonce. Strangers on the Wi-Fi get a bare 401 and never a pop-up."
3. "The setup: Claude as lead dev, spawning sub-agents for the QR encoder (written from scratch, stdlib only), crypto and auth, TLS, the Win32 UI, and a separate *release reviewer* whose only job was to break it."
4. "What the reviewer found: [2–3 real findings from the commits]. All fixed before v0.9. Findings and fixes are public: [link to HOW-IT-WAS-BUILT]."
5. "What AI was bad at: [honest list, e.g. Apple's Shortcut signing server rejecting patterns, Windows firewall/network-profile edge cases]. What only a human could do: test on a real iPhone, decide the threat model."
6. "What it still can't do: no push to the phone (needs an App Store app), same Wi-Fi only, TLS mode in beta. Threat model: [link]."
7. "Security people: please try to break it. Issues and PRs are welcome. Repo in the reply ↓" *(link in reply; tag @… Shortcuts creators only if genuinely relevant)*

**LinkedIn post**

> I shipped a small open-source tool this month: PhoneDrop, AirDrop-style sharing between an iPhone and a Windows PC with no phone app at all, just Apple Shortcuts and a tiny Go tray app.
>
> The interesting part for me was *how* it was built. I acted as product owner and reviewer. Claude Code acted as lead developer, splitting the work across sub-agents: a QR encoder, the authentication scheme, optional TLS, the Windows UI, and a dedicated release-review agent whose job was to find problems before users did.
>
> Three things I learned:
> 1. Agents are fast at code and slow at *reality*. Apple's signing service, Windows network profiles and real-device testing still needed a human.
> 2. A separate reviewer role is worth it. It caught [X] issues that the builders missed.
> 3. For anything security-related, speed of writing is the wrong metric. What matters is a written threat model, tests, and review you can show.
>
> It's free and MIT-licensed, and I'd especially welcome people who want to poke holes in the security model. Link in the comments.
>
> #opensource #AI #softwareengineering

**Reddit r/SideProject** (title and body; also adapt for r/shortcuts with the iCloud link and a "here's the shortcut" framing, and for r/Windows11 with a "free Unison replacement" framing)

> **Title:** I made a free way to AirDrop between iPhone and Windows, with no iPhone app (Shortcuts + a 7 MB tray app)
>
> Since Intel Unison shut down I had no good way to throw photos and links from my iPhone to my PC. LocalSend is great but needs an app on both sides; I wanted it in the share sheet, on Back Tap and on the Action Button.
>
> PhoneDrop: share → "Send to PC" → toast on Windows with Open / Show in folder. Text goes to the clipboard, links show an "Open link" button (never opened automatically), and right-click → Send to → PhoneDrop goes the other way.
>
> [GIF]
>
> Honest limits: same Wi-Fi only, PC must be awake, and the phone collects PC→phone items by running a shortcut (no push without an App Store app). The exe is [signed / not yet signed].
>
> Security, since it listens on your LAN: paired once by QR (256-bit secret, never sent over the network), every request HMAC-signed with a timestamp and nonce, only listens on Private networks, and strangers get a silent 401. Threat model's in the README.
>
> Full disclosure: I built it with Claude Code agents (one of them was a dedicated security and release reviewer), then tested it on my own devices. The write-up of what they got wrong is linked in the repo.
>
> MIT, free, no account. Feedback very welcome, especially "it didn't work on my network" reports. [link]

---

## 4. Distribution beyond GitHub

| Channel | Effort | Measured impact (evidence) | Recommendation |
|---|---|---|---|
| **winget** (`winget-pkgs` PR) | Low: one manifest per version via wingetcreate, Komac or YamlCreate; the automated pipeline validates installers (https://github.com/microsoft/winget-pkgs/blob/master/doc/README.md). `InstallerType: portable` is supported, so a single exe works (https://www.mintlify.com/microsoft/winget-cli/concepts/manifests) | No public winget install statistics exist (https://github.com/fastfetch-cli/fastfetch/issues/1868). Indirect evidence: PowerToys and LocalSend put winget first. Windows press and Reddit commenters routinely ask "is it on winget?" *(prior knowledge)* | **Do it at v1.0.** It's also the "one command" for the README. Automate it with a release-workflow action (e.g. winget-releaser) *(prior knowledge)* |
| **Scoop** | Low, but the main bucket needs **≥500 stars and 150 forks** (https://github.com/ScoopInstaller/Scoop/wiki/Criteria-for-including-apps-in-the-main-bucket); Extras has similar popularity norms *(not verified)* | Small, developer-heavy audience | Host your own bucket now (`scoop bucket add phonedrop https://github.com/…`); submit to Extras after about 500 stars |
| **Chocolatey** | Medium (moderation queue) | LocalSend's Chocolatey downloads are about 7k–19k against 95k stars (cached snapshots, https://community.chocolatey.org/packages/localsend) | Low priority. Let a community maintainer do it |
| **Microsoft Store** | **Individual registration is free since 10 Sep 2025** (https://blogs.windows.com/windowsdeveloper/2025/09/10/free-developer-registration-for-individual-developers-on-microsoft-store/). Win32 apps are allowed. **MSIX packages get free signing, hosting and auto-update** (https://betanews.com/2025/09/12/microsoft-helps-out-app-developers-by-wiping-out-store-fees/) | Store search is a discovery channel for non-technical users (ShareX and PowerToys use it). Store installs also avoid SmartScreen and "ExecutionPolicy Bypass" fears | **Worth it in phase 2.** Caveat: MSIX containerisation may complicate the Explorer **Send To** entry, the firewall rule and startup; test first. An unsigned exe submitted as a Win32 listing doesn't get Store signing (the free signing applies to MSIX) |
| **Code signing** | **SignPath Foundation**: free for OSI-licensed open-source projects with automated CI builds and a published signing policy; Windows shows "SignPath Foundation" as publisher (https://signpath.org/terms; https://about.signpath.io/product/open-source); LocalSend uses it. **Azure Artifact Signing**: $9.99/month, but individuals must be in the **US or Canada** (https://learn.microsoft.com/azure/artifact-signing/quickstart; https://devclass.com/2026/01/14/code-signing-windows-apps-may-be-easier-and-more-secure-with-new-azure-artifact-service/) | It removes the scariest step for a security-sensitive tool. SmartScreen reputation mechanics weren't verified this session **[gap]**. Signed binaries still build reputation over time *(prior knowledge)* | **Apply to SignPath now.** PhoneDrop already has an MIT licence and a CI build. Add `CODE_SIGNING.md` modelled on LocalSend's |
| **GitHub Pages landing site** | Low–medium | Indirect: the "AirDrop for Windows" search results are full of commercial SEO pages (bishare, copytrans, Wondershare, EaseUS), so demand exists and is being captured by others (https://bishare.app/airdrop-for-windows; https://www.copytrans.net/blog/airdrop-to-pc/; https://mobi.easeus.com/iphone-data-transfer/airdrop-alternatives-for-windows.html) | Do it: one page, the GIF, two buttons (winget/exe, iCloud shortcuts), FAQ targeting "AirDrop iPhone to Windows", "Intel Unison alternative" and "Phone Link iPhone file transfer not working". Custom domain optional |
| **AlternativeTo** | Very low | AlternativeTo ranks LocalSend as the top AirDrop alternative. Its pages rank on "alternative to X" searches and it posted news about Unison's shutdown (https://alternativeto.net/news/2025/4/intel-unison-app-to-be-discontinued-by-june-2025). Traffic estimates are unreliable, about 200k/month from a third party (https://www.stork.ai/startups/alternativeto) | Do it on day 1: list it as an alternative to AirDrop, Intel Unison, Phone Link, LocalSend and Snapdrop |
| **"Awesome" lists** | Low per list | Small direct traffic; adds backlinks and legitimacy. awesome-windows explicitly rejects "vibecoded slop" (above) | Submit to awesome-windows (System Utilities, next to LocalSend) only after v1.0 is polished, signed and on winget. Other candidates *(not verified this session)*: awesome-shortcuts-style lists, awesome-go (strict criteria, mostly libraries) |

**Which channels measurably drive installs for Windows utilities?** Public data is thin.

- The only hard per-channel number found was Chocolatey's, and it's small.
- The strongest *documented* driver for the closest analog (LocalSend) was **tech-press articles in many languages, and search**.
- Package managers mainly **reduce friction and add trust** for people who already found you; they don't bring discovery.
- The Store is the only package channel with built-in discovery for non-technical users.

---

## 5. Synthesis: the 10 highest-leverage actions for PhoneDrop

Ranked by expected impact relative to effort. "Impact" means real installs and retained users, not stars.

1. **Publish one-tap iCloud shortcut links before any broad launch.** *(Impact: very high. Effort: medium, needs a Mac or a friend with one.)*
   - **Evidence:** every high-growth case has a one-command or one-tap install (Ollama, OpenClaw, VoiceStudio, Ice/Rectangle via brew, PowerToys via winget; §1.15).
   - Today's iPhone path is about 20 minutes of manual building. Consumer channels (Reddit, TikTok and Instagram, Shortcuts YouTubers) won't convert with that.
2. **Record the 8–12 s side-by-side demo** (share sheet → toast), plus a vertical 15 s version. *(Impact: very high. Effort: low.)*
   - **Evidence:** the hero GIF or video is the common element of PairDrop, Open WebUI, browser-use, screenshot-to-code and VoiceStudio. Under 3 MB, about 10 fps, 10 MB GitHub cap (§2.1).
   - browser-use's inflection was a 2.4M-view X post showing a demo in a real product (TechCrunch, 12 Mar 2025).
3. **Fix PC install trust: SignPath signing, provenance attestation, SHA-256, and `winget install`.** *(Impact: high. Effort: low–medium.)*
   - **Evidence:** LocalSend signs through SignPath Foundation for free. PowerToys and LocalSend lead with winget or package managers (§1.1, §1.4).
   - "ExecutionPolicy Bypass" plus "Run anyway" undercuts a tool whose pitch is security.
4. **Write `docs/HOW-IT-WAS-BUILT.md` and `THREAT_MODEL.md`, and get one independent human security review.** *(Impact: high on credibility. Effort: medium.)*
   - **Evidence:** developer distrust of AI is at 46% (SO 2025). Veracode found 45% insecure choices. The Moltbook breach became the "vibe coding" cautionary tale. awesome-windows rejects vibe-coded tools (§3.3–3.4).
   - Publishing the release-review findings and fixes turns the AI story from a liability into proof.
5. **Show HN with a technical, non-AI title.** For example: "Show HN: PhoneDrop – AirDrop from iPhone to Windows using only Apple Shortcuts".
   - Post 12–17 UTC on a weekday from an established account. Have the author answer every comment for 6 hours. The body should explain the HMAC-in-Shortcuts and name-constrained CA tricks.
   - *(Impact: medium–high. Effort: low.)*
   - **Evidence:** average 289 stars in a week, medians lower, timing worth about 200 stars, the Show HN tag itself neutral (arXiv 2511.04453). HN also lifts contributors (arXiv 2506.12643).
   - Expect stricter 2026 moderation (keydiscussions, Mar 2026).
6. **Pitch the Apple Shortcuts world directly**: MacStories/Viticci, Stephen Robles, Matthew Cassinelli and r/shortcuts. Send the iCloud link and the vertical clip. *(Impact: high if one bites, otherwise zero. Effort: low.)*
   - **Evidence:** Viticci's essay was a key OpenClaw inflection (Daring Fireball, 26 Jan 2026). Robles has about 185k subscribers and covers the Action Button and Shortcuts.
   - PhoneDrop is a showcase of what Shortcuts can do, which is the angle these creators cover.
7. **Pitch Windows and international tech press on the "free Intel Unison replacement" angle.** Targets: Neowin, gHacks, XDA, How-To Geek, MakeUseOf, Windows Central, plus Genbeta, Dr. Windows, iphoned.nl and It's FOSS. *(Impact: medium–high, slow-burn. Effort: low.)*
   - **Evidence:** LocalSend's documented growth driver was people writing about it, including a Chinese site at New Year and many European outlets (§1.1).
   - The Unison shutdown left users looking for a replacement (TechRadar, Windows Latest, AlternativeTo news).
8. **List on AlternativeTo and build a one-page GitHub Pages site** targeting the "AirDrop for Windows / Unison alternative / Phone Link not working" searches. *(Impact: medium, compounding. Effort: low.)*
   - **Evidence:** commercial SEO pages already compete for this demand. AlternativeTo is where LocalSend ranks #1 for AirDrop alternatives (§4).
9. **Run problem-first Reddit posts, one subreddit at a time, following each sub's rules**: r/shortcuts, r/iphone, r/Windows11, r/software, r/opensource, r/SideProject, and r/selfhosted only on Fridays. Post the AI-build story separately to r/ClaudeAI. *(Impact: medium. Effort: low.)*
   - **Evidence:** r/selfhosted's Friday rule and AI-spam fatigue. Reddit's self-promotion norms favour problem-first posts asking for feedback (§3.2).
10. **Microsoft Store listing (MSIX, free individual account) and the build-story content** (X thread and LinkedIn) as a second wave after v1.0. *(Impact: medium. Effort: medium.)*
    - **Evidence:** the Store has been free for individuals since Sep 2025, with free MSIX signing and auto-update. ShareX and PowerToys use the Store for reach.
    - Build-in-public worked for Excalidraw, but because the author already had followers. Treat the AI story as amplification, not the launch.

**Don't:**
- Buy stars. The effect lasts under 2 months and it's detectable (ICSE 2026).
- Lead with "built by AI in 2 days" on security-minded channels.
- Add a star-history chart before about 1k stars.
- Launch broadly before items 1–3 are done. A launch spike that lands on a 20-minute setup wastes the one first impression most launches get.

**Realistic expectation:** a well-executed HN launch for a niche utility typically brings hundreds of stars, not tens of thousands (the base rates in §1.0). The big outcomes in this report all came from **compounding**: press articles, search, package-manager presence, and one creator or writer picking it up. These are slow, repeatable actions. Virality, when it came, was unplanned (browser-use via Manus, OpenClaw after 2 quiet months).
