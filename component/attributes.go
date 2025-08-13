package component

import (
	"io"

	"github.com/julez-dev/mjmlgo/node"
)

type Attributes struct{}

func (h Attributes) Render(ctx *RenderContext, w io.Writer, n *node.Node) error {
	for _, child := range n.Children {
		switch child.Type {
		case TextTagName:
			ctx.GlobalTextAttibutes = append(ctx.GlobalTextAttibutes, child.Attributes)
		case ClassTagName:
			ctx.GlobalMJClassAttributes = append(ctx.GlobalMJClassAttributes, child.Attributes)
		case AllTagName:
			ctx.GlobalAllAttibutes = append(ctx.GlobalAllAttibutes, child.Attributes)
		}
	}

	return nil
}
