package mjmlgo

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/julez-dev/mjmlgo/component"
	"github.com/julez-dev/mjmlgo/node"
)

var (
	ErrParsingFailed = errors.New("parsing MJML structure failed")

	// ErrIncludeNoFS is returned when a document contains mj-include but was
	// rendered through an entry point without a filesystem (Renderer.Render).
	ErrIncludeNoFS = errors.New("mjml: mj-include requires an fs.FS")
	// ErrIncludePath is returned for include paths that are absolute, escape the
	// filesystem root or are otherwise not valid fs.FS names.
	ErrIncludePath = errors.New("mjml: invalid mj-include path")
	// ErrIncludeType is returned for mj-include type values other than mjml,
	// html and css. MJML silently treats unknown types as mjml includes.
	ErrIncludeType = errors.New("mjml: unknown mj-include type")
	// ErrIncludeCycle is returned when a file includes itself, directly or
	// through other files.
	ErrIncludeCycle = errors.New("mjml: mj-include cycle")
	// ErrIncludeDepth is returned when includes nest deeper than maxIncludeDepth.
	ErrIncludeDepth = errors.New("mjml: mj-include depth limit exceeded")
)

// parse decodes an MJML document. keepComments preserves author comments as
// mj-raw nodes at their document position, which is exactly how MJML's parser
// represents them; when it is false they are dropped here, at parse time, so
// they never reach validation or rendering.
func parse(src []byte, keepComments bool) (*node.Node, error) {
	dec := xml.NewDecoder(strings.NewReader(protectRawContent(src)))
	dec.Strict = true

	var (
		stack []*node.Node
		root  *node.Node
	)

	for {
		token, err := dec.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return nil, component.NewError(KindParseFailure, parsePath(stack), "", "", fmt.Errorf("%w: %w", ErrParsingFailed, err))
		}

		switch t := token.(type) {
		case xml.StartElement:
			node := &node.Node{
				Type:       t.Name.Local,
				Attributes: t.Attr,
			}

			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				node.Parent = parent
				parent.Children = append(parent.Children, node)
			} else {
				root = node
			}

			stack = append(stack, node)
		case xml.CharData:
			if len(stack) > 0 {
				parent := stack[len(stack)-1]

				if _, isRaw := mjmlEndTags[parent.Type]; isRaw {
					// Raw content is protected with CDATA sections, so it must
					// be kept byte for byte.
					parent.Content += string(t)
					continue
				}

				parent.Content += string(bytes.TrimSpace(t))
			}
		case xml.Comment:
			// MJML turns every comment outside an ending tag into an mj-raw
			// node, so it is emitted verbatim where it stands and nowhere else
			// (a comment that is a direct child of <mjml> is dropped by the
			// renderer, like it is in MJML). Comments inside ending tags never
			// reach this branch: that content is protected as CDATA.
			if !keepComments || len(stack) == 0 {
				continue
			}

			parent := stack[len(stack)-1]
			parent.Children = append(parent.Children, &node.Node{
				Type:    component.RawTagName,
				Content: "<!--" + string(t) + "-->",
				Parent:  parent,
			})
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}

	if root == nil {
		return nil, component.NewError(KindParseFailure, parsePath(stack), "", "", fmt.Errorf("%w: no root element specified", ErrParsingFailed))
	}

	ensureHead(root)

	return root, nil
}

// fileStartRawContent returns the concatenated content of <mj-raw
// position="file-start"> elements that are direct children of <mjml>, in
// document order. MJML emits it before the doctype.
func fileStartRawContent(root *node.Node) string {
	var contents []string

	for _, child := range root.Children {
		if child.Type != component.RawTagName {
			continue
		}

		if position, ok := child.GetAttributeValue("position"); !ok || position != "file-start" {
			continue
		}

		contents = append(contents, strings.TrimSpace(child.Content))
	}

	return strings.Join(contents, "\n")
}

// includeTagName is the element that is replaced by the content of a file from
// the caller's fs.FS. The tag name is shared with package component, so the
// custom-component registry rejects it as a built-in element.
const includeTagName = component.IncludeTagName

// Include expansion limits. The reference implementation only detects cycles,
// by remembering which files a document was included from; it has no depth cap,
// so a diamond-shaped include graph (A includes B and C, both include D) grows
// exponentially. Both limits below are hard errors here, never warnings.
const (
	// maxIncludeDepth is how many includes may nest below the rendered
	// document: the document itself is depth 0, the files it includes depth 1.
	// MJML's own include tests nest three levels.
	maxIncludeDepth = 32
)

