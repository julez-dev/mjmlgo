package component

import (
	"fmt"
	"io"

	"github.com/aymerick/douceur/css"
	"github.com/aymerick/douceur/parser"
	"github.com/julez-dev/mjmlgo/node"
)

type MJMLHead struct{}

func (h MJMLHead) Name() string {
	return "mj-head"
}

func (h MJMLHead) AllowedAttributes() map[string]AttributeValidator {
	return make(map[string]AttributeValidator)
}

func (h MJMLHead) DefaultAttributes(_ *RenderContext) map[string]string {
	return make(map[string]string)
}

func (h MJMLHead) Render(ctx *RenderContext, w io.Writer, n *node.Node) error {
	var userStyles []string
	var title string
	_, _ = io.WriteString(w, `<head>`)
	for _, child := range n.Children {
		switch child.Type {
		case StyleTagName:
			// MJML adds <mj-style> content to the head verbatim; only
			// inline="inline" styles are parsed (for the inliner).
			if v, found := child.GetAttributeValue("inline"); found && v == "inline" {
				sheet, err := h.parseCSS(child.Content)
				if err != nil {
					return err
				}

				ctx.InlineStyles = append(ctx.InlineStyles, sheet)
				continue
			}

			userStyles = append(userStyles, child.Content)
		case TitleTagName:
			title = child.Content
		}
	}

	data := map[string]any{
		"Breakpoint":                  ctx.Breakpoint,
		"MJMLStyles":                  ctx.MJMLStylesheet,
		"MJMLStyleOrder":              ctx.orderedMJMLStyles(),
		"ComponentHeadStyles":         ctx.renderedComponentHeadStyles(),
		"InstanceComponentHeadStyles": ctx.renderedInstanceComponentHeadStyles(),
		"UserStyles":                  userStyles,
		"FluidOnMobileClasses":        ctx.fluidOnMobileRuleClasses(),
		"ForceOWADesktop":             ctx.ForceOWADesktop,
		"PrinterSupport":              ctx.PrinterSupport,
		"LowerBreakpoint":             ctx.makeLowerBreakpoint(),
		"Fonts":                       ctx.Fonts,
		"UsedFonts":                   ctx.usedFonts(ctx.BodyContent, ctx.InlineStyles),
	}

	_, _ = io.WriteString(w, fmt.Sprintf("<title>%s</title>\n", title))
	if err := templates.ExecuteTemplate(w, "head-style-section.tmpl", data); err != nil {
		return fmt.Errorf("error executing head-style-section template: %w", err)
	}

	for _, child := range n.Children {
		if child.Type != RawTagName {
			continue
		}
		if err := renderRawChild(ctx, w, child); err != nil {
			return err
		}

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
				mediaRule.Rules = append(mediaRule.Rules, newStylesheetRule(childRule.Prelude, childRule.Declarations))
			}

			outSheet.MediaRules = append(outSheet.MediaRules, mediaRule)
			continue
		}

		outSheet.Rules = append(outSheet.Rules, newStylesheetRule(rule.Prelude, rule.Declarations))
	}

	return outSheet, err
}

// newStylesheetRule converts one parsed CSS rule and compiles its selector list
// once, so the inliner can select nodes without re-parsing selectors per render.
// A rule whose selectors do not compile carries no compiled selector and is
// skipped by the inliner.
func newStylesheetRule(prelude string, declarations []*css.Declaration) Rule {
	r := Rule{
		Selectors:    prelude,
		Declarations: make([]Style, 0, len(declarations)),
	}

	for _, dec := range declarations {
		r.Declarations = append(r.Declarations, Style{
			Property:  dec.Property,
			Value:     dec.Value,
			Important: dec.Important,
		})
	}

	r.CompileSelectors()

	return r
}
