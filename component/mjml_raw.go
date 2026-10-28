package component

import (
	"io"
	"strings"

	"github.com/julez-dev/mjmlgo/node"
)

type MJMLRaw struct{}

// Raw block markers let the CSS inliner protect raw markup from HTML table
// foster-parenting. They are removed from the final output.
const (
	rawBlockStartMarker = "<!--mjmlgo-raw-start-->"
	rawBlockEndMarker   = "<!--mjmlgo-raw-end-->"
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
