package component

import (
	"slices"
	"strings"
)

// builtinFonts is MJML's default web-font registry. The order matters: font
// links are emitted in registry order and <mj-font> declarations append to it.
var builtinFonts = []struct {
	Name string
	Href string
}{
	{"Open Sans", "https://fonts.googleapis.com/css?family=Open+Sans:300,400,500,700"},
	{"Droid Sans", "https://fonts.googleapis.com/css?family=Droid+Sans:300,400,500,700"},
	{"Lato", "https://fonts.googleapis.com/css?family=Lato:300,400,500,700"},
	{"Roboto", "https://fonts.googleapis.com/css?family=Roboto:300,400,500,700"},
	{"Ubuntu", "https://fonts.googleapis.com/css?family=Ubuntu:300,400,500,700"},
}

// ensureFontRegistry seeds the default web fonts and records their order. A
// caller-provided registry is kept as it is: it replaces the defaults rather
// than extending them. Its emission order is the caller's, recorded in
// FontOrder; only a context that supplies Fonts without one falls back to
// ascending family-name order, which is the only order a bare Go map offers.
func (c *RenderContext) ensureFontRegistry() {
	if c.Fonts == nil {
		c.Fonts = make(map[string]string)
	}

	if c.FontsConfigured {
		if len(c.FontOrder) == 0 {
			c.FontOrder = make([]string, 0, len(c.Fonts))
			for name := range c.Fonts {
				c.FontOrder = append(c.FontOrder, name)
			}

			slices.Sort(c.FontOrder)
		}

		return
	}

	for _, font := range builtinFonts {
		if _, exists := c.Fonts[font.Name]; !exists {
			c.Fonts[font.Name] = font.Href
		}

		if !slices.Contains(c.FontOrder, font.Name) {
			c.FontOrder = append(c.FontOrder, font.Name)
		}
	}
}

// addFont registers a font declared with <mj-font>. Overriding an existing
// family keeps its original position, matching MJML's object insertion order.
func (c *RenderContext) addFont(name, href string) {
	if c.Fonts == nil {
		c.Fonts = make(map[string]string)
	}

	if _, exists := c.Fonts[name]; !exists {
		c.FontOrder = append(c.FontOrder, name)
	}

	c.Fonts[name] = href
}

// orderedFonts returns the registry names in emission order, falling back to
// the recorded order when a context was populated without it.
func (c *RenderContext) orderedFonts() []string {
	if len(c.FontOrder) > 0 {
		return c.FontOrder
	}

	names := make([]string, 0, len(c.Fonts))
	for name := range c.Fonts {
		names = append(names, name)
	}

	slices.Sort(names)

	return names
}

// usedFonts returns the fonts from the registry whose family appears in the
// rendered body or in inline CSS, in registry order. MJML only emits imports
// for fonts that are actually used.
//
// This is the allocation-light equivalent of MJML's buildFontsTags: the body is
// walked once for quoted font-family declarations and the inline stylesheets
// once for font-family declarations, instead of compiling two regexps per
// registered font.
func (c *RenderContext) usedFonts(body string, inlineStyles []Stylesheet) []string {
	names := c.orderedFonts()

	// matched marks registry entries that are either already used or that have
	// no href and can never be imported.
	matched := make([]bool, len(names))

	pending := 0

	for i, name := range names {
		if href, ok := c.Fonts[name]; !ok || href == "" {
			matched[i] = true

			continue
		}

		pending++
	}

	if pending > 0 {
		check := func(value string) {
			if pending == 0 {
				return
			}

			for i, name := range names {
				if matched[i] || indexFoldASCII(value, name) < 0 {
					continue
				}

				matched[i] = true
				pending--
			}
		}

		forEachBodyFontFamily(body, check)

		if pending > 0 {
			forEachInlineFontFamily(inlineStyles, check)
		}
	}

	var used []string

	for i, name := range names {
		if !matched[i] {
			continue
		}

		if href, ok := c.Fonts[name]; !ok || href == "" {
			continue
		}

		used = append(used, name)
	}

	return used
}

