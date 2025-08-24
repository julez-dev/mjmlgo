package mjmlgo

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/ericchiang/css"
	"github.com/julez-dev/mjmlgo/component"
	"golang.org/x/net/html"
)

var ErrUnknownStartingTag = errors.New("mjml: unknown starting tag")
var duplicateConditionalComments = regexp.MustCompile(`<!\[endif\]-->\s*<!--\[if mso \| IE\]>`)

func RenderMJML(input io.Reader) (string, error) {
	node, err := parse(input)
	if err != nil {
		return "", err
	}

	if node.Type != "mjml" {
		return "", fmt.Errorf("%w: %s", ErrUnknownStartingTag, node.Type)
	}

	var buff strings.Builder
	mjml := component.MJML{}

	ctx := &component.RenderContext{
		MJMLStylesheet: make(map[string][]string),
		Fonts:          make(map[string]string),
	}
	if err := component.InitComponent(ctx, mjml, node); err != nil {
		return "", err
	}

	if err := mjml.Render(ctx, &buff, node); err != nil {
		return "", err
	}

	//spew.Dump(ctx.InlineStyles)

	var out strings.Builder
	if err := inlineCSS(ctx, strings.NewReader(buff.String()), &out); err != nil {
		return "", err
	}

	return duplicateConditionalComments.ReplaceAllString(out.String(), ""), nil
}

func inlineCSS(ctx *component.RenderContext, r io.Reader, w io.Writer) error {
	htmlNode, err := html.Parse(r)
	if err != nil {
		return err
	}

	for _, sheet := range ctx.InlineStyles {
		for _, rule := range sheet.Rules {
			sel, err := css.Parse(rule.Selectors)
			if err != nil {
				continue
			}

			for _, n := range sel.Select(htmlNode) {
				var (
					styleIndex = -1
					styleAttr  html.Attribute
				)
				for i, attr := range n.Attr {
					if attr.Key == "style" {
						styleIndex = i
						styleAttr = attr
					}
				}

				styles, err := parseStyleAttribute(styleAttr)
				if err != nil {
					return fmt.Errorf("failed to parse style attribute: %w", err)
				}

				for _, dec := range rule.Declarations {
					if _, has := styles[dec.Property]; !has || dec.Important {
						styles[dec.Property] = dec.Value
					}
				}

				var styleText string
				for k, v := range styles {
					styleText += strings.TrimSpace(fmt.Sprintf("%s:%s;", k, v))
				}

				styleAttr.Val = styleText

				if n.Data == "table" || n.Data == "td" || n.Data == "div" {
					if v := styles["width"]; v != "" {
						var alreadyHasWidth bool
						for _, a := range n.Attr {
							if a.Key == "width" {
								alreadyHasWidth = true
								break
							}
						}
						if !alreadyHasWidth {
							if strings.HasSuffix(v, "px") {
								n.Attr = append(n.Attr, html.Attribute{Key: "width", Val: component.RemoveNonNumeric(v)})
							} else {
								n.Attr = append(n.Attr, html.Attribute{Key: "width", Val: v})
							}
						}
					}
				}

				if v := styles["text-align"]; v != "" {
					var alreadyHasAlign bool
					for _, a := range n.Attr {
						if a.Key == "align" {
							alreadyHasAlign = true
							break
						}
					}
					if !alreadyHasAlign {
						n.Attr = append(n.Attr, html.Attribute{Key: "align", Val: v})
					}
				}

				if v := styles["vertical-align"]; v != "" {
					var alreadyHasAlign bool
					for _, a := range n.Attr {
						if a.Key == "valign" {
							alreadyHasAlign = true
							break
						}
					}
					if !alreadyHasAlign {
						n.Attr = append(n.Attr, html.Attribute{Key: "valign", Val: v})
					}
				}

				if v := styles["background-color"]; v != "" {
					var alreadyHasBgColor bool
					for _, a := range n.Attr {
						if a.Key == "bgcolor" {
							alreadyHasBgColor = true
							break
						}
					}
					if !alreadyHasBgColor {
						n.Attr = append(n.Attr, html.Attribute{Key: "bgcolor", Val: styles["background-color"]})
					}
				}

				if styleIndex < 0 {
					n.Attr = append(n.Attr, html.Attribute{Key: "style", Val: styleAttr.Val})
				} else {
					n.Attr[styleIndex] = styleAttr
				}
			}
		}
	}

	if err := html.Render(w, htmlNode); err != nil {
		return err
	}

	return nil
}

func parseStyleAttribute(attr html.Attribute) (map[string]string, error) {
	styles := make(map[string]string)

	for style := range strings.SplitSeq(attr.Val, ";") {
		style = strings.TrimSpace(style)
		if style == "" {
			continue
		}
		parts := strings.SplitN(style, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid style declaration: %s", style)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" || value == "" {
			return nil, fmt.Errorf("invalid style declaration: %s", style)
		}
		styles[key] = value
	}

	return styles, nil
}
