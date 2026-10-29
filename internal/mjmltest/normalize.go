package mjmltest

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/aymerick/douceur/css"
	"github.com/aymerick/douceur/parser"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Kind describes the type of a normalized node.
type Kind int

const (
	// KindDocument is the document root.
	KindDocument Kind = iota
	// KindDoctype is a doctype declaration.
	KindDoctype
	// KindElement is a regular HTML element.
	KindElement
	// KindText is non-empty, whitespace-normalized text.
	KindText
	// KindConditional is an Outlook conditional comment (<!--[if mso]>...<![endif]-->).
	// Name holds the normalized condition, Children the parsed payload.
	KindConditional
	// KindComment is any other HTML comment. Text holds the comment body exactly
	// as written, without the <!-- and --> delimiters: MJML preserves author
	// comments byte for byte, spacing included, so the harness compares them
	// byte for byte too.
	KindComment
)

// Attr is a normalized HTML attribute.
type Attr struct {
	Key string
	Val string
}

// Node is a normalized HTML tree. Two documents that render the same in a mail
// client produce equal Node trees, regardless of formatting, attribute order or
// CSS declaration order.
type Node struct {
	Kind     Kind
	Name     string // element tag name, or the condition of a conditional comment
	Text     string // doctype value or text content
	Sheet    string // canonical stylesheet of a <style> element
	Attrs    []Attr
	Children []*Node
}

// ParseHTML parses an HTML document into its normalized tree form.
//
// Normalization rules:
//   - Outlook conditional comments are parsed and compared as part of the tree;
//     every other comment is kept as a KindComment node whose text is compared
//     byte for byte, so a dropped or re-spaced author comment fails the test;
//   - whitespace-only text nodes are dropped and all other text has runs of
//     whitespace collapsed to a single space;
//   - <style> contents are parsed and canonicalized so that generated CSS is
//     compared as well, keeping rule and declaration order (the cascade is
//     order sensitive) while ignoring formatting and comments;
//   - attributes are sorted by name, style declarations are parsed and sorted,
//     class tokens are sorted, and empty style/class/alt attributes are dropped.
func ParseHTML(document string) (*Node, error) {
	doc, err := html.Parse(strings.NewReader(document))
	if err != nil {
		return nil, err
	}

	node := convert(doc)
	if node == nil {
		return nil, fmt.Errorf("empty HTML document")
	}

	return node, nil
}

func convert(n *html.Node) *Node {
	switch n.Type {
	case html.DocumentNode:
		out := &Node{Kind: KindDocument, Name: "#document"}
		for child := range n.ChildNodes() {
			out.Children = appendConverted(out.Children, child)
		}
		return out

	case html.DoctypeNode:
		return &Node{Kind: KindDoctype, Text: strings.ToLower(strings.Join(strings.Fields(n.Data), " "))}

	case html.ElementNode:
		name := strings.ToLower(n.Data)
		out := &Node{
			Kind:  KindElement,
			Name:  name,
			Attrs: normalizeAttrs(n.Attr),
		}

		if name == "style" {
			out.Sheet = canonicalStylesheet(textContent(n))

			return out
		}

		for child := range n.ChildNodes() {
			out.Children = appendConverted(out.Children, child)
		}

		return out

	case html.TextNode:
		text := normalizeText(n.Data)
		if text == "" {
			return nil
		}
		return &Node{Kind: KindText, Text: text}

	case html.CommentNode:
		if conditional := convertConditional(n.Data); conditional != nil {
			return conditional
		}

		return &Node{Kind: KindComment, Text: n.Data}

	default:
		return nil
	}
}

func appendConverted(dst []*Node, n *html.Node) []*Node {
	converted := convert(n)
	if converted == nil {
		return dst
	}

	return append(dst, converted)
}

var (
	conditionalStart = regexp.MustCompile(`^\[if\s+([^\]]*)\]>\s*`)
	conditionalEnd   = regexp.MustCompile(`\s*<!\[endif\]\s*$`)
)

