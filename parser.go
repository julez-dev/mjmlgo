package mjmlgo

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/julez-dev/mjmlgo/internal/component"
	"github.com/julez-dev/mjmlgo/internal/node"
)

var (
	ErrParsingFailed = errors.New("parsing MJML structure failed")
)

func parse(input io.Reader) (*node.Node, error) {
	fullBytes, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}

	dec := xml.NewDecoder(strings.NewReader(protectRawContent(string(fullBytes))))
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

			return nil, fmt.Errorf("%w: %w", ErrParsingFailed, err)
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
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		}
	}

	if root == nil {
		return nil, fmt.Errorf("%w: no root element specified", ErrParsingFailed)
	}

	if !slices.ContainsFunc(root.Children, func(e *node.Node) bool {
		return e.Type == component.HeadTagName
	}) {
		root.Children = slices.Insert(root.Children, 0, &node.Node{
			Type:   component.HeadTagName,
			Parent: root,
		})
	}

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

// mjmlEndTags is a map of MJML tags whose inner content should be
// treated as a single raw string, not parsed into child nodes.
var mjmlEndTags = map[string]struct{}{
	"mj-raw":             struct{}{},
	"mj-title":           struct{}{},
	"mj-preview":         struct{}{},
	"mj-breakpoint":      struct{}{},
	"mj-style":           struct{}{},
	"mj-text":            struct{}{},
	"mj-button":          struct{}{},
	"mj-table":           struct{}{},
	"mj-navbar-link":     struct{}{},
	"mj-accordion-text":  struct{}{},
	"mj-accordion-title": struct{}{},
	"mj-social-element":  struct{}{},
}

// protectRawContent rewrites the inner content of raw-content elements into
// CDATA sections. Their content is arbitrary markup - not necessarily
// well-formed XML - and encoding/xml would otherwise reserialize it, inventing
// closing tags for void HTML elements (`<br>` becoming `<br></br>`).
func protectRawContent(input string) string {
	var b strings.Builder
	b.Grow(len(input) + len(input)/8)

	i := 0

	for i < len(input) {
		lt := strings.IndexByte(input[i:], '<')
		if lt < 0 {
			b.WriteString(input[i:])

			break
		}

		lt += i
		b.WriteString(input[i:lt])

		switch {
		case strings.HasPrefix(input[lt:], "<!--"):
			end := strings.Index(input[lt+4:], "-->")
			if end < 0 {
				b.WriteString(input[lt:])

				return b.String()
			}

			end += lt + 4 + len("-->")
			b.WriteString(input[lt:end])
			i = end

			continue

		case strings.HasPrefix(input[lt:], "</"), strings.HasPrefix(input[lt:], "<!"):
			end := scanTagEnd(input, lt)
			if end < 0 {
				b.WriteString(input[lt:])

				return b.String()
			}

			b.WriteString(input[lt:end])
			i = end

			continue
		}

		nameEnd := lt + 1
		for nameEnd < len(input) && isXMLNameByte(input[nameEnd]) {
			nameEnd++
		}

		name := input[lt+1 : nameEnd]
		if name == "" {
			b.WriteByte('<')
			i = lt + 1

			continue
		}

		tagEnd := scanTagEnd(input, lt)
		if tagEnd < 0 {
			b.WriteString(input[lt:])

			break
		}

		b.WriteString(input[lt:tagEnd])
		i = tagEnd

		if _, isRaw := mjmlEndTags[name]; !isRaw {
			continue
		}

		if strings.HasSuffix(strings.TrimSpace(input[lt:tagEnd]), "/>") {
			continue
		}

		closeStart, closeEnd := findClosingTag(input, i, name)
		if closeStart < 0 {
			b.WriteString(input[i:])

			return b.String()
		}

		b.WriteString(protectCDATA(input[i:closeStart]))
		b.WriteString(input[closeStart:closeEnd])
		i = closeEnd
	}

	return b.String()
}

// scanTagEnd returns the index just after the '>' that closes the tag starting
// at start, ignoring '>' inside quoted attribute values. It returns -1 when the
// tag is unterminated.
func scanTagEnd(input string, start int) int {
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
func findClosingTag(input string, from int, name string) (int, int) {
	needle := "</" + name
	i := from

	for {
		idx := strings.Index(input[i:], needle)
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
func protectCDATA(content string) string {
	if content == "" {
		return ""
	}

	return "<![CDATA[" + strings.ReplaceAll(content, "]]>", "]]]]><![CDATA[>") + "]]>"
}
