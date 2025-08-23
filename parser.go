package mjmlgo

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/julez-dev/mjmlgo/component"
	"github.com/julez-dev/mjmlgo/node"
)

// rawContentPlaceholderFormat defines the structure for the placeholder comment.
const rawContentPlaceholderFormat = "RAW_PLACEHOLDER_%d"

// rawTagRegex is used to find <mj-raw> blocks and capture their inner content.
// The (?s) flag allows '.' to match newline characters.
var rawTagRegex = regexp.MustCompile(`(?s)<mj-raw>(.*?)</mj-raw>`)

var (
	ErrParsingFailed = errors.New("parsing MJML structure failed")
)

func parse(input io.Reader) (*node.Node, error) {
	fullBytes, err := io.ReadAll(input)
	if err != nil {
		return nil, err
	}

	mjmlString := string(fullBytes)
	var rawContents []string
	processedMJML := rawTagRegex.ReplaceAllStringFunc(mjmlString, func(match string) string {
		// Extract the inner content from the full match.
		innerContent := rawTagRegex.FindStringSubmatch(match)[1]
		// Store the raw inner content.
		rawContents = append(rawContents, innerContent)
		// Return a placeholder comment. The index will correspond to the slice.
		placeholder := fmt.Sprintf(rawContentPlaceholderFormat, len(rawContents)-1)
		return fmt.Sprintf("<!--%s-->", placeholder)
	})

	dec := xml.NewDecoder(strings.NewReader(processedMJML))
	dec.Strict = false

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

			if _, has := mjmlEndTags[node.Type]; has {
				s, err := streamInnerRawContent(dec, t)
				if err != nil {
					return nil, err
				}

				node.Content = s
				// Attach to parent
				if len(stack) > 0 {
					parent := stack[len(stack)-1]
					node.Parent = parent
					parent.Children = append(parent.Children, node)
				} else {
					root = node
				}

				// Do NOT push to stack, since streamInnerRawContent already consumes the end tag
				continue
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
				content := string(bytes.TrimSpace(t))
				parent.Content += content
			}
		case xml.Comment:
			// Check if this comment is one of our placeholders
			comment := string(t)
			if after, ok := strings.CutPrefix(comment, "RAW_PLACEHOLDER_"); ok {
				// Parse the index from the placeholder string.
				indexStr := strings.TrimSpace(after)
				index, err := strconv.Atoi(indexStr)
				if err != nil || index >= len(rawContents) {
					// This shouldn't happen
					continue
				}

				node := &node.Node{
					Type:    "mj-raw",
					Content: rawContents[index], // the original content
				}

				if len(stack) > 0 {
					parent := stack[len(stack)-1]
					node.Parent = parent // Set parent node for raw content
					parent.Children = append(parent.Children, node)
				}
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

// mjmlEndTags is a map of MJML tags whose inner content should be
// treated as a single raw string, not parsed into child nodes.
var mjmlEndTags = map[string]struct{}{
	"mj-text":            struct{}{},
	"mj-button":          struct{}{},
	"mj-table":           struct{}{},
	"mj-navbar-link":     struct{}{},
	"mj-accordion-text":  struct{}{},
	"mj-accordion-title": struct{}{},
	"mj-social-element":  struct{}{},
}

// streamInnerRawContent captures the inner content of an element as a raw string.
// It starts after the initial start tag and stops before the final end tag.
func streamInnerRawContent(decoder *xml.Decoder, startElement xml.StartElement) (string, error) {
	var builder strings.Builder
	depth := 1

	for {
		token, err := decoder.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return "", fmt.Errorf("unexpected EOF while streaming raw content for <%s>", startElement.Name.Local)
			}

			return "", err
		}

		// Check for the final EndElement before writing, so we don't include it
		if end, ok := token.(xml.EndElement); ok && end.Name.Local == startElement.Name.Local {
			depth--
			if depth == 0 {
				return builder.String(), nil // We're done
			}
		}

		// Re-serialize the token back to a string.
		switch se := token.(type) {
		case xml.StartElement:
			if se.Name.Local == startElement.Name.Local {
				depth++
			}
			builder.WriteString("<" + se.Name.Local)
			for _, attr := range se.Attr {
				builder.WriteString(fmt.Sprintf(` %s="%s"`, attr.Name.Local, attr.Value))
			}
			builder.WriteString(">")
		case xml.EndElement:
			builder.WriteString("</" + se.Name.Local + ">")
		case xml.CharData:
			builder.Write(se)
		case xml.Comment:
			builder.WriteString(fmt.Sprintf("<!--%s-->", string(se)))
		}
	}
}