// fontFamilyProperty is the CSS property whose declarations select a web font.
const fontFamilyProperty = "font-family"

// forEachBodyFontFamily calls fn with the text that follows the first
// font-family declaration of every segment enclosed by two quotes of body. The
// value is a sub-slice of body and is not copied.
//
// MJML matches `"[^"]*font-family:[^"]*<name>[^"]*"` against the rendered
// body, i.e. the family name has to appear after a font-family declaration, in
// text that has a quote on both sides. The reference regexp anchors on quote
// characters, not on attribute values, so every consecutive pair of quotes is a
// candidate segment - including the text between two attributes. Scanning each
// segment once per render and testing every registered name against the
// remainder of its first declaration is equivalent to the per-font regexp: a
// name that appears after any declaration in the segment also appears after the
// first one.
func forEachBodyFontFamily(body string, fn func(value string)) {
	const declaration = fontFamilyProperty + ":"

	for i := 0; i < len(body); i++ {
		if body[i] != '"' {
			continue
		}

		closing := strings.IndexByte(body[i+1:], '"')
		if closing < 0 {
			// A quote with no quote after it cannot end a match, so no later
			// quote can start one either.
			return
		}

		segment := body[i+1 : i+1+closing]

		// Resume on the closing quote: it opens the next segment.
		i += closing

		at := indexFoldASCII(segment, declaration)
		if at < 0 {
			continue
		}

		fn(segment[at+len(declaration):])
	}
}

// forEachInlineFontFamily calls fn with the value of every font-family
// declaration of the inline stylesheets, including declarations nested in
// @media rules.
//
// MJML applies `font-family:[^;}]*<name>` to each inline stylesheet string, so a
// name only counts when it appears in the value of a declaration whose property
// ends in `font-family` (the `-webkit-font-family` case matches the reference
// regexp too, since it is a plain substring match).
func forEachInlineFontFamily(sheets []Stylesheet, fn func(value string)) {
	for _, sheet := range sheets {
		for i := range sheet.Rules {
			forEachFontFamilyDeclaration(sheet.Rules[i].Declarations, fn)
		}

		for _, media := range sheet.MediaRules {
			for i := range media.Rules {
				forEachFontFamilyDeclaration(media.Rules[i].Declarations, fn)
			}
		}
	}
}

func forEachFontFamilyDeclaration(declarations []Style, fn func(value string)) {
	for _, declaration := range declarations {
		if !hasSuffixFoldASCII(declaration.Property, fontFamilyProperty) {
			continue
		}

		fn(declaration.Value)
	}
}

// indexFoldASCII returns the byte index of the first occurrence of substr in s,
// comparing ASCII letters case-insensitively, or -1 when substr is absent.
// Non-ASCII bytes only ever compare equal to themselves, which is enough for
// CSS keywords and web-font family names.
func indexFoldASCII(s, substr string) int {
	if len(substr) == 0 {
		return 0
	}

	if len(substr) > len(s) {
		return -1
	}

	first := lowerASCII(substr[0])

	for i, last := 0, len(s)-len(substr); i <= last; i++ {
		if lowerASCII(s[i]) != first || !equalFoldASCII(s[i:i+len(substr)], substr) {
			continue
		}

		return i
	}

	return -1
}

// hasSuffixFoldASCII reports whether s ends with suffix, comparing ASCII letters
// case-insensitively.
func hasSuffixFoldASCII(s, suffix string) bool {
	if len(suffix) > len(s) {
		return false
	}

	return equalFoldASCII(s[len(s)-len(suffix):], suffix)
}

func equalFoldASCII(a, b string) bool {
	for i := 0; i < len(b); i++ {
		if lowerASCII(a[i]) != lowerASCII(b[i]) {
			return false
		}
	}

	return true
}

func lowerASCII(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + ('a' - 'A')
	}

	return c
}