// includeResolver splices mj-include elements in a parsed document with content
// read from an fs.FS.
//
// Paths resolve relative to the directory of the file that contains the include,
// inside the fs.FS the caller passed, exactly like the reference implementation:
// it resolves includes against the directory of the file it is parsing, which is
// the included file itself for nested includes.
type includeResolver struct {
	ctx  context.Context
	fsys fs.FS
	// keepComments applies to every included document as well: comments are
	// author content, wherever the file is parsed from.
	keepComments bool
	// stack holds the documents currently being expanded, outermost first. It is
	// what makes cycle detection work: a file may not include one of its own
	// ancestors, and it also may not include itself.
	stack []string
	// pending collects the styles contributed by type="css" includes of the
	// document currently being expanded. MJML appends them to the end of that
	// document's head, after its own head children, and a document's head is in
	// turn merged after its includes' heads; collecting them per document keeps
	// the resulting rule order identical to MJML's.
	pending []*node.Node
}

// resolveIncludes expands every mj-include in root, in document order.
//
// name is the path of the document inside fsys and is only used to resolve
// relative include paths and to detect a file that includes the rendered
// document itself. A nil fsys means the caller has no filesystem to read from,
// so any mj-include is an error instead of a silently dropped element.
func resolveIncludes(ctx context.Context, fsys fs.FS, name string, root *node.Node, keepComments bool) error {
	if !containsIncludeTag(root) {
		return nil
	}

	if fsys == nil {
		return component.NewError(KindParseFailure, parsePath(nil), "", "", fmt.Errorf("%w: render through Renderer.RenderFS with a caller-provided filesystem", ErrIncludeNoFS))
	}

	resolver := &includeResolver{ctx: ctx, fsys: fsys, keepComments: keepComments, stack: []string{path.Clean(name)}}
	head := ensureHead(root)

	if err := resolver.expandChildren(root, path.Dir(name), head, 0); err != nil {
		return err
	}

	// Styles that the rendered document itself includes go last, after every
	// head merged from an included document.
	head.Children = append(head.Children, resolver.pending...)

	return nil
}

// stripIncludes removes every mj-include element from the document without
// resolving it. MJML's parser never creates a node for an include it is told to
// ignore, so the element and its content vanish from the rendered document and
// the files it names are never read.
func stripIncludes(n *node.Node) {
	children := n.Children[:0]

	for _, child := range n.Children {
		if child.Type == includeTagName {
			continue
		}

		stripIncludes(child)
		children = append(children, child)
	}

	n.Children = children
}

// expandChildren expands the includes among the children of parent and, before
// that, everything nested below the children that stay. dir is the directory of
// the file parent was parsed from, which is what relative include paths resolve
// against. head is the head of the rendered document: includes of type css and
// the head of an included document both append to it, like MJML does.
func (r *includeResolver) expandChildren(parent *node.Node, dir string, head *node.Node, depth int) error {
	if !containsIncludeTag(parent) {
		return nil
	}

	// Two passes: expand everything nested below the children that stay, then
	// splice in the files the includes name. That keeps the splice a plain
	// rebuild of Children, and nested includes are expanded relative to the
	// directory of the file that contains them, never relative to the document
	// that started the render.
	for _, child := range parent.Children {
		if child.Type == includeTagName {
			continue
		}

		if err := r.expandChildren(child, dir, head, depth); err != nil {
			return err
		}
	}

	expanded := make([]*node.Node, 0, len(parent.Children))

	for _, child := range parent.Children {
		if child.Type != includeTagName {
			expanded = append(expanded, child)

			continue
		}

		included, err := r.include(child, dir, head, depth)
		if err != nil {
			return err
		}

		expanded = append(expanded, included...)
	}

	parent.Children = expanded

	return nil
}

