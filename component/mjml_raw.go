package component

import (
	"io"
	"strings"

	"github.com/julez-dev/mjmlgo/node"
)

type MJMLRaw struct{}

func (r MJMLRaw) Name() string {
	return "mj-raw"
}

func (r MJMLRaw) AllowedAttributes() map[string]validateAttributeFunc {
	return make(map[string]validateAttributeFunc)
}

func (r MJMLRaw) DefaultAttributes(_ *RenderContext) map[string]string {
	return make(map[string]string)
}

func (r MJMLRaw) Render(_ *RenderContext, w io.Writer, n *node.Node) error {
	_, _ = io.WriteString(w, strings.TrimSpace(n.Content))
	return nil
}
