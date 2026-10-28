package component

import (
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

func (r MJMLRaw) AllowedAttributes() map[string]validateAttributeFunc {
	return map[string]validateAttributeFunc{
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
