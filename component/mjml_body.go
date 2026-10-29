package component

import (
	"fmt"
	"io"

	"github.com/julez-dev/mjmlgo/v2/node"
)

type MJMLBody struct{}

func (b MJMLBody) Name() string {
	return "mj-body"
}

var bodyAllowedAttributes = map[string]AttributeValidator{
	"background-color": validateColor(),
	"id":               validateType("string"),
	"width":            validateUnit([]string{"px"}, false),
}

func (b MJMLBody) AllowedAttributes() map[string]AttributeValidator {
	return bodyAllowedAttributes
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

	bodyAttr := inlineAttributes{
		"id":    n.GetAttributeValueDefault("id"),
		"class": n.GetAttributeValueDefault("css-class"),
		"style": bodyInlineStyles.InlineString(),
	}

	_, _ = io.WriteString(w, "<body "+bodyAttr.InlineString()+">\n")
	if ctx.PreviewText != "" {
		if err := templates.ExecuteTemplate(w, "preview-text.tmpl", ctx.PreviewText); err != nil {
			return err
		}
	}

	// The inner div keeps the background color but, since MJML 5, not the
	// author's class: mj-body's css-class lands on <body>.
	divStyle := inlineStyle{
		{Property: "word-spacing", Value: "normal"},
		{Property: "background-color", Value: n.GetAttributeValueDefault("background-color")},
	}

	_, _ = io.WriteString(w, fmt.Sprintf("<div %s>\n", inlineAttributes{
		"aria-label":           ctx.Title,
		"aria-roledescription": "email",
		"role":                 "article",
		"lang":                 ctx.Language,
		"dir":                  ctx.Direction,
		"style":                divStyle.InlineString()}.InlineString()),
	)

	for _, child := range n.Children {
		switch child.Type {
		case RawTagName:
			if err := renderRawChild(ctx, w, child); err != nil {
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
		case HeroTagName:
			var hero MJMLHero
			if err := InitComponent(ctx, hero, child); err != nil {
				return err
			}
			if err := hero.Render(ctx, w, child); err != nil {
				return err
			}
		default:
			custom, isCustom := customComponent(ctx, child.Type)
			if !isCustom {
				// Unreachable while ValidateDocument runs first: every other child
				// of mj-body is a disallowed-child error before rendering. Report
				// the broken invariant as internal, not as author input.
				return fmt.Errorf("no component for allowed child <%s> in <mj-body>", child.Type)
			}

			if err := InitComponent(ctx, custom, child); err != nil {
				return err
			}
			if err := custom.Render(ctx, w, child); err != nil {
				return fmt.Errorf("failed to render <%s>: %w", child.Type, err)
			}
		}
	}

	_, _ = io.WriteString(w, "</div>\n")
	_, _ = io.WriteString(w, "</body>\n")
	return nil
}