func convertConditional(data string) *Node {
	m := conditionalStart.FindStringSubmatch(data)
	if m == nil {
		return nil
	}

	condition := normalizeText(m[1])

	payload := data[len(m[0]):]
	payload = conditionalEnd.ReplaceAllString(payload, "")
	// Handles the split `<!--[if !mso]><!-->` form.
	payload = strings.TrimSuffix(strings.TrimSpace(payload), "<!")
	payload = strings.TrimSpace(payload)

	if payload == "" {
		// Downlevel-reveal wrappers (<!--[if !mso]><!--> ... <!--<![endif]-->) have
		// an empty payload but still shape the document, so keep them. Plain end
		// markers (<![endif]-->) never reach this point: they do not start with
		// "[if" and are dropped as ordinary comments.
		if strings.HasPrefix(strings.TrimSpace(condition), "!") {
			return &Node{Kind: KindConditional, Name: condition}
		}

		return nil
	}

	out := &Node{Kind: KindConditional, Name: condition}

	contextNode := &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body}

	nodes, err := html.ParseFragment(strings.NewReader(payload), contextNode)
	if err != nil {
		// Fall back to comparing the raw payload.
		out.Children = append(out.Children, &Node{Kind: KindText, Text: normalizeText(payload)})
		return out
	}

	for _, fragment := range nodes {
		out.Children = appendConverted(out.Children, fragment)
	}

	return out
}

// normalizeAttrs returns the attributes sorted by name with values normalized.
func normalizeAttrs(attrs []html.Attribute) []Attr {
	out := make([]Attr, 0, len(attrs))

	for _, attr := range attrs {
		key := strings.ToLower(attr.Key)
		val := strings.TrimSpace(attr.Val)

		switch key {
		case "id", "for", "name":
			// Generated ids are also embedded in longer values, e.g. the
			// carousel radios use id="mj-carousel-<id>-radio-1" and a shared
			// name="mj-carousel-radio-<id>".
			val = normalizeGeneratedIDs(val)
		case "style":
			val = normalizeStyle(val)
			if val == "" {
				continue
			}
		case "class":
			val = normalizeClass(val)
			if val == "" {
				continue
			}
		case "alt":
			if val == "" {
				continue
			}
		}

		out = append(out, Attr{Key: key, Val: val})
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Key != out[j].Key {
			return out[i].Key < out[j].Key
		}

		return out[i].Val < out[j].Val
	})

	return out
}

// generatedIDPlaceholder replaces the 16-character hex ids MJML generates for
// navbar hamburger labels and carousels. They are random on every reference
// render, so the harness treats them as opaque identifiers.
const generatedIDPlaceholder = "mjml-generated-id"

// normalizeGeneratedIDs replaces generated ids that are embedded in a longer
// value. MJML also puts the random carousel id inside class names
// (mj-carousel-<id>-thumbnail-1) and inside the generated CSS selectors, so the
// substitution has to work on substrings and not only on whole attribute
// values. Only maximal runs of exactly 16 lowercase hex characters are
// replaced, so ordinary names are left alone.
func normalizeGeneratedIDs(val string) string {
	if len(val) < 16 {
		return val
	}

	var b strings.Builder

	for i := 0; i < len(val); {
		if !isHexDigit(val[i]) {
			b.WriteByte(val[i])
			i++

			continue
		}

		end := i
		for end < len(val) && isHexDigit(val[end]) {
			end++
		}

		if end-i == 16 {
			b.WriteString(generatedIDPlaceholder)
		} else {
			b.WriteString(val[i:end])
		}

		i = end
	}

	return b.String()
}

func isHexDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')
}

// normalizeClass sorts class tokens and collapses whitespace. Generated ids
// embedded in class names are replaced so two renders of the same document
// compare equal.
func normalizeClass(val string) string {
	tokens := strings.Fields(normalizeGeneratedIDs(val))
	sort.Strings(tokens)

	return strings.Join(tokens, " ")
}

