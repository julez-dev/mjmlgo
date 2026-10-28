package component

import (
	"fmt"
	"io"
	"strconv"

	"github.com/julez-dev/mjmlgo/internal/node"
)

// MJMLNavbar renders mj-navbar. The markup follows mjml-navbar 4.18.0
// (node_modules/mjml-navbar/lib/Navbar.js) including its hamburger checkbox,
// the Outlook-only link table and the component head style with the mobile
// menu rules.
type MJMLNavbar struct{}

func (n MJMLNavbar) Name() string {
	return "mj-navbar"
}

func (n MJMLNavbar) AllowedAttributes() map[string]validateAttributeFunc {
	return map[string]validateAttributeFunc{
		"align":               validateEnum([]string{"left", "center", "right"}),
		"base-url":            validateType("string"),
		"hamburger":           validateType("string"),
		"ico-align":           validateEnum([]string{"left", "center", "right"}),
		"ico-open":            validateType("string"),
		"ico-close":           validateType("string"),
		"ico-color":           validateColor(),
		"ico-font-size":       validateUnit([]string{"px", "%"}, false),
		"ico-font-family":     validateType("string"),
		"ico-text-transform":  validateType("string"),
		"ico-text-decoration": validateType("string"),
		"ico-line-height":     validateUnit([]string{"px", "%", ""}, false),
		"ico-padding":         validateUnit([]string{"px", "%"}, true),
		"ico-padding-left":    validateUnit([]string{"px", "%"}, false),
		"ico-padding-top":     validateUnit([]string{"px", "%"}, false),
		"ico-padding-right":   validateUnit([]string{"px", "%"}, false),
		"ico-padding-bottom":  validateUnit([]string{"px", "%"}, false),
		"padding":             validateUnit([]string{"px", "%"}, true),
		"padding-left":        validateUnit([]string{"px", "%"}, false),
		"padding-top":         validateUnit([]string{"px", "%"}, false),
		"padding-right":       validateUnit([]string{"px", "%"}, false),
		"padding-bottom":      validateUnit([]string{"px", "%"}, false),
	}
}

func (n MJMLNavbar) DefaultAttributes(_ *RenderContext) map[string]string {
	return map[string]string{
		"align":               "center",
		"ico-align":           "center",
		"ico-open":            "&#9776;",
		"ico-close":           "&#8855;",
		"ico-color":           "#000000",
		"ico-font-size":       "30px",
		"ico-font-family":     "Ubuntu, Helvetica, Arial, sans-serif",
		"ico-text-transform":  "uppercase",
		"ico-padding":         "10px",
		"ico-text-decoration": "none",
		"ico-line-height":     "30px",
	}
}

func (n MJMLNavbar) Render(ctx *RenderContext, w io.Writer, node *node.Node) error {
	ctx.addComponentHeadStyle(n.Name(), func(breakpoint string) string {
		return navbarHeadStyle(breakpoint)
	})

	if node.GetAttributeValueDefault("hamburger") == "hamburger" {
		n.renderHamburger(ctx, w, node)
	}

	_, _ = io.WriteString(w, `<div class="mj-inline-links" style="">`)
	_, _ = io.WriteString(w, conditionalTag(
		`<table role="presentation" border="0" cellpadding="0" cellspacing="0" align="`+
			node.GetAttributeValueDefault("align")+`"><tr>`, false))

	for _, child := range node.Children {
		switch child.Type {
		case NavbarLinkTagName:
			var link MJMLNavbarLink
			if err := InitComponent(ctx, link, child); err != nil {
				return err
			}

			if err := link.Render(ctx, w, child); err != nil {
				return fmt.Errorf("failed to render navbar link: %w", err)
			}
		case RawTagName:
			// Navbar raw content is emitted inside the navbar cell, not in a
			// table body, so it needs no markers.
			if err := renderRawChild(ctx, w, child); err != nil {
				return err
			}
		}
	}

	_, _ = io.WriteString(w, conditionalTag(`</tr></table>`, false))
	_, _ = io.WriteString(w, "</div>")

	return nil
}