// include expands a single mj-include element and returns the nodes that take
// its place in the document. Includes of type css do not splice anything: their
// content becomes a style in the document head. depth is the nesting level of the
// include itself, counting the rendered document as 0.
func (r *includeResolver) include(n *node.Node, dir string, head *node.Node, depth int) ([]*node.Node, error) {
	if len(n.Children) > 0 || strings.TrimSpace(n.Content) != "" {
		return nil, component.NewError(
			KindParseFailure,
			nodePath(n),
			"",
			"",
			fmt.Errorf("%w: mj-include is replaced by the included file and must be empty", ErrParsingFailed),
		)
	}

	includePath, ok := n.GetAttributeValue("path")
	if !ok || includePath == "" {
		return nil, component.NewError(KindParseFailure, nodePath(n), "path", "", fmt.Errorf("%w: mj-include requires a path attribute", ErrIncludePath))
	}

	kind := "mjml"
	if value, found := n.GetAttributeValue("type"); found && value != "" {
		kind = value
	}

	switch kind {
	case "mjml", "html", "css":
	default:
		return nil, component.NewError(KindUnknownIncludeType, nodePath(n), "", kind, fmt.Errorf("%w: %q (want mjml, html or css)", ErrIncludeType, kind))
	}

	files, err := r.resolve(dir, includePath, nodePath(n))
	if err != nil {
		return nil, err
	}

	var spliced []*node.Node

	for _, file := range files {
		switch kind {
		case "html":
			if err := r.ctx.Err(); err != nil {
				return nil, err
			}

			content, err := r.read(file, nodePath(n))
			if err != nil {
				return nil, err
			}

			// type="html" is inserted verbatim, which is exactly what mj-raw
			// does at this position.
			spliced = append(spliced, &node.Node{
				Type:    component.RawTagName,
				Content: string(content),
				Parent:  n.Parent,
			})
		case "css":
			if err := r.ctx.Err(); err != nil {
				return nil, err
			}

			content, err := r.read(file, nodePath(n))
			if err != nil {
				return nil, err
			}

			style := &node.Node{
				Type:    component.StyleTagName,
				Content: string(content),
				Parent:  head,
			}

			// MJML turns css-inline="inline" into <mj-style inline="inline">,
			// so the content is inlined into the rendered markup.
			if inline, found := n.GetAttributeValue("css-inline"); found && inline == "inline" {
				style.SetAttribute("inline", "inline")
			}

			r.pending = append(r.pending, style)
		case "mjml":
			included, err := r.expandDocument(file, head, depth+1, nodePath(n))
			if err != nil {
				return nil, err
			}

			spliced = append(spliced, included...)
		}
	}

	for _, included := range spliced {
		included.Parent = n.Parent
	}

	return spliced, nil
}

// expandDocument parses an included file and returns the children of its mj-body,
// which is what replaces the mj-include element. The children of its mj-head are
// appended to the head of the rendered document instead, the same way MJML
// merges an included document's head.
func (r *includeResolver) expandDocument(file string, head *node.Node, depth int, at []string) ([]*node.Node, error) {
	if depth > maxIncludeDepth {
		return nil, component.NewError(KindParseFailure, at, "", file, fmt.Errorf("%w: %d levels at %s", ErrIncludeDepth, depth, file))
	}

	if slices.Contains(r.stack, file) {
		chain := append(slices.Clone(r.stack), file)

		return nil, component.NewError(KindParseFailure, at, "", file, fmt.Errorf("%w: %s", ErrIncludeCycle, strings.Join(chain, " -> ")))
	}

	if err := r.ctx.Err(); err != nil {
		return nil, err
	}

	content, err := r.read(file, at)
	if err != nil {
		return nil, err
	}

	// An included file may be a full document or a bare fragment. MJML wraps
	// anything without a literal <mjml> element in <mjml><mj-body>, so a
	// fragment contributes its elements at the position of the include.
	document := string(content)
	if !strings.Contains(document, "<mjml>") {
		document = "<mjml><mj-body>" + document + "</mj-body></mjml>"
	}

	root, err := parse([]byte(document), r.keepComments)
	if err != nil {
		// Attribute the failure to the include site, not only to where parsing
		// stopped inside the included file: that is where the author can act.
		// The inner *Error stays reachable through Err.
		return nil, component.NewError(
			KindParseFailure,
			at,
			"",
			file,
			fmt.Errorf("mjml: mj-include %s: %w", file, err),
		)
	}

	r.stack = append(r.stack, file)

	// Styles belong to this document until it is fully expanded.
	outerPending := r.pending
	r.pending = nil

	body := findDirectChild(root, component.BodyTagName)
	includedHead := findDirectChild(root, component.HeadTagName)
	dir := path.Dir(file)

	// The head is expanded first because it comes first in the document: MJML
	// collects the styles of a document in parse order, so a css include in the
	// head ends up before one in the body.
	if includedHead != nil {
		if err := r.expandChildren(includedHead, dir, head, depth); err != nil {
			return nil, err
		}
	}

	if body != nil {
		if err := r.expandChildren(body, dir, head, depth); err != nil {
			return nil, err
		}
	}

	styles := r.pending
	r.pending = outerPending
	r.stack = r.stack[:len(r.stack)-1]

	var spliced []*node.Node

	if body != nil {
		spliced = body.Children
		body.Children = nil
	}

	// The included head is merged before this document's styles so that the
	// cascade keeps MJML's order: head children first, included css last.
	if includedHead != nil {
		for _, child := range includedHead.Children {
			child.Parent = head
			head.Children = append(head.Children, child)
		}

		includedHead.Children = nil
	}

	head.Children = append(head.Children, styles...)

	return spliced, nil
}