// normalizeStyle parses an inline style attribute and serializes it with sorted
// declarations so that declaration order does not matter.
func normalizeStyle(val string) string {
	declarations := make(map[string]string)

	for _, declaration := range strings.Split(val, ";") {
		declaration = strings.TrimSpace(declaration)
		if declaration == "" {
			continue
		}

		property, value, ok := strings.Cut(declaration, ":")
		if !ok {
			// Keep malformed declarations visible instead of dropping them.
			declarations[declaration] = ""
			continue
		}

		property = strings.ToLower(strings.TrimSpace(property))
		value = normalizeText(value)
		if property == "" {
			continue
		}

		declarations[property] = value
	}

	if len(declarations) == 0 {
		return ""
	}

	properties := make([]string, 0, len(declarations))
	for property := range declarations {
		properties = append(properties, property)
	}

	sort.Strings(properties)

	var b strings.Builder
	for _, property := range properties {
		value := declarations[property]
		if value == "" {
			b.WriteString(property)
			b.WriteString(";")
			continue
		}

		b.WriteString(property)
		b.WriteString(":")
		b.WriteString(value)
		b.WriteString(";")
	}

	return b.String()
}

// normalizeText collapses all whitespace runs into single spaces and trims.
func normalizeText(val string) string {
	return strings.Join(strings.Fields(val), " ")
}

// CanonicalMust is Canonical for an HTML document, ignoring parse errors.
func CanonicalMust(document string) string {
	node, err := ParseHTML(document)
	if err != nil {
		return "parse error: " + err.Error()
	}

	return Canonical(node)
}

// Canonical renders the normalized tree as indented text. It is used for
// readable failure output and debugging.
func Canonical(n *Node) string {
	var b strings.Builder
	writeCanonical(&b, n, 0)

	return b.String()
}

func writeCanonical(b *strings.Builder, n *Node, depth int) {
	indent := strings.Repeat("  ", depth)

	switch n.Kind {
	case KindDocument:
		b.WriteString(indent)
		b.WriteString("#document\n")
	case KindDoctype:
		b.WriteString(indent)
		b.WriteString("<!doctype ")
		b.WriteString(n.Text)
		b.WriteString(">\n")
	case KindText:
		b.WriteString(indent)
		b.WriteString("#text ")
		b.WriteString(n.Text)
		b.WriteString("\n")
	case KindConditional:
		b.WriteString(indent)
		b.WriteString("#conditional [if ")
		b.WriteString(n.Name)
		b.WriteString("]\n")
	case KindComment:
		b.WriteString(indent)
		b.WriteString("#comment <")
		b.WriteString(n.Text)
		b.WriteString(">\n")
	case KindElement:
		b.WriteString(indent)
		b.WriteString("<")
		b.WriteString(n.Name)

		for _, attr := range n.Attrs {
			b.WriteString(" ")
			b.WriteString(attr.Key)
			b.WriteString(`="`)
			b.WriteString(attr.Val)
			b.WriteString(`"`)
		}

		if n.Sheet != "" {
			b.WriteString(">\n")
			b.WriteString(indent)
			b.WriteString("  #sheet ")
			b.WriteString(n.Sheet)
			b.WriteString("\n")
			return
		}

		if len(n.Children) == 0 {
			b.WriteString(">\n")
			return
		}

		b.WriteString(">\n")
	}

	for _, child := range n.Children {
		writeCanonical(b, child, depth+1)
	}
}

// textContent returns the concatenated text of a node's children.
func textContent(n *html.Node) string {
	var b strings.Builder

	for descendant := range n.Descendants() {
		if descendant.Type == html.TextNode {
			b.WriteString(descendant.Data)
		}
	}

	return b.String()
}

