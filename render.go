package mjmlgo

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"regexp"
	"strconv"
	"strings"

	"github.com/ericchiang/css"
	"github.com/julez-dev/mjmlgo/component"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// ErrUnknownStartingTag reports a document whose root element is not <mjml>.
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
	return defaultRenderer.renderDocument(context.Background(), nil, "", input)
}

// renderDocument is the single rendering pipeline shared by every entry point.
// fsys and name identify the document for mj-include resolution; fsys is nil
// when the caller has no filesystem of its own.
//
// Cancellation is checked between stages: parsing, include expansion, rendering
// and inlining all happen in memory, so a cancelled context cannot interrupt
// them part-way, but a caller is not left waiting for the remaining stages.
func (r *Renderer) renderDocument(callCtx context.Context, fsys fs.FS, name string, input io.Reader) (string, error) {
	if err := callCtx.Err(); err != nil {
		return "", err
	}

	node, err := parse(input, !r.dropComments)
	if err != nil {
		return "", err
	}

	if node.Type != "mjml" {
		return "", component.NewError(KindUnknownElement, []string{node.Type}, "", node.Type, ErrUnknownStartingTag)
	}

	if err := callCtx.Err(); err != nil {
		return "", err
	}

	if r.ignoreIncludes {
		stripIncludes(node)
	} else if err := resolveIncludes(callCtx, fsys, name, node, !r.dropComments); err != nil {
		return "", err
	}

	if err := callCtx.Err(); err != nil {
		return "", err
	}

	var buff strings.Builder
	mjml := component.MJML{}

	ctx := &component.RenderContext{
		MJMLStylesheet: make(map[string][]string),
		Fonts:          r.fontRegistry(),
		// A caller-provided registry replaces the built-in one instead of
		// being extended with it.
		FontsConfigured: r.fonts != nil,
		PrinterSupport:  r.printerSupport,
		IDGenerator:     r.idFunc(),
	}
	if err := component.InitComponent(ctx, mjml, node); err != nil {
		return "", err
	}

	if err := mjml.Render(ctx, &buff, node); err != nil {
		return "", err
	}

	if err := callCtx.Err(); err != nil {
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

// fontRegistry returns the web-font registry to seed the render context with.
// The renderer's map is copied because rendering registers <mj-font> families in
// the context and a Renderer may be used concurrently.
func (r *Renderer) fontRegistry() map[string]string {
	if r.fonts == nil {
		return make(map[string]string)
	}

	registry := make(map[string]string, len(r.fonts))
	for name, href := range r.fonts {
		registry[name] = href
	}

	return registry
}

// Placeholders stand in for protected raw blocks while the document is parsed
// for CSS inlining. The index of the block is embedded in the comment.
const (
	rawBlockPlaceholderPrefix = "<!--mjmlgo-raw-placeholder-"
	rawBlockPlaceholderSuffix = "-->"
)

// rawBlockPlaceholderComment returns the placeholder comment that stands for
// block i in the document handed to the HTML parser.
func rawBlockPlaceholderComment(i int) string {
	return rawBlockPlaceholderPrefix + strconv.Itoa(i) + rawBlockPlaceholderSuffix
}

var rawBlockMarkerPattern = regexp.MustCompile(`(?s)` + regexp.QuoteMeta(component.RawBlockStartMarker) + `(.*?)` + regexp.QuoteMeta(component.RawBlockEndMarker))

// protectRawBlocks replaces raw blocks with comment placeholders so the HTML
// parser cannot move them out of table elements. The original blocks are
// restored after serialization. Documents without raw blocks - the common case -
// are returned as they are, without scanning for markers.
func protectRawBlocks(input string) (string, []string) {
	if !strings.Contains(input, component.RawBlockStartMarker) {
		return input, nil
	}

	var blocks []string

	protected := rawBlockMarkerPattern.ReplaceAllStringFunc(input, func(match string) string {
		content := strings.TrimSuffix(strings.TrimPrefix(match, component.RawBlockStartMarker), component.RawBlockEndMarker)
		blocks = append(blocks, content)

		return rawBlockPlaceholderComment(len(blocks) - 1)
	})

	return protected, blocks
}

// writeRestoredRawBlocks writes output with every placeholder replaced by the
// raw block it stands for, directly into w: a per-block strings.Replace would
// copy the whole document once per protected block, and an intermediate string
// would copy it one more time.
func writeRestoredRawBlocks(w io.Writer, output string, blocks []string) error {
	if len(blocks) == 0 {
		_, err := io.WriteString(w, output)

		return err
	}

	write := func(s string) error {
		_, err := io.WriteString(w, s)

		return err
	}

	rest := output
	for {
		start := strings.Index(rest, rawBlockPlaceholderPrefix)
		if start < 0 {
			return write(rest)
		}

		if err := write(rest[:start]); err != nil {
			return err
		}

		rest = rest[start+len(rawBlockPlaceholderPrefix):]

		end := strings.Index(rest, rawBlockPlaceholderSuffix)
		if end < 0 {
			// A prefix without a suffix is not one of our placeholders.
			return write(rawBlockPlaceholderPrefix + rest)
		}

		index, err := strconv.Atoi(rest[:end])
		if err != nil || index < 0 || index >= len(blocks) {
			// Leave an unknown placeholder alone rather than losing content.
			if err := write(rawBlockPlaceholderPrefix + rest[:end+len(rawBlockPlaceholderSuffix)]); err != nil {
				return err
			}

			rest = rest[end+len(rawBlockPlaceholderSuffix):]

			continue
		}

		if err := write(blocks[index]); err != nil {
			return err
		}

		rest = rest[end+len(rawBlockPlaceholderSuffix):]
	}
}

// styleRawBlocks applies the inline stylesheets and the <mj-html-attributes>
// selectors to the content of protected raw blocks and replaces each block with
// its styled, re-serialized form.
//
// A block is parsed as a fragment rather than as part of the document, because
// the markers deliberately keep it out of the whole-document parse so that the
// HTML parser cannot foster-parent it. The fragment's context comes from the
// block's own content (see rawBlockFragmentContext): a marked block always sits
// inside a table body, and parsing its content in "in table body" mode is not
// lossless - a <table> start tag there is treated as an end tag, which would
// delete an author's nested layout table - while a generic context would drop
// bare rows and cells.
//
// Two limits follow from the fragment scope, and both are documented rather than
// worked around:
//
//   - a selector that depends on an ancestor outside the block (`.hero .raw`)
//     cannot match inside it, because those ancestors are not part of the
//     fragment. Selectors that match the block's own elements - the common case
//     for raw content - apply exactly as they do in the document;
//   - a block that mixes table shapes cannot round-trip through any context:
//     a <tr> before a <table> loses the nested table in table-section mode and
//     the row in generic mode, and a bare table child (a section, caption or
//     column group) followed by rows is invalid nesting inside the surrounding
//     table body, which the reference's own parser normalizes differently.
func styleRawBlocks(ctx *component.RenderContext, blocks []string) error {
	if len(blocks) == 0 || (len(ctx.HTMLAttributes) == 0 && len(ctx.InlineStyles) == 0) {
		return nil
	}

	for i := range blocks {
		nodes, err := html.ParseFragment(strings.NewReader(blocks[i]), rawBlockFragmentContext(blocks[i]))
		if err != nil {
			return fmt.Errorf("parse raw block: %w", err)
		}

		// The document pass runs the attribute selectors before the inline
		// styles so that a rule matching an injected attribute sees it; the
		// block pass keeps that order.
		changed := applyHTMLAttributesIn(ctx.HTMLAttributes, nodes, nil)

		inlined, err := inlineStylesIn(ctx.InlineStyles, nodes)
		if err != nil {
			return err
		}

		if !changed && !inlined {
			continue
		}

		var styled strings.Builder
		styled.Grow(len(blocks[i]) + len(blocks[i])/8)

		for _, n := range nodes {
			if err := html.Render(&styled, n); err != nil {
				return err
			}
		}

		blocks[i] = styled.String()
	}

	return nil
}

// rawBlockFragmentContext returns the parsing context for a protected raw block,
// chosen from the block's own content.
//
// The author's markup decides, because no single context is lossless for every
// shape a raw block can have:
//
//   - a bare row or cell is table-section content. A table body context keeps it
//     a row instead of dropping it, and adds no wrapper the author did not
//     write: the HTML parser inserts the same rows when it parses the reference's
//     output;
//   - a table section, caption or column group is a table child, which a table
//     body context would drop and a table context keeps;
//   - everything else - including a nested <table>, which "in table body" mode
//     would treat as an end tag - is parsed as ordinary flow content.
func rawBlockFragmentContext(block string) *html.Node {
	switch firstElementName(block) {
	case "tr", "td", "th":
		return &html.Node{Type: html.ElementNode, Data: "tbody", DataAtom: atom.Tbody}
	case "tbody", "thead", "tfoot", "caption", "colgroup", "col":
		return &html.Node{Type: html.ElementNode, Data: "table", DataAtom: atom.Table}
	default:
		return &html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div}
	}
}

// firstElementName returns the name of the first start or self-closing tag in
// block, or "" when the block has none. Text, comments and doctypes are skipped,
// so a comment before the markup does not decide the context.
func firstElementName(block string) string {
	tokenizer := html.NewTokenizer(strings.NewReader(block))

	for {
		switch tokenizer.Next() {
		case html.ErrorToken:
			return ""
		case html.StartTagToken, html.SelfClosingTagToken:
			name, _ := tokenizer.TagName()

			return strings.ToLower(string(name))
		}
	}
}

// styleDecl is one declaration of a style attribute. Inline style lists are
// tiny, so an ordered slice of property/value pairs beats a map: it preserves
// the node's declaration order, keeps renders byte-for-byte reproducible and
// avoids a hash table per matched node.
type styleDecl struct {
	property string
	value    string
}

// styleDecls is an ordered list of inline style declarations.
type styleDecls []styleDecl

// reset drops the declarations but keeps the backing array, so one scratch list
// can be reused for every matched node of a render.
func (d *styleDecls) reset() {
	*d = (*d)[:0]
}

// set applies one declaration in cascade order: it replaces the value of an
// existing property only when that property is absent or the new declaration is
// important, and appends new properties in application order. It reports whether
// the list changed. The keyword !important is not serialized, matching the
// inliner's historical output.
func (d *styleDecls) set(decl component.Style) bool {
	property := strings.TrimSpace(decl.Property)

	for i := range *d {
		if (*d)[i].property != property {
			continue
		}

		if !decl.Important {
			return false
		}

		changed := (*d)[i].value != decl.Value
		(*d)[i].value = decl.Value

		return changed
	}

	*d = append(*d, styleDecl{property: property, value: decl.Value})

	return true
}

// lookup returns the value of property, if the node has one.
func (d styleDecls) lookup(property string) (string, bool) {
	for i := range d {
		if d[i].property == property {
			return d[i].value, true
		}
	}

	return "", false
}

// String serializes the declarations the way the inliner always has: one
// "property:value;" per declaration, in declaration order.
func (d styleDecls) String() string {
	if len(d) == 0 {
		return ""
	}

	size := 0
	for i := range d {
		size += len(d[i].property) + len(d[i].value) + 2
	}

	var b strings.Builder
	b.Grow(size)

	for i := range d {
		b.WriteString(d[i].property)
		b.WriteByte(':')
		b.WriteString(d[i].value)
		b.WriteByte(';')
	}

	return b.String()
}

func inlineCSS(ctx *component.RenderContext, r io.Reader, w io.Writer) error {
	// Reading through a strings.Builder hands the document to the HTML parser
	// without the extra copy io.ReadAll plus a string conversion would make.
	var input strings.Builder
	if _, err := io.Copy(&input, r); err != nil {
		return err
	}

	protected, rawBlocks := protectRawBlocks(input.String())

	htmlNode, err := html.Parse(strings.NewReader(protected))
	if err != nil {
		return err
	}

	changed := applyHTMLAttributes(ctx.HTMLAttributes, htmlNode)

	inlined, err := inlineStyles(ctx.InlineStyles, htmlNode)
	if err != nil {
		return err
	}

	changed = changed || inlined

	if err := styleRawBlocks(ctx, rawBlocks); err != nil {
		return err
	}

	if !changed {
		// No rule matched and no selector set an attribute, so the parsed
		// document is the input document: re-rendering it would only burn time.
		return writeRestoredRawBlocks(w, protected, rawBlocks)
	}

	var rendered strings.Builder
	rendered.Grow(len(protected) + len(protected)/8)

	if err := html.Render(&rendered, htmlNode); err != nil {
		return err
	}

	return writeRestoredRawBlocks(w, rendered.String(), rawBlocks)
}

// inlineStyles applies every rule of every inline stylesheet to the document in
// declaration order and reports whether the document changed. Selectors are
// compiled with the stylesheet, so this never parses CSS.
func inlineStyles(sheets []component.Stylesheet, doc *html.Node) (bool, error) {
	return inlineStylesIn(sheets, []*html.Node{doc})
}

// inlineStylesIn is inlineStyles over a set of disjoint roots: the document for
// the whole-document pass, the top-level fragment nodes for a raw block.
func inlineStylesIn(sheets []component.Stylesheet, roots []*html.Node) (bool, error) {
	var (
		changed bool
		scratch styleDecls
	)

	for i := range sheets {
		rules := sheets[i].Rules

		for j := range rules {
			sel, ok := rules[j].CompiledSelectors()
			if !ok {
				// MJML silently ignores a rule whose selectors do not parse.
				continue
			}

			for _, root := range roots {
				for _, n := range sel.Select(root) {
					nodeChanged, err := applyDeclarations(n, rules[j].Declarations, &scratch)
					if err != nil {
						return false, err
					}

					changed = changed || nodeChanged
				}
			}
		}
	}

	return changed, nil
}

// applyDeclarations merges declarations into the node's style attribute and
// mirrors the width/text-align/vertical-align/background-color declarations to
// the legacy HTML attributes MJML emits. It reports whether the node changed.
func applyDeclarations(n *html.Node, declarations []component.Style, scratch *styleDecls) (bool, error) {
	styleIndex := -1
	for i := range n.Attr {
		if n.Attr[i].Key == "style" {
			styleIndex = i
		}
	}

	scratch.reset()

	if styleIndex >= 0 {
		if err := parseStyleAttribute(scratch, n.Attr[styleIndex].Val); err != nil {
			return false, fmt.Errorf("failed to parse style attribute: %w", err)
		}
	}

	changed := false

	if styleIndex < 0 && len(declarations) == 0 {
		// A matching rule without declarations adds nothing: an empty style
		// attribute carries no meaning, so do not rewrite the node.
		return false, nil
	}

	declarationsChanged := false
	for _, decl := range declarations {
		if scratch.set(decl) {
			declarationsChanged = true
		}
	}

	if n.Data == "table" || n.Data == "td" || n.Data == "div" {
		if v, ok := scratch.lookup("width"); ok && v != "" {
			if !hasAttribute(n, "width") {
				if strings.HasSuffix(v, "px") {
					n.Attr = append(n.Attr, html.Attribute{Key: "width", Val: component.RemoveNonNumeric(v)})
				} else {
					n.Attr = append(n.Attr, html.Attribute{Key: "width", Val: v})
				}

				changed = true
			}
		}
	}

	if v, ok := scratch.lookup("text-align"); ok && v != "" && !hasAttribute(n, "align") {
		n.Attr = append(n.Attr, html.Attribute{Key: "align", Val: v})
		changed = true
	}

	if v, ok := scratch.lookup("vertical-align"); ok && v != "" && !hasAttribute(n, "valign") {
		n.Attr = append(n.Attr, html.Attribute{Key: "valign", Val: v})
		changed = true
	}

	if v, ok := scratch.lookup("background-color"); ok && v != "" && !hasAttribute(n, "bgcolor") {
		n.Attr = append(n.Attr, html.Attribute{Key: "bgcolor", Val: v})
		changed = true
	}

	// The style attribute is written last, after the mirrored attributes, so the
	// emitted attribute order stays what it was before the merge was rewritten.
	// An unchanged style attribute keeps its exact current text instead of being
	// reserialized.
	if declarationsChanged || styleIndex < 0 {
		value := scratch.String()

		if styleIndex < 0 {
			n.Attr = append(n.Attr, html.Attribute{Key: "style", Val: value})
		} else if n.Attr[styleIndex].Val != value {
			n.Attr[styleIndex].Val = value
		}

		changed = true
	}

	return changed, nil
}

// hasAttribute reports whether n already carries the attribute.
func hasAttribute(n *html.Node, name string) bool {
	for i := range n.Attr {
		if n.Attr[i].Key == name {
			return true
		}
	}

	return false
}

// applyHTMLAttributes runs the <mj-html-attributes> selector pass on the parsed
// document before inline styles are applied, mirroring MJML's pipeline. MJML
// runs the pass against the rendered body content, so only the body subtree is
// considered and the body element itself is not a target.
//
// Each selector path is compiled once per render, outside the node loop, and an
// unparsable path is skipped like MJML does. It reports whether any attribute
// changed.
func applyHTMLAttributes(selectors []component.HTMLAttributeSelector, doc *html.Node) bool {
	if len(selectors) == 0 {
		return false
	}

	body := findHTMLElement(doc, "body")
	if body == nil {
		return false
	}

	return applyHTMLAttributesIn(selectors, []*html.Node{body}, body)
}

// applyHTMLAttributesIn is applyHTMLAttributes over a set of disjoint roots,
// skipping skip itself when a selector matches it. The raw-block pass passes the
// block's fragment roots and no skip.
func applyHTMLAttributesIn(selectors []component.HTMLAttributeSelector, roots []*html.Node, skip *html.Node) bool {
	changed := false

	for _, htmlSelector := range selectors {
		sel, err := css.Parse(htmlSelector.Path)
		if err != nil {
			continue
		}

		for _, root := range roots {
			for _, target := range sel.Select(root) {
				if target == skip {
					continue
				}

				for _, htmlAttribute := range htmlSelector.Attributes {
					if setHTMLAttribute(target, htmlAttribute.Name, htmlAttribute.Value) {
						changed = true
					}
				}
			}
		}
	}

	return changed
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

// setHTMLAttribute sets one attribute on n and reports whether it changed.
func setHTMLAttribute(n *html.Node, name, value string) bool {
	for i := range n.Attr {
		if n.Attr[i].Key != name {
			continue
		}

		changed := n.Attr[i].Val != value
		n.Attr[i].Val = value

		return changed
	}

	n.Attr = append(n.Attr, html.Attribute{Key: name, Val: value})

	return true
}

// parseStyleAttribute appends the declarations of an existing style attribute to
// dst, keeping their order. A repeated property keeps its first position and
// takes the last value, like the map it replaces.
func parseStyleAttribute(dst *styleDecls, style string) error {
	for declaration := range strings.SplitSeq(style, ";") {
		declaration = strings.TrimSpace(declaration)
		if declaration == "" {
			continue
		}

		parts := strings.SplitN(declaration, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid style declaration: %s", declaration)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" || value == "" {
			return fmt.Errorf("invalid style declaration: %s", declaration)
		}

		dst.appendOrReplace(styleDecl{property: key, value: value})
	}

	return nil
}

// appendOrReplace keeps the position of an existing property and takes the new
// value; a new property is appended.
func (d *styleDecls) appendOrReplace(decl styleDecl) {
	for i := range *d {
		if (*d)[i].property == decl.property {
			(*d)[i].value = decl.value

			return
		}
	}

	*d = append(*d, decl)
}
