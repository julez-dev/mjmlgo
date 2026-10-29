package customcomponent

import (
	"errors"
	"fmt"
	"html"
	"io"
	"regexp"
	"strings"

	"github.com/julez-dev/mjmlgo/v2/component"
	"github.com/julez-dev/mjmlgo/v2/node"
)

// BadgeTagName is the tag Badge registers.
const BadgeTagName = "mj-badge"

var hexColor = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)

// Badge is a small label rendered as a single table cell. It is a leaf: a child
// element under <mj-badge> is a disallowed child, because Badge does not
// implement component.ChildComponent.
type Badge struct{}

// Name returns the element's tag name.
func (Badge) Name() string {
	return BadgeTagName
}

// AllowedAttributes declares the attributes a document may write on <mj-badge>.
// css-class and mj-class are universal and need no entry.
func (Badge) AllowedAttributes() map[string]component.AttributeValidator {
	return map[string]component.AttributeValidator{
		"text":             validateText,
		"background-color": validateHexColor,
		"color":            validateHexColor,
	}
}

// DefaultAttributes returns the values applied when the author wrote none.
func (Badge) DefaultAttributes(*component.RenderContext) map[string]string {
	return map[string]string{
		"background-color": "#1f2937",
		"color":            "#ffffff",
	}
}

// Render writes the badge's HTML. The renderer has already validated the author
// attributes and applied both the component's defaults and the document's
// mj-attributes/mj-class defaults, so every accessor below sees the effective
// value.
func (b Badge) Render(_ *component.RenderContext, w io.Writer, n *node.Node) error {
	classAttribute := ""
	if class := n.GetAttributeValueDefault("css-class"); class != "" {
		classAttribute = fmt.Sprintf(` class="%s"`, html.EscapeString(class))
	}

	_, err := fmt.Fprintf(
		w,
		`<table role="presentation" border="0" cellpadding="0" cellspacing="0"%s>`+
			`<tbody><tr><td align="center" style="background-color:%s;border-radius:4px;color:%s;`+
			`font-family:Helvetica,Arial,sans-serif;font-size:13px;line-height:1;padding:8px 12px;">`+
			`%s</td></tr></tbody></table>`,
		classAttribute,
		n.GetAttributeValueDefault("background-color"),
		n.GetAttributeValueDefault("color"),
		html.EscapeString(n.GetAttributeValueDefault("text")),
	)

	return err
}

// validateText rejects an empty badge label: a badge without text renders an
// empty pill, which is always an authoring mistake.
func validateText(value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New("badge text must not be empty")
	}

	return nil
}

// validateHexColor accepts the hex colour forms MJML's color type allows for
// colours written in CSS, which is all the example needs.
func validateHexColor(value string) error {
	if !hexColor.MatchString(value) {
		return fmt.Errorf("%q is not a hex colour such as #1f2937", value)
	}

	return nil
}
