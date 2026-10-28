package component

import (
	"fmt"
	"io"
	"strings"

	"github.com/julez-dev/mjmlgo/node"
)

type MJMLRaw struct{}

// RawBlockStartMarker and RawBlockEndMarker wrap raw column content so the CSS
// inliner can protect it from HTML table foster-parenting. They are removed from
// the final output by StripRawBlockMarkers.
const (
	RawBlockStartMarker = "<!--mjmlgo-raw-start-->"
	RawBlockEndMarker   = "<!--mjmlgo-raw-end-->"
)

func (r MJMLRaw) Name() string {
	return "mj-raw"
}

func (r MJMLRaw) AllowedAttributes() map[string]AttributeValidator {
	return map[string]AttributeValidator{
		"position": validateEnum([]string{"file-start"}),
	}
}

func (r MJMLRaw) DefaultAttributes(_ *RenderContext) map[string]string {
	return make(map[string]string)
}

func (r MJMLRaw) Render(_ *RenderContext, w io.Writer, n *node.Node) error {
	_, _ = io.WriteString(w, strings.TrimSpace(n.Content))
	return nil
}

// renderRawChild renders a raw child - an <mj-raw> element, or an author
// comment, which the parser represents as one - at its document position.
//
// It is the single place that renders raw children, so the per-container child
// loops cannot drift apart. Containers whose raw content sits directly inside a
// <table> body use renderRawChildInTableBody instead. A custom component chooses
// between the two through RenderChildren and RenderChildrenInTableBody, which
// call this function and renderRawChildInTableBody respectively.
func renderRawChild(ctx *RenderContext, w io.Writer, n *node.Node) error {
	var raw MJMLRaw
	if err := InitComponent(ctx, raw, n); err != nil {
		return err
	}

	if err := raw.Render(ctx, w, n); err != nil {
		return fmt.Errorf("failed to render raw: %w", err)
	}

	return nil
}

// renderRawChildInTableBody is renderRawChild for raw content that sits
// directly inside a <tbody>. The block is wrapped in the raw-block markers
// because the CSS inliner parses the document with html.Parse, whose table
// rules would otherwise foster-parent the content out of the table and
// re-serialize it. The markers are stripped again after inlining (and by
// StripRawBlockMarkers for documents that skip the inliner).
//
// Containers that emit raw content outside a table body - the body div, the
// head, the navbar, the horizontal social layout, the section, group and
// wrapper cell content, the accordion element content - must use renderRawChild:
// marking them would keep the inliner from applying inline styles to content it
// can safely re-serialize. A custom container makes the same choice through
// RenderChildren (no marker, any markup) or RenderChildrenInTableBody (marked,
// children directly inside a <tbody>).
func renderRawChildInTableBody(ctx *RenderContext, w io.Writer, n *node.Node) error {
	if _, err := io.WriteString(w, RawBlockStartMarker); err != nil {
		return err
	}

	if err := renderRawChild(ctx, w, n); err != nil {
		return err
	}

	_, err := io.WriteString(w, RawBlockEndMarker)

	return err
}

// StripRawBlockMarkers removes the raw-block markers from rendered HTML. The
// inliner already restores protected blocks, but documents that skip the
// inlining pass (no inline styles, no <mj-html-attributes>) still carry them.
func StripRawBlockMarkers(html string) string {
	if !strings.Contains(html, RawBlockStartMarker) && !strings.Contains(html, RawBlockEndMarker) {
		return html
	}

	html = strings.ReplaceAll(html, RawBlockStartMarker, "")
	html = strings.ReplaceAll(html, RawBlockEndMarker, "")

	return html
}