func (n MJMLNavbar) renderHamburger(ctx *RenderContext, w io.Writer, node *node.Node) {
	labelKey := ctx.nextComponentID()

	_, _ = io.WriteString(w, msoConditionalTag(
		`<input type="checkbox" id="`+labelKey+`" class="mj-menu-checkbox" style="display:none !important; max-height:0; visibility:hidden;" />`,
		true))

	triggerStyle := inlineStyle{
		{Property: "display", Value: "none"},
		{Property: "max-height", Value: "0px"},
		{Property: "max-width", Value: "0px"},
		{Property: "font-size", Value: "0px"},
		{Property: "overflow", Value: "hidden"},
	}

	_, _ = io.WriteString(w, `<div class="mj-menu-trigger" style="`+triggerStyle.InlineString()+`">`)

	labelStyle := inlineStyle{
		{Property: "display", Value: "block"},
		{Property: "cursor", Value: "pointer"},
		{Property: "mso-hide", Value: "all"},
		{Property: "-moz-user-select", Value: "none"},
		{Property: "user-select", Value: "none"},
		{Property: "color", Value: node.GetAttributeValueDefault("ico-color")},
		{Property: "font-size", Value: node.GetAttributeValueDefault("ico-font-size")},
		{Property: "font-family", Value: node.GetAttributeValueDefault("ico-font-family")},
		{Property: "text-transform", Value: node.GetAttributeValueDefault("ico-text-transform")},
		{Property: "text-decoration", Value: node.GetAttributeValueDefault("ico-text-decoration")},
		{Property: "line-height", Value: node.GetAttributeValueDefault("ico-line-height")},
		{Property: "padding", Value: node.GetAttributeValueDefault("ico-padding")},
		{Property: "padding-top", Value: node.GetAttributeValueDefault("ico-padding-top")},
		{Property: "padding-right", Value: node.GetAttributeValueDefault("ico-padding-right")},
		{Property: "padding-bottom", Value: node.GetAttributeValueDefault("ico-padding-bottom")},
		{Property: "padding-left", Value: node.GetAttributeValueDefault("ico-padding-left")},
	}

	labelAttr := inlineAttributes{
		"for":   labelKey,
		"class": "mj-menu-label",
		"style": labelStyle.InlineString(),
		"align": node.GetAttributeValueDefault("ico-align"),
	}

	_, _ = io.WriteString(w, `<label `+labelAttr.InlineString()+`>`)
	_, _ = io.WriteString(w, `<span class="mj-menu-icon-open" style="mso-hide:all;">`+
		node.GetAttributeValueDefault("ico-open")+`</span>`)
	_, _ = io.WriteString(w, `<span class="mj-menu-icon-close" style="display:none;mso-hide:all;">`+
		node.GetAttributeValueDefault("ico-close")+`</span>`)
	_, _ = io.WriteString(w, `</label></div>`)
}

func navbarHeadStyle(breakpoint string) string {
	// Keep the formatting MJML's beautifier produces: component styles are
	// emitted verbatim, and a stylesheet that the canonicalizer cannot parse
	// (the accordion style ends with @goodbye) is compared as normalized text.
	return `noinput.mj-menu-checkbox { display: block !important; max-height: none !important; visibility: visible !important; }
@media only screen and (max-width:` + navbarLowerBreakpoint(breakpoint) + `) {
  .mj-menu-checkbox[type="checkbox"]~.mj-inline-links { display: none !important; }
  .mj-menu-checkbox[type="checkbox"]:checked~.mj-inline-links,
  .mj-menu-checkbox[type="checkbox"]~.mj-menu-trigger { display: block !important; max-width: none !important; max-height: none !important; font-size: inherit !important; }
  .mj-menu-checkbox[type="checkbox"]~.mj-inline-links>a { display: block !important; }
  .mj-menu-checkbox[type="checkbox"]:checked~.mj-menu-trigger .mj-menu-icon-close { display: block !important; }
  .mj-menu-checkbox[type="checkbox"]:checked~.mj-menu-trigger .mj-menu-icon-open { display: none !important; }
}`
}

// MJMLNavbarLink renders mj-navbar-link. Its content is raw HTML (the parser
// streams it as a single string).
type MJMLNavbarLink struct{}

func (l MJMLNavbarLink) Name() string {
	return "mj-navbar-link"
}

func (l MJMLNavbarLink) AllowedAttributes() map[string]validateAttributeFunc {
	return map[string]validateAttributeFunc{
		"color":           validateColor(),
		"font-family":     validateType("string"),
		"font-size":       validateUnit([]string{"px"}, false),
		"font-style":      validateType("string"),
		"font-weight":     validateType("string"),
		"href":            validateType("string"),
		"name":            validateType("string"),
		"target":          validateType("string"),
		"rel":             validateType("string"),
		"letter-spacing":  validateUnitWithNegative([]string{"px", "em"}, false),
		"line-height":     validateUnit([]string{"px", "%", ""}, false),
		"padding-bottom":  validateUnit([]string{"px", "%"}, false),
		"padding-left":    validateUnit([]string{"px", "%"}, false),
		"padding-right":   validateUnit([]string{"px", "%"}, false),
		"padding-top":     validateUnit([]string{"px", "%"}, false),
		"padding":         validateUnit([]string{"px", "%"}, true),
		"text-decoration": validateType("string"),
		"text-transform":  validateType("string"),
	}
}