// resolve turns the path attribute of an include into the files it names,
// relative to dir. Glob patterns are expanded with fs.Glob, in lexical order; the
// reference implementation has no pattern support, so the ordering is ours.
func (r *includeResolver) resolve(dir, includePath string, at []string) ([]string, error) {
	if includePath == "" {
		return nil, component.NewError(KindParseFailure, at, "path", includePath, fmt.Errorf("%w: mj-include requires a path attribute", ErrIncludePath))
	}

	if strings.HasPrefix(includePath, "/") {
		return nil, component.NewError(KindParseFailure, at, "path", includePath, fmt.Errorf("%w: %q is absolute", ErrIncludePath, includePath))
	}

	// The include path is joined before it is validated, so a partial may reach a
	// sibling directory with "../" while anything that would climb above the
	// filesystem root stays rejected: path.Join cleans the result, and
	// fs.ValidPath then rejects a leading "..". Nothing is ever read outside the
	// caller's fs.FS.
	joined := path.Join(dir, includePath)
	if !fs.ValidPath(joined) {
		return nil, component.NewError(KindParseFailure, at, "path", includePath, fmt.Errorf("%w: %q escapes the filesystem root", ErrIncludePath, includePath))
	}

	if !strings.ContainsAny(includePath, "*?[") {
		return []string{joined}, nil
	}

	matches, err := fs.Glob(r.fsys, joined)
	if err != nil {
		return nil, component.NewError(KindParseFailure, at, "path", includePath, fmt.Errorf("%w: %w", ErrIncludePath, err))
	}

	if len(matches) == 0 {
		return nil, component.NewError(KindParseFailure, at, "path", includePath, fmt.Errorf("%w: %s matches no files", ErrIncludePath, joined))
	}

	slices.Sort(matches)

	return matches, nil
}

// read reads an include target, wrapping the filesystem error with the path that
// was attempted.
func (r *includeResolver) read(file string, at []string) ([]byte, error) {
	content, err := fs.ReadFile(r.fsys, file)
	if err != nil {
		return nil, component.NewError(KindParseFailure, at, "", file, fmt.Errorf("mjml: mj-include %s: %w", file, err))
	}

	return content, nil
}

// parsePath returns the element path a parse failure is attributed to: the
// element that was open when parsing stopped, or the document element the
// parser failed to build when it never opened one.
func parsePath(stack []*node.Node) []string {
	if len(stack) > 0 {
		if segments := nodePath(stack[len(stack)-1]); len(segments) > 0 {
			return segments
		}
	}

	return []string{component.MJMLTagName}
}

// nodePath returns the ancestry of n from the document root, with the root
// <mjml> element omitted, like component element paths are. The rule is
// duplicated from component's elementPathSegments because that helper is
// unexported; keep the two in step.
func nodePath(n *node.Node) []string {
	var parts []string

	for current := n; current != nil; current = current.Parent {
		parts = append(parts, current.Type)
	}

	slices.Reverse(parts)

	if len(parts) > 1 && parts[0] == component.MJMLTagName {
		parts = parts[1:]
	}

	return parts
}

// containsIncludeTag reports whether n or any node below it is an mj-include.
// Documents without includes skip the expansion pass entirely.
func containsIncludeTag(n *node.Node) bool {
	if n.Type == includeTagName {
		return true
	}

	return slices.ContainsFunc(n.Children, containsIncludeTag)
}

// findDirectChild returns the first direct child of n with the given tag name.
func findDirectChild(n *node.Node, tag string) *node.Node {
	for _, child := range n.Children {
		if child.Type == tag {
			return child
		}
	}

	return nil
}

// ensureHead returns the mj-head of root, creating and prepending one if the
// document has none. MJML always renders a head, and includes of type css need a
// place to put their style.
func ensureHead(root *node.Node) *node.Node {
	if head := findDirectChild(root, component.HeadTagName); head != nil {
		return head
	}

	head := &node.Node{Type: component.HeadTagName, Parent: root}
	root.Children = slices.Insert(root.Children, 0, head)

	return head
}

// mjmlEndTags is a map of MJML tags whose inner content should be
// treated as a single raw string, not parsed into child nodes.
var mjmlEndTags = map[string]struct{}{
	component.RawTagName:            {},
	component.TitleTagName:          {},
	component.PreviewTagName:        {},
	component.BreakpointTagName:     {},
	component.StyleTagName:          {},
	component.TextTagName:           {},
	component.ButtonTagName:         {},
	component.TableTagName:          {},
	component.NavbarLinkTagName:     {},
	component.AccordionTextTagName:  {},
	component.AccordionTitleTagName: {},
	component.SocialElementTagName:  {},
}

