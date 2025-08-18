package component

import (
	"io"

	"github.com/julez-dev/mjmlgo/node"
)

type MJMLAttributes struct{}

func (h MJMLAttributes) Render(ctx *RenderContext, w io.Writer, n *node.Node) error {
	for _, child := range n.Children {
		switch child.Type {
		case TextTagName:
			ctx.GlobalTextAttributes = append(ctx.GlobalTextAttributes, child.Attributes...)
		case AllTagName:
			ctx.GlobalAllAttributes = append(ctx.GlobalAllAttributes, child.Attributes...)
		}
	}

	return nil
}