func (l MJMLNavbarLink) DefaultAttributes(_ *RenderContext) map[string]string {
	return map[string]string{
		"color":           "#000000",
		"font-family":     "Ubuntu, Helvetica, Arial, sans-serif",
		"font-size":       "13px",
		"font-weight":     "normal",
		"line-height":     "22px",
		"padding":         "15px 10px",
		"target":          "_blank",
		"text-decoration": "none",
		"text-transform":  "uppercase",
	}
}

func (l MJMLNavbarLink) Render(ctx *RenderContext, w io.Writer, node *node.Node) error {
	tdStyle := inlineStyle{
		{Property: "padding", Value: node.GetAttributeValueDefault("padding")},
		{Property: "padding-top", Value: node.GetAttributeValueDefault("padding-top")},
		{Property: "padding-left", Value: node.GetAttributeValueDefault("padding-left")},
		{Property: "padding-right", Value: node.GetAttributeValueDefault("padding-right")},
		{Property: "padding-bottom", Value: node.GetAttributeValueDefault("padding-bottom")},
	}

	tdAttr := inlineAttributes{
		"style": tdStyle.InlineString(),
		"class": addSuffixToClasses(node.GetAttributeValueDefault("css-class"), "outlook"),
	}

	_, _ = io.WriteString(w, conditionalTag(`<td `+tdAttr.InlineString()+`>`, false))
	l.renderContent(ctx, w, node)
	_, _ = io.WriteString(w, conditionalTag(`</td>`, false))

	return nil
}

func (l MJMLNavbarLink) renderContent(ctx *RenderContext, w io.Writer, node *node.Node) {
	href, hasHref := node.GetAttributeValue("href")
	if node.Parent != nil && node.Parent.Type == NavbarTagName {
		if baseURL := node.Parent.GetAttributeValueDefault("base-url"); baseURL != "" {
			// MJML builds the link with a template literal, so a missing href
			// becomes the literal "undefined" when base-url is set.
			if !hasHref {
				href = "undefined"
			}

			href = baseURL + href
		}
	}

	aStyle := inlineStyle{
		{Property: "display", Value: "inline-block"},
		{Property: "color", Value: node.GetAttributeValueDefault("color")},
		{Property: "font-family", Value: node.GetAttributeValueDefault("font-family")},
		{Property: "font-size", Value: node.GetAttributeValueDefault("font-size")},
		{Property: "font-style", Value: node.GetAttributeValueDefault("font-style")},
		{Property: "font-weight", Value: node.GetAttributeValueDefault("font-weight")},
		{Property: "letter-spacing", Value: node.GetAttributeValueDefault("letter-spacing")},
		{Property: "line-height", Value: node.GetAttributeValueDefault("line-height")},
		{Property: "text-decoration", Value: node.GetAttributeValueDefault("text-decoration")},
		{Property: "text-transform", Value: node.GetAttributeValueDefault("text-transform")},
		{Property: "padding", Value: node.GetAttributeValueDefault("padding")},
		{Property: "padding-top", Value: node.GetAttributeValueDefault("padding-top")},
		{Property: "padding-left", Value: node.GetAttributeValueDefault("padding-left")},
		{Property: "padding-right", Value: node.GetAttributeValueDefault("padding-right")},
		{Property: "padding-bottom", Value: node.GetAttributeValueDefault("padding-bottom")},
	}

	className := "mj-link"
	if cssClass := node.GetAttributeValueDefault("css-class"); cssClass != "" {
		className += " " + cssClass
	}

	aAttr := inlineAttributes{
		"class":  className,
		"href":   href,
		"rel":    node.GetAttributeValueDefault("rel"),
		"target": node.GetAttributeValueDefault("target"),
		"name":   node.GetAttributeValueDefault("name"),
		"style":  aStyle.InlineString(),
	}

	_, _ = io.WriteString(w, `<a `+aAttr.InlineString()+`>`+node.Content+`</a>`)
}

// navbarLowerBreakpoint mirrors mjml-core's makeLowerBreakpoint.
func navbarLowerBreakpoint(breakpoint string) string {
	parsed, err := strconv.Atoi(RemoveNonNumeric(breakpoint))
	if err != nil {
		return breakpoint
	}

	return fmt.Sprintf("%dpx", parsed-1)
}
