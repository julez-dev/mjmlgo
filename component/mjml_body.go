package component

import (
	"fmt"
	"io"

	"github.com/julez-dev/mjmlgo/node"
)

type MJMLBody struct{}

func (b MJMLBody) Name() string {
	return "mj-body"
}

func (b MJMLBody) AllowedAttributes() map[string]validateAttributeFunc {
	return map[string]validateAttributeFunc{
		"background-color": validateColor(),
		"width":            validateUnit([]string{"px"}, false),
	}
}

func (b MJMLBody) DefaultAttributes(_ *RenderContext) map[string]string {
	return map[string]string{
		"width": "600px",
	}
}

func (b MJMLBody) Render(ctx *RenderContext, w io.Writer, n *node.Node) error {
	bodyInlineStyles := inlineStyle{
		{Property: "word-spacing", Value: "normal"},
	}

	if bg, ok := n.GetAttributeValue("background-color"); ok {
		bodyInlineStyles = append(bodyInlineStyles, Style{Property: "background-color", Value: bg})
	}

	if maxWidth, ok := n.GetAttributeValue("width"); ok {
		ctx.ContainerWidth = maxWidth
	}

	bodyDivName, _ := n.GetAttributeValue("css-class")

	_, _ = io.WriteString(w, fmt.Sprintf("<body style=\"%s\">\n", bodyInlineStyles.InlineString()))
	if ctx.PreviewText != "" {
		if err := templates.ExecuteTemplate(w, "preview-text.tmpl", ctx.PreviewText); err != nil {
			return err
		}
	}

	_, _ = io.WriteString(w, fmt.Sprintf("<div %s>\n", inlineAttributes{
		"lang":  ctx.Language,
		"dir":   ctx.Direction,
		"class": bodyDivName,
		"style": inlineStyle{{Property: "background-color", Value: n.GetAttributeValueDefault("background-color")}}.InlineString()}.InlineString()),
	)

	for _, child := range n.Children {
		switch child.Type {
		case RawTagName:
			var raw MJMLRaw
			if err := InitComponent(ctx, raw, child); err != nil {
				return err
			}
			if err := raw.Render(ctx, w, child); err != nil {
				return err
			}
		case SectionTagName:
			var section MJMLSection
			if err := InitComponent(ctx, section, child); err != nil {
				return err
			}
			if err := section.Render(ctx, w, child); err != nil {
				return err
			}
		case WrapperTagName:
			var section MJMLSection
			section.IsWrapper = true
			if err := InitComponent(ctx, section, child); err != nil {
				return err
			}
			if err := section.Render(ctx, w, child); err != nil {
				return err
			}
		}
	}

	_, _ = io.WriteString(w, "</div>\n")
	_, _ = io.WriteString(w, "</body>\n")
	return nil
}
