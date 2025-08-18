package mjmlgo

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/ericchiang/css"
	"github.com/julez-dev/mjmlgo/component"
	"golang.org/x/net/html"
)

var ErrUnknownStartingTag = errors.New("mjml: unknown starting tag")

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
	}
	ctx.Breakpoint = "320px"
	ctx.ContainerWidth = "600px"
	ctx.Direction = "ltr"

	if err := mjml.Render(ctx, &buff, node); err != nil {
		return "", err
	}

	//spew.Dump(ctx.InlineStyles)

	var out strings.Builder
	if err := inlineCSS(ctx, strings.NewReader(buff.String()), &out); err != nil {
		return "", err
	}

	return out.String(), nil
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

				for _, dec := range rule.Declarations {
					var decAsText string
					if dec.Important {
						decAsText += fmt.Sprintf("%s: %s !important;", dec.Property, dec.Value)
					} else {
						decAsText += fmt.Sprintf("%s: %s;", dec.Property, dec.Value)
					}
					styleAttr.Val += decAsText
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
