package component

import (
	"io"

	"github.com/julez-dev/mjmlgo/node"
)

type MJMLSpacer struct{}

func (s MJMLSpacer) Name() string {
	return "mj-spacer"
}

func (s MJMLSpacer) AllowedAttributes() map[string]validateAttributeFunc {
	return map[string]validateAttributeFunc{
		"border":                     validateType("string"),
		"border-bottom":              validateType("string"),
		"border-left":                validateType("string"),
		"border-right":               validateType("string"),
		"border-top":                 validateType("string"),
		"container-background-color": validateColor(),
		"padding-bottom":             validateUnit([]string{"px", "%"}, false),
		"padding-left":               validateUnit([]string{"px", "%"}, false),
		"padding-right":              validateUnit([]string{"px", "%"}, false),
		"padding-top":                validateUnit([]string{"px", "%"}, false),
		"padding":                    validateUnit([]string{"px", "%"}, true),
		"height":                     validateUnit([]string{"px", "%"}, false),
	}
}

func (s MJMLSpacer) DefaultAttributes(ctx *RenderContext) map[string]string {
	return map[string]string{
		"height": "20px",
	}
}

func (s MJMLSpacer) Render(ctx *RenderContext, w io.Writer, n *node.Node) error {
	divStyle := inlineStyle{
		{Property: "height", Value: n.GetAttributeValueDefault("height")},
		{Property: "line-height", Value: n.GetAttributeValueDefault("height")},
	}

	divAttr := inlineAttributes{
		"style": divStyle.InlineString(),
	}

	_, _ = io.WriteString(w, "<div "+divAttr.InlineString()+">&#8202;</div>")
	return nil
}
