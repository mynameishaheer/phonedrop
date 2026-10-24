package main

// theme follows Windows: "light/dark for Windows" (SystemUsesLightTheme, which flyouts use) and the
// accent palette (Settings > Personalisation > Colours), as Windows 11's quick settings do.
type theme struct {
	dark                                                     bool
	bg, footer, border, card, cardHover, cardPress, cardLine uint32
	text, text2, text3, accent, accentHover, accentPress     uint32
	onText, onText2, divider, danger, warn, good, chip       uint32
	shadow                                                   uint32
}

func loadTheme() theme {
	light, ok := regDword(`Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, "SystemUsesLightTheme")
	palette := regBinary(`Software\Microsoft\Windows\CurrentVersion\Explorer\Accent`, "AccentPalette")
	pick := func(i int, def uint32) uint32 { // palette: 8 RGBA entries, Light3 .. Dark3
		if len(palette) < 32 {
			return def
		}
		p := palette[i*4:]
		return 0xFF000000 | uint32(p[0])<<16 | uint32(p[1])<<8 | uint32(p[2])
	}
	if !ok || light == 0 {
		return darkTheme(vivid(pick(1, 0xFF60CDFF), 0xFF60CDFF)) // AccentLight2, like Windows 11 dark-mode accent fills
	}
	return lightTheme(vivid(pick(4, 0xFF005FB8), 0xFF005FB8)) // AccentDark1
}

// vivid: a grey accent (e.g. "automatic" from a dark wallpaper) makes "on" states look like "off"
// ones, so fall back to Windows' default blue then.
func vivid(col, fallback uint32) uint32 {
	r, g, b := int(col>>16&0xFF), int(col>>8&0xFF), int(col&0xFF)
	hi, lo := max(r, g, b), min(r, g, b)
	if hi == 0 || float32(hi-lo)/float32(hi) < 0.25 {
		return fallback
	}
	return col
}

func darkTheme(accent uint32) theme {
	t := theme{dark: true, accent: accent}
	t.bg, t.footer, t.border = 0xFF2C2C2C, 0xFF242424, 0xFF404040
	t.card, t.cardHover, t.cardPress, t.cardLine = 0x10FFFFFF, 0x18FFFFFF, 0x0AFFFFFF, 0x0CFFFFFF
	t.text, t.text2, t.text3 = 0xFFFFFFFF, 0xA6FFFFFF, 0x5CFFFFFF
	t.onText, t.onText2 = 0xFF000000, 0x99000000
	t.divider, t.danger, t.warn, t.good, t.chip = 0x14FFFFFF, 0xFFFF99A4, 0xFFFCE100, 0xFF6CCB5F, 0x14FFFFFF
	t.shadow = 0xFF000000
	t.accentHover, t.accentPress = alpha(accent, 0.9), alpha(accent, 0.8)
	return t
}

func lightTheme(accent uint32) theme {
	t := theme{accent: accent}
	t.bg, t.footer, t.border = 0xFFF9F9F9, 0xFFEEEEEE, 0xFFDCDCDC
	t.card, t.cardHover, t.cardPress, t.cardLine = 0xFFFFFFFF, 0xFFF4F4F4, 0xFFEFEFEF, 0x12000000
	t.text, t.text2, t.text3 = 0xE6000000, 0x9E000000, 0x5C000000
	t.onText, t.onText2 = 0xFFFFFFFF, 0xB3FFFFFF
	t.divider, t.danger, t.warn, t.good, t.chip = 0x12000000, 0xFFC42B1C, 0xFF9D5D00, 0xFF0F7B0F, 0x0F000000
	t.shadow = 0xFF000000
	t.accentHover, t.accentPress = alpha(accent, 0.9), alpha(accent, 0.8)
	return t
}

func themeFor(dark bool) theme {
	if dark {
		return darkTheme(0xFF60CDFF)
	}
	return lightTheme(0xFF005FB8)
}
