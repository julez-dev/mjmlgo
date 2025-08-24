package component

import (
	"io"

	"github.com/julez-dev/mjmlgo/node"
)

type MJMLSocial struct{}

func (s MJMLSocial) Name() string {
	return "mj-social"
}

func (s MJMLSocial) AllowedAttributes() map[string]validateAttributeFunc {
	return map[string]validateAttributeFunc{
		"align":                      validateEnum([]string{"left", "right", "center"}),
		"border-radius":              validateUnit([]string{"px", "%"}, false),
		"container-background-color": validateColor(),
		"color":                      validateColor(),
		"font-family":                validateType("string"),
		"font-size":                  validateUnit([]string{"px"}, false),
		"font-style":                 validateType("string"),
		"font-weight":                validateType("string"),
		"icon-size":                  validateUnit([]string{"px", "%"}, false),
		"icon-height":                validateUnit([]string{"px", "%"}, false),
		"icon-padding":               validateUnit([]string{"px", "%"}, true),
		"inner-padding":              validateUnit([]string{"px", "%"}, true),
		"line-height":                validateUnit([]string{"px", "%"}, false),
		"mode":                       validateEnum([]string{"horizontal", "vertical"}),
		"padding-bottom":             validateUnit([]string{"px", "%"}, false),
		"padding-left":               validateUnit([]string{"px", "%"}, false),
		"padding-right":              validateUnit([]string{"px", "%"}, false),
		"padding-top":                validateUnit([]string{"px", "%"}, false),
		"padding":                    validateUnit([]string{"px", "%"}, true),
		"table-layout":               validateEnum([]string{"auto", "fixed"}),
		"text-padding":               validateUnit([]string{"px", "%"}, true),
		"text-decoration":            validateType("string"),
		"vertical-align":             validateEnum([]string{"top", "bottom", "middle"}),
	}
}

func (s MJMLSocial) DefaultAttributes(_ *RenderContext) map[string]string {
	return map[string]string{
		"align":           "center",
		"border-radius":   "3px",
		"color":           "#333333",
		"font-family":     "Ubuntu, Helvetica, Arial, sans-serif",
		"font-size":       "13px",
		"icon-size":       "20px",
		"line-height":     "22px",
		"mode":            "horizontal",
		"padding":         "10px 25px",
		"text-decoration": "none",
	}
}

func (s MJMLSocial) Render(ctx *RenderContext, w io.Writer, n *node.Node) error {
	if n.GetAttributeValueDefault("mode") == "horizontal" {
		if err := s.renderHorizontal(ctx, w, n); err != nil {
			return err
		}
		return nil
	}

	return s.renderVertical(ctx, w, n)
}

func (s MJMLSocial) renderHorizontal(ctx *RenderContext, w io.Writer, n *node.Node) error {
	tdAttr := inlineAttributes{
		"align":       n.GetAttributeValueDefault("align"),
		"border":      "0",
		"cellpadding": "0",
		"cellspacing": "0",
		"role":        "presentation",
	}

	_, _ = io.WriteString(w, "<!--[if mso | IE]>")
	_, _ = io.WriteString(w, "<table "+tdAttr.InlineString()+">")
	_, _ = io.WriteString(w, "<tr mjml-social-test>")
	_, _ = io.WriteString(w, "<![endif]-->")

	attr := s.getSocialElementAttributes(n)

	for _, child := range n.Children {
		if child.Type != SocialElementTagName {
			continue
		}

		for key, value := range attr {
			child.SetAttribute(key, value)
		}

		childTdAttr := inlineAttributes{
			"align":       n.GetAttributeValueDefault("align"),
			"border":      "0",
			"cellpadding": "0",
			"cellspacing": "0",
			"role":        "presentation",
			"style": inlineStyle{
				{Property: "float", Value: "none"},
				{Property: "display", Value: "inline-table"},
			}.InlineString(),
		}

		_, _ = io.WriteString(w, "<!--[if mso | IE]><td><![endif]-->")
		_, _ = io.WriteString(w, "<table "+childTdAttr.InlineString()+">\n")
		_, _ = io.WriteString(w, "<tbody>\n")

		var socialElement MJMLSocialElement
		if err := InitComponent(ctx, socialElement, child); err != nil {
			return err
		}
		if err := socialElement.Render(ctx, w, child); err != nil {
			return err
		}

		_, _ = io.WriteString(w, "</tbody>\n")
		_, _ = io.WriteString(w, "</table>\n")
		_, _ = io.WriteString(w, "<!--[if mso | IE]></td><![endif]-->")
	}

	_, _ = io.WriteString(w, "<!--[if mso | IE]></tr></table><![endif]-->")

	return nil
}

func (s MJMLSocial) renderVertical(ctx *RenderContext, w io.Writer, n *node.Node) error {
	tbAttr := inlineAttributes{
		"border":      "0",
		"cellpadding": "0",
		"cellspacing": "0",
		"role":        "presentation",
		"style": inlineStyle{
			{Property: "margin", Value: "0px"},
		}.InlineString(),
	}

	_, _ = io.WriteString(w, "<table "+tbAttr.InlineString()+">")
	_, _ = io.WriteString(w, "<tbody>")
	attr := s.getSocialElementAttributes(n)

	for _, child := range n.Children {
		if child.Type == RawTagName {
			var raw MJMLRaw
			if err := raw.Render(ctx, w, child); err != nil {
				return err
			}
		}

		if child.Type != SocialElementTagName {
			continue
		}

		for key, value := range attr {
			child.SetAttribute(key, value)
		}

		var socialElement MJMLSocialElement
		if err := InitComponent(ctx, socialElement, child); err != nil {
			return err
		}

		if err := socialElement.Render(ctx, w, child); err != nil {
			return err
		}
	}

	_, _ = io.WriteString(w, "</tbody>")
	_, _ = io.WriteString(w, "</table>")

	return nil
}

func (s MJMLSocial) getSocialElementAttributes(n *node.Node) map[string]string {
	toMatch := [...]string{
		"border-radius",
		"color",
		"font-family",
		"font-size",
		"font-weight",
		"font-style",
		"icon-size",
		"icon-height",
		"icon-padding",
		"text-padding",
		"line-height",
		"text-decoration",
	}

	matched := make(map[string]string, len(toMatch))

	for _, attr := range toMatch {
		v, ok := n.GetAttributeValue(attr)
		if !ok {
			continue
		}

		matched[attr] = v
	}

	if v, has := n.GetAttributeValue("inner-padding"); has {
		matched["padding"] = v
	}

	return matched
}
