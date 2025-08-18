package component

import (
	"io"

	"github.com/julez-dev/mjmlgo/node"
)

type MJMLSpacer struct{}

func (s MJMLSpacer) applyDefaults(n *node.Node) {
	defaults := map[string]string{
		"height": "20px",
	}

	for key, value := range defaults {
		if _, ok := n.GetAttributeValue(key); !ok {
			n.SetAttribute(key, value)
		}
	}
}

func (s MJMLSpacer) Render(ctx *RenderContext, w io.Writer, n *node.Node) error {
	s.applyDefaults(n)

	divStyle := inlineStyle{
		{Property: "height", Value: n.GetAttributeValueDefault("height")},
	}

	divAttr := inlineAttributes{
		"style": divStyle.InlineString(),
	}

	_, _ = io.WriteString(w, "<div "+divAttr.InlineString()+">&#8202;</div>")
	return nil
}
