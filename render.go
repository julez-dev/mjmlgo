package mjmlgo

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"regexp"
	"strings"

	"github.com/ericchiang/css"
	"github.com/julez-dev/mjmlgo/internal/component"
	"golang.org/x/net/html"
)

var ErrUnknownStartingTag = errors.New("mjml: unknown starting tag")

// defaultRenderer backs RenderMJML. It has no mutable per-render state, so a
// single shared instance is safe for concurrent use.
var defaultRenderer = NewRenderer()

// RenderMJML renders the MJML document read from input and returns the
// resulting HTML.
//
// Deprecated: use NewRenderer and Renderer.Render, Renderer.RenderString,
// Renderer.RenderBytes or Renderer.RenderFile instead. RenderMJML is a thin
// wrapper around the default Renderer; it has no filesystem, so documents that
// use mj-include are rejected with an error.
func RenderMJML(input io.Reader) (string, error) {
	return defaultRenderer.renderDocument(nil, "", input)
}

// renderDocument is the single rendering pipeline shared by every entry point.
// fsys and name identify the document for mj-include resolution; fsys is nil
// when the caller has no filesystem of its own.
func (r *Renderer) renderDocument(fsys fs.FS, name string, input io.Reader) (string, error) {
	node, err := parse(input)
	if err != nil {
		return "", err
	}

	if node.Type != "mjml" {
		return "", fmt.Errorf("%w: %s", ErrUnknownStartingTag, node.Type)
	}

	if err := resolveIncludes(fsys, name, node); err != nil {
		return "", err
	}

	var buff strings.Builder
	mjml := component.MJML{}

	ctx := &component.RenderContext{
		MJMLStylesheet: make(map[string][]string),
		Fonts:          make(map[string]string),
		IDGenerator:    r.idFunc(),
	}
	if err := component.InitComponent(ctx, mjml, node); err != nil {
		return "", err
	}

	if err := mjml.Render(ctx, &buff, node); err != nil {
		return "", err
	}

	rendered := buff.String()

	// The HTML round-trip below only exists to apply inline styles and
	// <mj-html-attributes>; documents that use neither keep the rendered
	// markup untouched, which avoids parsing and re-rendering the whole
	// document for nothing.
	if len(ctx.InlineStyles) > 0 || len(ctx.HTMLAttributes) > 0 {
		var inlined strings.Builder
		if err := inlineCSS(ctx, strings.NewReader(rendered), &inlined); err != nil {
			return "", err
		}

		rendered = inlined.String()
	}

	// The inliner restores protected raw blocks; documents that skip it still
	// carry the markers, so strip them either way.
	rendered = component.StripRawBlockMarkers(rendered)

	// MJML merges adjacent Outlook conditional blocks after rendering, so
	// contiguous MSO fragments form a single conditional comment.
	html := component.MergeOutlookConditionals(rendered)

	if beforeDoctype := fileStartRawContent(node); beforeDoctype != "" {
		return beforeDoctype + "\n" + html, nil
	}

	return html, nil
}

// rawBlockPlaceholder stands in for a protected raw block while the document is
// parsed for CSS inlining.
const rawBlockPlaceholder = "<!--mjmlgo-raw-placeholder-%d-->"

var rawBlockMarkerPattern = regexp.MustCompile(`(?s)` + regexp.QuoteMeta(component.RawBlockStartMarker) + `(.*?)` + regexp.QuoteMeta(component.RawBlockEndMarker))

// protectRawBlocks replaces raw blocks with comment placeholders so the HTML
// parser cannot move them out of table elements. The original blocks are
// restored after serialization.
func protectRawBlocks(input string) (string, []string) {
	var blocks []string

	protected := rawBlockMarkerPattern.ReplaceAllStringFunc(input, func(match string) string {
		content := strings.TrimSuffix(strings.TrimPrefix(match, component.RawBlockStartMarker), component.RawBlockEndMarker)
		blocks = append(blocks, content)

		return fmt.Sprintf(rawBlockPlaceholder, len(blocks)-1)
	})

	return protected, blocks
}

func restoreRawBlocks(output string, blocks []string) string {
	for i, content := range blocks {
		output = strings.Replace(output, fmt.Sprintf(rawBlockPlaceholder, i), content, 1)
	}

	return output
}

func inlineCSS(ctx *component.RenderContext, r io.Reader, w io.Writer) error {
	input, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	protected, rawBlocks := protectRawBlocks(string(input))

	htmlNode, err := html.Parse(strings.NewReader(protected))
	if err != nil {
		return err
	}

	applyHTMLAttributes(ctx.HTMLAttributes, htmlNode)

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

	var rendered strings.Builder
	if err := html.Render(&rendered, htmlNode); err != nil {
		return err
	}

	_, err = io.WriteString(w, restoreRawBlocks(rendered.String(), rawBlocks))
	return err
}

// applyHTMLAttributes runs the <mj-html-attributes> selector pass on the parsed
// document before inline styles are applied, mirroring MJML's pipeline. MJML
// runs the pass against the rendered body content, so only the body subtree is
// considered and the body element itself is not a target.
func applyHTMLAttributes(selectors []component.HTMLAttributeSelector, doc *html.Node) {
	if len(selectors) == 0 {
		return
	}

	body := findHTMLElement(doc, "body")
	if body == nil {
		return
	}

	for _, htmlSelector := range selectors {
		sel, err := css.Parse(htmlSelector.Path)
		if err != nil {
			continue
		}

		for _, target := range sel.Select(body) {
			if target == body {
				continue
			}

			for _, htmlAttribute := range htmlSelector.Attributes {
				setHTMLAttribute(target, htmlAttribute.Name, htmlAttribute.Value)
			}
		}
	}
}

func findHTMLElement(n *html.Node, tag string) *html.Node {
	if n.Type == html.ElementNode && n.Data == tag {
		return n
	}

	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if found := findHTMLElement(child, tag); found != nil {
			return found
		}
	}

	return nil
}

func setHTMLAttribute(n *html.Node, name, value string) {
	for i := range n.Attr {
		if n.Attr[i].Key == name {
			n.Attr[i].Val = value
			return
		}
	}

	n.Attr = append(n.Attr, html.Attribute{Key: name, Val: value})
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
