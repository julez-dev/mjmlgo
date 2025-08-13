package mjmlgo

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

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
				parent.Children = append(parent.Children, node)
			} else {
				root = node
			}

			stack = append(stack, node)
		case xml.CharData:
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				content := string(bytes.TrimSpace(t))
				parent.Content = content
			}
		case xml.Comment:
			// Check if this comment is one of our placeholders.
			comment := string(t)
			if after, ok := strings.CutPrefix(comment, "RAW_PLACEHOLDER_"); ok {
				// Parse the index from the placeholder string.
				indexStr := strings.TrimSpace(after)
				index, err := strconv.Atoi(indexStr)
				if err != nil || index >= len(rawContents) {
					// This shouldn't happen if the regex is correct.
					continue
				}

				// Create a node for the raw content.
				node := &node.Node{
					Type:    "mj-raw",
					Content: rawContents[index], // Substitute the original content back.
				}

				if len(stack) > 0 {
					parent := stack[len(stack)-1]
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

	return root, nil
}
