package component

import (
	"fmt"
	"io"

	"github.com/julez-dev/mjmlgo/node"
)

type MJMLText struct{}

func (t MJMLText) applyDefaults(n *node.Node) {
	defaults := map[string]string{
		"align":       "left",
		"color":       "#000000",
		"font-family": "Ubuntu, Helvetica, Arial, sans-serif",
		"font-size":   "13px",
		"line-height": "1",
		"padding":     "10px 25px",
	}

	for key, value := range defaults {
		if _, ok := n.GetAttributeValue(key); !ok {
			n.SetAttribute(key, value)
		}
	}
}

func (t MJMLText) Render(ctx *RenderContext, w io.Writer, n *node.Node) error {
	height, has := n.GetAttributeValue("height")

	if has {
		_, _ = io.WriteString(w, conditionalTag(fmt.Sprintf("<table role=\"presentation\" border=\"0\" cellpadding=\"0\" cellspacing=\"0\"><tr><td height=\"%s\" style=\"vertical-align:top;height:%s;\">", height, height), false))
		if err := t.renderContent(ctx, w, n); err != nil {
			return err
		}
		_, _ = io.WriteString(w, conditionalTag("</td></tr></table>", false))
		return nil
	}

	return t.renderContent(ctx, w, n)
}

func (t MJMLText) renderContent(_ *RenderContext, w io.Writer, n *node.Node) error {
	_, _ = io.WriteString(w, fmt.Sprintf("<div %s>", inlineAttributes{"style": t.getStyle(n).InlineString()}.InlineString()))
	_, _ = io.WriteString(w, n.Content)
	_, _ = io.WriteString(w, "</div>\n")
	return nil
}

func (t MJMLText) getStyle(n *node.Node) inlineStyle {
	style := inlineStyle{
		{Property: "font-family", Value: n.GetAttributeValueDefault("font-family")},
		{Property: "font-size", Value: n.GetAttributeValueDefault("font-size")},
		{Property: "font-style", Value: n.GetAttributeValueDefault("font-style")},
		{Property: "font-weight", Value: n.GetAttributeValueDefault("font-weight")},
		{Property: "letter-spacing", Value: n.GetAttributeValueDefault("letter-spacing")},
		{Property: "line-height", Value: n.GetAttributeValueDefault("line-height")},
		{Property: "text-align", Value: n.GetAttributeValueDefault("text-align")},
		{Property: "text-decoration", Value: n.GetAttributeValueDefault("text-decoration")},
		{Property: "text-transform", Value: n.GetAttributeValueDefault("text-transform")},
		{Property: "color", Value: n.GetAttributeValueDefault("color")},
		{Property: "height", Value: n.GetAttributeValueDefault("height")},
	}

	return style
}
