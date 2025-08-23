package component

import (
	"fmt"
	"io"

	"github.com/aymerick/douceur/parser"
	"github.com/julez-dev/mjmlgo/node"
)

type MJMLHead struct{}

func (h MJMLHead) Name() string {
	return "mj-image"
}

func (h MJMLHead) AllowedAttributes() map[string]validateAttributeFunc {
	return make(map[string]validateAttributeFunc)
}

func (h MJMLHead) DefaultAttributes(_ *RenderContext) map[string]string {
	return make(map[string]string)
}

func (h MJMLHead) Render(ctx *RenderContext, w io.Writer, n *node.Node) error {
	var headStylesheets []Stylesheet
	var title string
	_, _ = io.WriteString(w, `<head>`)
	for _, child := range n.Children {
		switch child.Type {
		case StyleTagName:
			sheet, err := h.parseCSS(child.Content)
			if err != nil {
				return err
			}

			if v, found := child.GetAttributeValue("inline"); found && v == "inline" {
				ctx.InlineStyles = append(ctx.InlineStyles, sheet)
				continue
			}

			headStylesheets = append(headStylesheets, sheet)
		case TitleTagName:
			title = child.Content
		case RawTagName:
			var raw MJMLRaw
			if err := raw.Render(ctx, w, child); err != nil {
				return err
			}
		}
	}

	data := map[string]any{
		"Breakpoint":                  ctx.Breakpoint,
		"MJMLStyles":                  ctx.MJMLStylesheet,
		"UserStyles":                  headStylesheets,
		"IncludeMobileFullWidthStyle": ctx.IncludeMobileFullWidthStyle,
		"LowerBreakpoint":             ctx.makeLowerBreakpoint(),
		"Fonts":                       ctx.Fonts,
	}

	_, _ = io.WriteString(w, fmt.Sprintf("<title>%s</title>\n", title))
	if err := templates.ExecuteTemplate(w, "head-style-section.tmpl", data); err != nil {
		return fmt.Errorf("error executing head-style-section template: %w", err)
	}

	_, _ = io.WriteString(w, "</head>\n")

	return nil
}

// ParseCSS takes a raw CSS string and parses it into a slice of Style structs.
// It handles multiple CSS rules within the input string.
func (h MJMLHead) parseCSS(cssText string) (Stylesheet, error) {
	sheet, err := parser.Parse(cssText)
	if err != nil {
		return Stylesheet{}, fmt.Errorf("error parsing CSS: %w", err)
	}

	outSheet := Stylesheet{}

	for _, rule := range sheet.Rules {
		if rule.Name == "@media" {
			mediaRule := MediaRule{
				Condition: "@media " + rule.Prelude,
			}

			for _, childRule := range rule.Rules {
				decs := make([]Style, 0, len(childRule.Declarations))
				for _, dec := range childRule.Declarations {
					// Convert each declaration to a Style struct
					decs = append(decs, Style{
						Property:  dec.Property,
						Value:     dec.Value,
						Important: dec.Important,
					})
				}

				mediaRule.Rules = append(mediaRule.Rules, Rule{
					Selectors:    childRule.Prelude,
					Declarations: decs,
				})
			}

			outSheet.MediaRules = append(outSheet.MediaRules, mediaRule)
			continue
		}

		r := Rule{
			Selectors:    rule.Prelude,
			Declarations: make([]Style, 0, len(rule.Declarations)),
		}

		for _, dec := range rule.Declarations {
			// Convert each declaration to a Style struct
			r.Declarations = append(r.Declarations, Style{
				Property:  dec.Property,
				Value:     dec.Value,
				Important: dec.Important,
			})
		}

		outSheet.Rules = append(outSheet.Rules, r)
	}

	return outSheet, err
}
