package component

import (
	"fmt"
	"io"
	"strings"

	"github.com/julez-dev/mjmlgo/node"
)

type MJMLRaw struct{}

// RawBlockStartMarker and RawBlockEndMarker wrap raw content that a container
// renders directly inside a <tbody>, so its position stays explicit in the
// markup. They are removed from the final output by StripRawBlockMarkers.
const (
	RawBlockStartMarker = "<!--mjmlgo-raw-start-->"
	RawBlockEndMarker   = "<!--mjmlgo-raw-end-->"
)

func (r MJMLRaw) Name() string {
	return "mj-raw"
}

var rawAllowedAttributes = map[string]AttributeValidator{
	"position": validateEnum([]string{"file-start"}),
}

func (r MJMLRaw) AllowedAttributes() map[string]AttributeValidator {
	return rawAllowedAttributes
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
// loops cannot drift apart. Raw content is emitted verbatim, wherever it sits:
// the inline stage matches selectors against the emitted markup itself, so no
// parser can move a block out of a table and nothing needs re-serializing.
//
// A custom component chooses between this and renderRawChildInTableBody through
// RenderChildren and RenderChildrenInTableBody.
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

// renderRawChildInTableBody is renderRawChild for raw content that sits directly
// inside a <tbody>: it wraps the block in the raw-block comments, so the position
// the author wrote the content in is explicit in the markup.
//
// The markers no longer protect anything - the inline stage builds its tree from
// the emitted markup and styles a raw block in place, exactly where it stands -
// but they still name the container's position contract, and the renderer strips
// them from the finished document before the caller sees it. A custom container
// makes the same choice through RenderChildren (no marker, any position) or
// RenderChildrenInTableBody (marked, children directly inside a <tbody>).
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

// StripRawBlockMarkers removes the raw-block markers from rendered HTML. They are
// internal transport for RenderChildrenInTableBody and are not output.
func StripRawBlockMarkers(html string) string {
	if !strings.Contains(html, RawBlockStartMarker) && !strings.Contains(html, RawBlockEndMarker) {
		return html
	}

	html = strings.ReplaceAll(html, RawBlockStartMarker, "")
	html = strings.ReplaceAll(html, RawBlockEndMarker, "")

	return html
}
