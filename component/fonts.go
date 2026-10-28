package component

import (
	"regexp"
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

// ensureFontRegistry seeds the default web fonts and records their order. Fonts
// already configured by the caller are kept as they are.
func (c *RenderContext) ensureFontRegistry() {
	if c.Fonts == nil {
		c.Fonts = make(map[string]string)
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
func (c *RenderContext) usedFonts(body string, inlineStyles []Stylesheet) []string {
	inlineCSS := inlineStylesText(inlineStyles)

	var used []string

	for _, name := range c.orderedFonts() {
		href, ok := c.Fonts[name]
		if !ok || href == "" {
			continue
		}

		if fontUsed(body, inlineCSS, name) {
			used = append(used, name)
		}
	}

	return used
}

func fontUsed(body, inlineCSS, name string) bool {
	quoted := regexp.MustCompile(`(?i)"[^"]*font-family:[^"]*` + regexp.QuoteMeta(name) + `[^"]*"`)
	if quoted.MatchString(body) {
		return true
	}

	inline := regexp.MustCompile(`(?i)font-family:[^;}]*` + regexp.QuoteMeta(name))

	return inline.MatchString(body) || inline.MatchString(inlineCSS)
}

// inlineStylesText flattens inline stylesheets back into CSS so font usage in
// <mj-style inline="inline"> is detected the same way MJML does it.
func inlineStylesText(sheets []Stylesheet) string {
	var b strings.Builder

	for _, sheet := range sheets {
		for _, rule := range sheet.Rules {
			b.WriteString(rule.Selectors)
			b.WriteString("{")

			for _, declaration := range rule.Declarations {
				b.WriteString(declaration.Property)
				b.WriteString(":")
				b.WriteString(declaration.Value)
				b.WriteString(";")
			}

			b.WriteString("}")
		}

		for _, media := range sheet.MediaRules {
			for _, rule := range media.Rules {
				b.WriteString(rule.Selectors)
				b.WriteString("{")

				for _, declaration := range rule.Declarations {
					b.WriteString(declaration.Property)
					b.WriteString(":")
					b.WriteString(declaration.Value)
					b.WriteString(";")
				}

				b.WriteString("}")
			}
		}
	}

	return b.String()
}