// protectRawContent rewrites the inner content of raw-content elements into
// CDATA sections. Their content is arbitrary markup - not necessarily
// well-formed XML - and encoding/xml would otherwise reserialize it, inventing
// closing tags for void HTML elements (`<br>` becoming `<br></br>`).
func protectRawContent(input []byte) string {
	var b strings.Builder
	b.Grow(len(input) + len(input)/8)

	i := 0

	for i < len(input) {
		lt := bytes.IndexByte(input[i:], '<')
		if lt < 0 {
			b.Write(input[i:])

			break
		}

		lt += i
		b.Write(input[i:lt])

		switch {
		case bytes.HasPrefix(input[lt:], commentOpen):
			end := bytes.Index(input[lt+len(commentOpen):], commentClose)
			if end < 0 {
				b.Write(input[lt:])

				return b.String()
			}

			end += lt + len(commentOpen) + len(commentClose)
			b.Write(input[lt:end])
			i = end

			continue

		case bytes.HasPrefix(input[lt:], endTagOpen), bytes.HasPrefix(input[lt:], bangTagOpen):
			end := scanTagEnd(input, lt)
			if end < 0 {
				b.Write(input[lt:])

				return b.String()
			}

			b.Write(input[lt:end])
			i = end

			continue
		}

		nameEnd := lt + 1
		for nameEnd < len(input) && isXMLNameByte(input[nameEnd]) {
			nameEnd++
		}

		name := input[lt+1 : nameEnd]
		if len(name) == 0 {
			b.WriteByte('<')
			i = lt + 1

			continue
		}

		tagEnd := scanTagEnd(input, lt)
		if tagEnd < 0 {
			b.Write(input[lt:])

			break
		}

		b.Write(input[lt:tagEnd])
		i = tagEnd

		// A map lookup through string(name) does not allocate.
		if _, isRaw := mjmlEndTags[string(name)]; !isRaw {
			continue
		}

		if bytes.HasSuffix(bytes.TrimSpace(input[lt:tagEnd]), selfClosing) {
			continue
		}

		closeStart, closeEnd := findClosingTag(input, i, string(name))
		if closeStart < 0 {
			b.Write(input[i:])

			return b.String()
		}

		b.WriteString(protectCDATA(input[i:closeStart]))
		b.Write(input[closeStart:closeEnd])
		i = closeEnd
	}

	return b.String()
}

var (
	commentOpen  = []byte("<!--")
	commentClose = []byte("-->")
	endTagOpen   = []byte("</")
	bangTagOpen  = []byte("<!")
	selfClosing  = []byte("/>")
)

func scanTagEnd(input []byte, start int) int {
	var quote byte

	for i := start + 1; i < len(input); i++ {
		switch char := input[i]; {
		case quote != 0:
			if char == quote {
				quote = 0
			}
		case char == '"' || char == '\'':
			quote = char
		case char == '>':
			return i + 1
		}
	}

	return -1
}

// findClosingTag returns the span of the end tag that closes the element named
// name. The search is literal: raw content is arbitrary text, so a '<' inside it
// ("{ if item < 5 }") must not be mistaken for markup.
func findClosingTag(input []byte, from int, name string) (int, int) {
	needle := []byte("</" + name)
	i := from

	for {
		idx := bytes.Index(input[i:], needle)
		if idx < 0 {
			return -1, -1
		}

		idx += i

		// Guard against names sharing the prefix (</mj-text-x>).
		if after := idx + len(needle); after < len(input) && isXMLNameByte(input[after]) {
			i = after

			continue
		}

		end := scanTagEnd(input, idx)
		if end < 0 {
			return -1, -1
		}

		return idx, end
	}
}

func isXMLNameByte(char byte) bool {
	switch {
	case char >= 'a' && char <= 'z', char >= 'A' && char <= 'Z':
		return true
	case char >= '0' && char <= '9', char == '-', char == '_', char == ':', char == '.':
		return true
	default:
		return false
	}
}

// protectCDATA wraps content in CDATA sections, splitting any literal "]]>"
// so the result stays valid XML.
func protectCDATA(content []byte) string {
	if len(content) == 0 {
		return ""
	}

	return "<![CDATA[" + strings.ReplaceAll(string(content), "]]>", "]]]]><![CDATA[>") + "]]>"
}
