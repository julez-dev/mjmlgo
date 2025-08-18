package component

import (
	"io"
	"strings"

	"github.com/julez-dev/mjmlgo/node"
)

type MJMLRaw struct{}

func (r MJMLRaw) Render(_ *RenderContext, w io.Writer, n *node.Node) error {
	_, _ = io.WriteString(w, strings.TrimSpace(n.Content))
	return nil
}
