package mjmlgo

import (
	"fmt"
	"io"
	"strings"

	"github.com/ericchiang/css"
	"github.com/julez-dev/mjmlgo/component"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// The inline stage applies the document's <mj-style inline> stylesheets and its
// <mj-html-attributes> selectors to the markup the components emitted.
//
// It does that on an element tree built from the emitted markup itself (see
// buildEmittedDocument), and writes the document back out unchanged except for
// the start tags a pass actually modified. The markup is therefore emitted once
// and never parsed and re-serialized as a whole: an element no rule touches
// keeps the exact bytes its component wrote, and a matched element is patched in
// place. The tree carries no implied elements, no table fixups and no
// parser-inserted tags, because it is a record of what was emitted, not a
// reconstruction of what a browser would build.

// startTag is where one element start tag sits in the emitted document.
type startTag struct {
	start int
	end   int
	// selfClosing records that the element was emitted as <name ... />.
	selfClosing bool
}

// emittedDocument is the emitted markup together with the element tree the
// selector passes run against.
type emittedDocument struct {
	// source is the emitted markup, byte for byte.
	source string
	// root is the document node of the element tree; its descendant elements
	// correspond one to one, in document order, with spans.
	root *html.Node
	// spans holds the start tag of every element, in document order.
	spans []startTag
}

// buildEmittedDocument records the element structure of the emitted markup.
//
// The tokenizer supplies the tokens; this function only builds the ancestor and
// sibling relations the selector passes need, plus the byte span of each start
// tag so a matched element can be patched without rewriting the rest. Text and
// comments are not recorded at all: every matcher in the selector engine - the
// sibling and positional pseudo-classes included - counts element nodes only, so
// a document node per text token would buy nothing.
//
// Two deliberate simplifications follow from matching the emitted markup instead
// of re-parsing the document the way a browser would:
//
//   - a self-closing tag is a leaf. The renderer writes <img ... /> and
//     <td ... /> that way; the reference inliner uses the same tolerant parser
//     (htmlparser2) it always has, and neither inserts implied end tags;
//   - an end tag closes the nearest matching open element and is otherwise
//     ignored, so author raw content with unbalanced markup nests the way it was
//     written.
func buildEmittedDocument(source string) *emittedDocument {
	doc := &emittedDocument{
		source: source,
		root:   &html.Node{Type: html.DocumentNode},
	}

	stack := []*html.Node{doc.root}
	offset := 0

	tokenizer := html.NewTokenizer(strings.NewReader(source))

	for {
		tokenType := tokenizer.Next()

		// Raw() is only valid until the next token is read, so take its length
		// before parsing attributes out of it.
		length := len(tokenizer.Raw())
		if tokenType == html.ErrorToken {
			// io.EOF: the tokens covered the whole document.
			break
		}

		parent := stack[len(stack)-1]

		switch tokenType {
		case html.StartTagToken, html.SelfClosingTagToken:
			element := newElementNode(tokenizer)
			parent.AppendChild(element)

			selfClosing := tokenType == html.SelfClosingTagToken
			doc.spans = append(doc.spans, startTag{start: offset, end: offset + length, selfClosing: selfClosing})

			if !selfClosing && !voidElements[element.Data] {
				stack = append(stack, element)
			}
		case html.EndTagToken:
			name, _ := tokenizer.TagName()
			stack = popToElement(stack, string(name))
		}

		offset += length
	}

	return doc
}

// newElementNode builds the element node for the start tag the tokenizer is
// positioned at. Tag and attribute names arrive lower-cased, like they do from
// the HTML parser; a repeated attribute keeps its first value, again like the
// parser.
func newElementNode(tokenizer *html.Tokenizer) *html.Node {
	name, hasAttributes := tokenizer.TagName()

	element := &html.Node{Type: html.ElementNode}

	if a := atom.Lookup(name); a != 0 {
		// atom.String returns the interned name, so the common tags cost no
		// allocation and give the matchers the atom they compare against.
		element.DataAtom = a
		element.Data = a.String()
	} else {
		element.Data = string(name)
	}

	if !hasAttributes {
		return element
	}

	for {
		key, value, more := tokenizer.TagAttr()

		if !hasAttribute(element, string(key)) {
			element.Attr = append(element.Attr, html.Attribute{Key: string(key), Val: string(value)})
		}

		if !more {
			return element
		}
	}
}

// popToElement closes every open element up to and including the nearest one
// named name and returns the remaining stack. An end tag without a matching open
// element is ignored.
func popToElement(stack []*html.Node, name string) []*html.Node {
	for i := len(stack) - 1; i > 0; i-- {
		if stack[i].Data == name {
			return stack[:i]
		}
	}

	return stack
}

// voidElements are the elements that never have content, so an emitted
// <img ...> must not swallow the markup that follows it.
var voidElements = map[string]bool{
	"area":   true,
	"base":   true,
	"br":     true,
	"col":    true,
	"embed":  true,
	"hr":     true,
	"img":    true,
	"input":  true,
	"link":   true,
	"meta":   true,
	"param":  true,
	"source": true,
	"track":  true,
	"wbr":    true,
}

// write serializes the document: the emitted markup with the start tag of every
// element in changed replaced by a re-serialization of its current attributes.
func (d *emittedDocument) write(w io.Writer, changed map[*html.Node]struct{}) error {
	var (
		cursor     int
		next       int
		writeErr   error
		startTags  strings.Builder
		writeChunk = func(s string) {
			if writeErr != nil {
				return
			}

			_, writeErr = io.WriteString(w, s)
		}
	)

	var walk func(*html.Node)

	walk = func(n *html.Node) {
		if writeErr != nil {
			return
		}

		if n.Type == html.ElementNode {
			// Elements are visited in the order their start tags were emitted,
			// which is the order spans was filled in.
			span := d.spans[next]
			next++

			writeChunk(d.source[cursor:span.start])

			if _, dirty := changed[n]; dirty {
				startTags.Reset()
				writeStartTag(&startTags, n, span.selfClosing)
				writeChunk(startTags.String())
			} else {
				writeChunk(d.source[span.start:span.end])
			}

			cursor = span.end
		}

		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}

	walk(d.root)

	if writeErr != nil {
		return writeErr
	}

	writeChunk(d.source[cursor:])

	return writeErr
}

// writeStartTag writes one element start tag from its current attributes,
// escaping attribute values the way the HTML renderer did. An element the
// document emitted self-closed keeps that form, so author content that depends
// on it - a VML or SVG shape, say - is not rewritten into an open tag.
func writeStartTag(b *strings.Builder, n *html.Node, selfClosing bool) {
	b.WriteByte('<')
	b.WriteString(n.Data)

	for _, attribute := range n.Attr {
		b.WriteByte(' ')
		b.WriteString(attribute.Key)
		b.WriteString(`="`)
		b.WriteString(html.EscapeString(attribute.Val))
		b.WriteByte('"')
	}

	if selfClosing {
		b.WriteString(" />")
	} else {
		b.WriteByte('>')
	}
}

// styleApplier applies the two selector passes to an emitted document and
// remembers which elements they changed, so only those start tags are rewritten.
type styleApplier struct {
	// changed holds the elements whose markup no longer matches what was
	// emitted, either because the attribute pass set an attribute or because the
	// inline pass did.
	changed map[*html.Node]struct{}
	// styleChanged holds the elements whose style attribute a winning
	// declaration rewrote, so an element a rule matched without changing it
	// keeps the style attribute it was emitted with.
	styleChanged map[*html.Node]struct{}
	// merged holds the cascade result of every element a rule matched, and
	// touched lists those elements in first-match order so the result is
	// materialized deterministically.
	merged  map[*html.Node]*styleDecls
	touched []*html.Node
	// err records a style attribute the cascade could not parse.
	err error
}

// markChanged records that the node's attributes no longer match the emitted
// markup.
func (a *styleApplier) markChanged(n *html.Node) {
	if a.changed == nil {
		a.changed = make(map[*html.Node]struct{})
	}

	a.changed[n] = struct{}{}
}

// markStyleChanged records that a declaration won the cascade for n, so its
// style attribute has to be written out.
func (a *styleApplier) markStyleChanged(n *html.Node) {
	if a.styleChanged == nil {
		a.styleChanged = make(map[*html.Node]struct{})
	}

	a.styleChanged[n] = struct{}{}
}

// inlineCSS applies the inline stylesheets and the <mj-html-attributes> selectors
// in ctx to the document read from r and writes the result to w.
func inlineCSS(ctx *component.RenderContext, r io.Reader, w io.Writer) error {
	// Reading through a strings.Builder keeps one copy of the document: the
	// tokenizer reads it through a strings.Reader, and the writer slices it.
	var source strings.Builder
	if _, err := io.Copy(&source, r); err != nil {
		return err
	}

	emitted := buildEmittedDocument(source.String())

	var applier styleApplier

	// The <mj-html-attributes> pass runs before the inline styles so that a rule
	// matching an injected attribute sees it, like the reference pipeline.
	applier.applyHTMLAttributes(ctx.HTMLAttributes, emitted.root)
	if err := applier.applyStyles(ctx.InlineStyles, emitted.root); err != nil {
		return err
	}

	if len(applier.changed) == 0 {
		// Nothing matched: the emitted document is the result.
		_, err := io.WriteString(w, emitted.source)

		return err
	}

	return emitted.write(w, applier.changed)
}

// applyStyles collects every inline stylesheet's declarations for the document
// and merges them into the matched elements in cascade order: importance first,
// then specificity, then source order, like CSS and the reference inliner. A rule
// that lists several selectors is applied once per selector, so each match
// carries the specificity of the selector that actually matched.
//
// Declarations are collected per element and written out afterwards, because the
// cascade compares declarations with each other: flattening them into the style
// attribute after every rule - and re-reading the attribute for the next one -
// would make a declaration an element received from an earlier rule look like an
// authored inline style and outrank the rest of the stylesheet.
func (a *styleApplier) applyStyles(sheets []component.Stylesheet, doc *html.Node) error {
	order := 0

	for i := range sheets {
		rules := sheets[i].Rules

		for j := range rules {
			rule := &rules[j]

			for _, resolved := range resolveRule(rule) {
				order++

				for _, n := range resolved.selector.Select(doc) {
					decls := a.declarationsFor(n)

					if mergeDeclarations(decls, rule.Declarations, resolved.specificity, order) {
						a.markStyleChanged(n)
					}
				}
			}
		}
	}

	if a.err != nil {
		return a.err
	}

	for _, n := range a.touched {
		_, styleChanged := a.styleChanged[n]

		if materializeDeclarations(n, *a.merged[n], styleChanged) {
			a.markChanged(n)
		}
	}

	return nil
}

// declarationsFor returns the cascade list for n, which starts as the element's
// own style attribute so that an authored inline style outranks the stylesheets.
// A style attribute that cannot be parsed is recorded in err, which applyStyles
// reports once the pass is done.
func (a *styleApplier) declarationsFor(n *html.Node) *styleDecls {
	if decls, ok := a.merged[n]; ok {
		return decls
	}

	decls := new(styleDecls)

	if a.err == nil {
		if style, has := attributeValue(n, "style"); has && style != "" {
			if err := parseStyleAttribute(decls, style); err != nil {
				a.err = fmt.Errorf("failed to parse style attribute: %w", err)
			}
		}
	}

	if a.merged == nil {
		a.merged = make(map[*html.Node]*styleDecls)
	}

	a.merged[n] = decls
	a.touched = append(a.touched, n)

	return decls
}

// resolvedSelector is one selector of a rule, with the specificity its
// declarations cascade at.
type resolvedSelector struct {
	selector    *css.Selector
	specificity specificity
}

// resolveRule returns the rule's selectors, ready to be matched. A rule with a
// single selector reuses the selector compiled with the stylesheet; a selector
// list is split and compiled here, because a compiled list reports the nodes it
// matches but not which of its selectors matched them, and the cascade needs
// that selector's specificity. A rule with no usable selector returns nil, like
// the reference inliner's silent skip of a selector its parser rejects.
func resolveRule(rule *component.Rule) []resolvedSelector {
	selectors := splitSelectorList(rule.Selectors)

	if len(selectors) == 1 {
		compiled, ok := rule.CompiledSelectors()
		if !ok {
			return nil
		}

		return []resolvedSelector{{selector: compiled, specificity: selectorSpecificity(selectors[0])}}
	}

	resolved := make([]resolvedSelector, 0, len(selectors))

	for _, text := range selectors {
		compiled, err := css.Parse(text)
		if err != nil {
			// MJML silently ignores a selector its parser does not accept.
			continue
		}

		resolved = append(resolved, resolvedSelector{selector: compiled, specificity: selectorSpecificity(text)})
	}

	return resolved
}

// applyHTMLAttributes runs the <mj-html-attributes> selector pass on the emitted
// document, mirroring MJML's pipeline. MJML runs the pass against the rendered
// body content, so only the body subtree is considered and the body element
// itself is not a target.
//
// Each selector path is compiled once per render, outside the node loop, and an
// unparsable path is skipped like MJML does.
func (a *styleApplier) applyHTMLAttributes(selectors []component.HTMLAttributeSelector, doc *html.Node) {
	if len(selectors) == 0 {
		return
	}

	body := findHTMLElement(doc, "body")
	if body == nil {
		return
	}

	for _, htmlSelector := range selectors {
		selector, err := css.Parse(htmlSelector.Path)
		if err != nil {
			continue
		}

		for _, target := range selector.Select(body) {
			if target == body {
				continue
			}

			for _, htmlAttribute := range htmlSelector.Attributes {
				if setAttribute(target, htmlAttribute.Name, htmlAttribute.Value) {
					a.markChanged(target)
				}
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

// styleDecl is one declaration of a style attribute. Inline style lists are
// tiny, so an ordered slice of property/value pairs beats a map: it preserves
// the node's declaration order, keeps renders byte-for-byte reproducible and
// avoids a hash table per matched node.
//
// Besides the declaration itself a styleDecl carries what the cascade compares:
// the importance and the specificity of the origin, and the declaration's
// position in the stylesheet.
type styleDecl struct {
	property string
	value    string
	// priority is 2 for an !important declaration and 0 otherwise; it raises
	// the first specificity entry, so an important declaration outranks a more
	// specific plain one.
	priority int
	// specificity is the specificity vector of the origin selector. The element's
	// own style attribute cascades as {1, 0, 0, 0}.
	specificity specificity
	// order is the position of the declaration in the stylesheet; it decides
	// between declarations of equal importance and specificity, where the later
	// one wins.
	order int
}

// styleDecls is an ordered list of inline style declarations.
type styleDecls []styleDecl

// reset drops the declarations but keeps the backing array, so one scratch list
// can be reused for every matched node of a render.
func (d *styleDecls) reset() {
	*d = (*d)[:0]
}

// set applies one declaration in cascade order: a property keeps the position it
// first appeared at, and takes the new value only when the new declaration wins
// the cascade (see winsOver). It reports whether the serialized declarations
// changed. The keyword !important is not serialized, matching the inliner's
// historical output.
func (d *styleDecls) set(decl styleDecl) bool {
	property := strings.TrimSpace(decl.property)

	for i := range *d {
		if (*d)[i].property != property {
			continue
		}

		if !winsOver(decl, (*d)[i]) {
			return false
		}

		changed := (*d)[i].value != decl.value
		(*d)[i] = decl
		(*d)[i].property = property

		return changed
	}

	decl.property = property
	*d = append(*d, decl)

	return true
}

// winsOver reports whether a outranks b in the cascade. It compares the same
// tuple the reference inliner sorts properties by: importance added to the first
// specificity entry, then the rest of the specificity vector, then the
// declaration's position, where a later declaration wins a full tie.
func winsOver(a, b styleDecl) bool {
	for i := range a.specificity {
		left, right := a.specificity[i], b.specificity[i]

		if i == 0 {
			left += a.priority
			right += b.priority
		}

		if left != right {
			return left > right
		}
	}

	return a.order >= b.order
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

// mergeDeclarations merges one rule's declarations into the element's cascade
// list and reports whether a serialized value changed.
func mergeDeclarations(decls *styleDecls, declarations []component.Style, origin specificity, order int) bool {
	changed := false

	for _, decl := range declarations {
		priority := 0
		if decl.Important {
			priority = 2
		}

		if decls.set(styleDecl{
			property:    decl.Property,
			value:       decl.Value,
			priority:    priority,
			specificity: origin,
			order:       order,
		}) {
			changed = true
		}
	}

	return changed
}

// materializeDeclarations writes the winning declarations back to the node: the
// legacy width/text-align/vertical-align/background-color attributes MJML mirrors
// from them, plus the style attribute when a declaration won the cascade for the
// element (setStyle) - an element a rule matched without changing it keeps the
// style attribute it was emitted with. It reports whether the node changed.
func materializeDeclarations(n *html.Node, decls styleDecls, setStyle bool) bool {
	if len(decls) == 0 {
		// A matching rule without winning declarations adds nothing: an empty
		// style attribute carries no meaning, so do not rewrite the node.
		return false
	}

	changed := false

	if n.Data == "table" || n.Data == "td" || n.Data == "div" {
		if v, ok := decls.lookup("width"); ok && v != "" {
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

	// The mirrored attributes follow the winning declaration, replacing a value
	// the element carried before: the stylesheet, not the emitted attribute, is
	// what a reader of the copied document sees.
	if v, ok := decls.lookup("text-align"); ok && v != "" && setAttribute(n, "align", v) {
		changed = true
	}

	if v, ok := decls.lookup("vertical-align"); ok && v != "" && setAttribute(n, "valign", v) {
		changed = true
	}

	if v, ok := decls.lookup("background-color"); ok && v != "" && setAttribute(n, "bgcolor", v) {
		changed = true
	}

	if !setStyle {
		return changed
	}

	// The style attribute is written last, after the mirrored attributes, so the
	// emitted attribute order stays what it was before the merge was rewritten.
	// An unchanged style attribute keeps its exact current text instead of being
	// reserialized.
	value := decls.String()

	for i := range n.Attr {
		if n.Attr[i].Key != "style" {
			continue
		}

		if n.Attr[i].Val != value {
			n.Attr[i].Val = value

			return true
		}

		return changed
	}

	n.Attr = append(n.Attr, html.Attribute{Key: "style", Val: value})

	return true
}

// setAttribute sets the attribute to value - replacing a value the node already
// carries - and reports whether the node changed. The inline stage uses it for
// the <mj-html-attributes> pass and for the legacy attributes it mirrors from the
// winning declarations: in both cases the stylesheet, not the emitted attribute,
// is what a reader of the copied document should see.
func setAttribute(n *html.Node, name, value string) bool {
	for i := range n.Attr {
		if n.Attr[i].Key != name {
			continue
		}

		if n.Attr[i].Val == value {
			return false
		}

		n.Attr[i].Val = value

		return true
	}

	n.Attr = append(n.Attr, html.Attribute{Key: name, Val: value})

	return true
}

// attributeValue returns the value of the node's attribute, and whether it has
// one.
func attributeValue(n *html.Node, name string) (string, bool) {
	for i := range n.Attr {
		if n.Attr[i].Key == name {
			return n.Attr[i].Val, true
		}
	}

	return "", false
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

// parseStyleAttribute appends the declarations of an existing style attribute to
// dst, keeping their order. A repeated property keeps its first position and
// takes the last value, like the map it replaces. The declarations cascade as the
// element's own style, so they outrank every rule that is not important.
func parseStyleAttribute(dst *styleDecls, style string) error {
	order := 0

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

		priority := 0
		if strings.HasSuffix(value, "!important") {
			priority = 2
		}

		order++
		dst.appendOrReplace(styleDecl{
			property:    key,
			value:       value,
			priority:    priority,
			specificity: specificity{1},
			order:       order,
		})
	}

	return nil
}

// appendOrReplace keeps the position of an existing property and takes the new
// value; a new property is appended.
func (d *styleDecls) appendOrReplace(decl styleDecl) {
	for i := range *d {
		if (*d)[i].property != decl.property {
			continue
		}

		(*d)[i] = decl

		return
	}

	*d = append(*d, decl)
}