// canonicalStylesheet parses a stylesheet and renders it in a canonical form:
// comments and formatting dropped, property names lowercased, whitespace
// collapsed, while rule and declaration order are preserved because they
// determine the cascade (and shorthand/longhand precedence).
func canonicalStylesheet(stylesheet string) string {
	stylesheet = cssComments.ReplaceAllString(stylesheet, "")

	parsed, err := parser.Parse(stylesheet)
	if err != nil {
		// Unparsable stylesheets (template payloads, vendor hacks) are compared
		// as text with formatting removed. Whitespace around CSS punctuation is
		// never significant, and formatters reflow it.
		return normalizeCSSWhitespace(stylesheet)
	}

	var b strings.Builder
	writeCanonicalRules(&b, parsed.Rules)

	return b.String()
}

func writeCanonicalRules(b *strings.Builder, rules []*css.Rule) {
	for _, rule := range rules {
		if rule.Kind == css.AtRule {
			b.WriteString(strings.ToLower(normalizeText(rule.Name)))

			if prelude := normalizeCSSValue(rule.Prelude); prelude != "" {
				b.WriteString(" ")
				b.WriteString(prelude)
			}

			switch {
			case len(rule.Rules) > 0:
				b.WriteString("{")
				writeCanonicalRules(b, rule.Rules)
				b.WriteString("}")
			case len(rule.Declarations) > 0:
				b.WriteString("{")
				writeCanonicalDeclarations(b, rule.Declarations)
				b.WriteString("}")
			}

			continue
		}

		selectors := make([]string, 0, len(rule.Selectors))
		for _, selector := range rule.Selectors {
			selectors = append(selectors, normalizeCSSSelector(selector))
		}

		if len(selectors) == 0 {
			selectors = append(selectors, normalizeCSSSelector(rule.Prelude))
		}

		b.WriteString(strings.Join(selectors, ","))
		b.WriteString("{")
		writeCanonicalDeclarations(b, rule.Declarations)
		b.WriteString("}")
	}
}

func writeCanonicalDeclarations(b *strings.Builder, declarations []*css.Declaration) {
	for i, declaration := range declarations {
		if i > 0 {
			b.WriteString(";")
		}

		b.WriteString(strings.ToLower(normalizeText(declaration.Property)))
		b.WriteString(":")
		b.WriteString(normalizeCSSValue(declaration.Value))

		if declaration.Important {
			b.WriteString(" !important")
		}
	}
}

var (
	cssComments          = regexp.MustCompile(`(?s)/\*.*?\*/`)
	cssCombinatorSpacing = regexp.MustCompile(`\s*([>+~])\s*`)
)

// normalizeCSSSelector collapses whitespace, drops it around combinators and
// replaces generated ids, which appear in the carousel selectors.
func normalizeCSSSelector(selector string) string {
	selector = normalizeGeneratedIDs(normalizeText(selector))

	return cssCombinatorSpacing.ReplaceAllString(selector, "$1")
}

// normalizeCSSValue collapses runs of whitespace and drops whitespace around
// CSS punctuation. Values carrying template payloads (templating braces are not
// CSS) are thereby compared independently of how a formatter reflowed them.
func normalizeCSSValue(value string) string {
	return normalizeCSSWhitespace(strings.TrimSuffix(normalizeText(value), ";"))
}

// normalizeCSSWhitespace collapses whitespace and removes it where it cannot be
// significant: around braces, colons, semicolons and commas.
func normalizeCSSWhitespace(value string) string {
	var b strings.Builder
	b.Grow(len(value))

	pendingSpace := false
	var prev rune

	for _, char := range value {
		if unicode.IsSpace(char) {
			pendingSpace = true

			continue
		}

		if pendingSpace && b.Len() > 0 && !isCSSPunctuation(prev) && !isCSSPunctuation(char) {
			b.WriteByte(' ')
		}

		pendingSpace = false
		b.WriteRune(char)
		prev = char
	}

	return b.String()
}

func isCSSPunctuation(char rune) bool {
	switch char {
	case '{', '}', ':', ';', ',':
		return true
	default:
		return false
	}
}
