package customcomponent

import (
	"fmt"
	"io"
	"strings"

	"github.com/julez-dev/mjmlgo/v2/component"
	"github.com/julez-dev/mjmlgo/v2/node"
)

// NoticeTagName is the tag Notice registers.
const NoticeTagName = "mj-notice"

// Notice is a container: it declares mj-text children, renders them through
// component.RenderChildren into its own wrapper markup, and contributes CSS to
// the document head. It is the container counterpart of Badge.
type Notice struct{}

// Name returns the element's tag name.
func (Notice) Name() string {
	return NoticeTagName
}

// AllowedAttributes declares the attributes a document may write on
// <mj-notice>. css-class and mj-class are universal and need no entry.
func (Notice) AllowedAttributes() map[string]component.AttributeValidator {
	return map[string]component.AttributeValidator{
		"background-color": validateHexColor,
		"border-color":     validateHexColor,
	}
}

// DefaultAttributes returns the values applied when the author wrote none.
func (Notice) DefaultAttributes(*component.RenderContext) map[string]string {
	return map[string]string{
		"background-color": "#fef3c7",
		"border-color":     "#f59e0b",
	}
}

// AllowedChildren declares the only child Notice accepts. A child outside the
// list is a disallowed-child error carrying the element path, and registration
// rejects the component if a declared child is not a content component.
func (Notice) AllowedChildren() []string {
	return []string{component.TextTagName}
}

// Render writes the notice's wrapper around its rendered children and
// contributes the CSS the wrapper depends on to the head.
//
// The three head mechanisms a custom component has are all exercised here:
//
//   - AddComponentHeadStyle a named block, emitted once per document even when
//     the document holds several notices;
//   - AddMediaQuery a class rule inside the document's breakpoint media query,
//     with the Thunderbird, print and desktop-Outlook repeats MJML emits;
//   - AddFluidOnMobileClass the fluid-on-mobile rule for a class of its own.
func (Notice) Render(ctx *component.RenderContext, w io.Writer, n *node.Node) error {
	ctx.AddComponentHeadStyle(NoticeTagName, func(breakpoint string) string {
		return "    @media only screen and (min-width:" + breakpoint + ") {\n" +
			"        .mj-notice { padding: 16px !important; }\n" +
			"    }\n"
	})
	ctx.AddMediaQuery("mj-notice-inline", []string{"display: inline-block !important"})
	ctx.AddFluidOnMobileClass("mj-notice-fluid")

	// Render the children first, into a buffer this component owns, so the
	// wrapper markup is written around finished children. The renderer applies
	// each child's defaults and dispatches it, built-in or custom.
	var children strings.Builder
	if err := component.RenderChildren(ctx, &children, n); err != nil {
		return fmt.Errorf("failed to render the notice's children: %w", err)
	}

	_, err := fmt.Fprintf(
		w,
		`<table role="presentation" border="0" cellpadding="0" cellspacing="0" class="mj-notice mj-notice-fluid">`+
			`<tbody><tr><td align="left" style="background-color:%s;border:1px solid %s;border-radius:6px;`+
			`font-family:Helvetica,Arial,sans-serif;font-size:13px;line-height:1;padding:12px 16px;">%s</td></tr></tbody></table>`,
		n.GetAttributeValueDefault("background-color"),
		n.GetAttributeValueDefault("border-color"),
		children.String(),
	)

	return err
}
