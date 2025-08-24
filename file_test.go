package mjmlgo

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/Boostport/mjml-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

var htmlCommentRegex = regexp.MustCompile(`(?s)<!--.*?-->`)

func removeHTMLComments(html string) string {
	return htmlCommentRegex.ReplaceAllString(html, "")
}

func TestMJMLFiles(t *testing.T) {
	t.Parallel()

	err := fs.WalkDir(os.DirFS("./testdata"), ".", func(fpath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if path.Ext(fpath) != ".mjml" {
			return nil
		}

		f, err := os.ReadFile(filepath.Join("./testdata", fpath))
		if err != nil {
			return err
		}

		t.Run(fpath, func(t *testing.T) {
			html1, err := RenderMJML(bytes.NewReader(f))
			require.NoError(t, err)
			html1 = removeHTMLComments(html1)

			html2, err := mjml.ToHTML(context.Background(), string(f))
			require.NoError(t, err)
			html2 = removeHTMLComments(html2)

			n1, err := html.Parse(strings.NewReader(html1))
			require.NoError(t, err)

			n2, err := html.Parse(strings.NewReader(html2))
			require.NoError(t, err)

			// os.WriteFile(fpath+"1.html", []byte(html1), 0644)
			// os.WriteFile(fpath+"2.html", []byte(html2), 0644)
			// var o bytes.Buffer
			// PrettyPrint(&o, n1)
			// fmt.Println(o.String())

			// var o2 bytes.Buffer
			// PrettyPrint(&o2, n2)
			// fmt.Println(o2.String())

			assert.NoError(t, compareNodes(n1, n2))
		})

		return nil
	})

	require.NoError(t, err)
}

func matchStyle(s1, s2 string) bool {
	splits1, splits2 := strings.Split(s1, ";"), strings.Split(s2, ";")

	styles1 := make(map[string]string)
	for _, style := range splits1 {
		parts := strings.Split(style, ":")
		if len(parts) == 2 {
			styles1[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	styles2 := make(map[string]string)
	for _, style := range splits2 {
		parts := strings.Split(style, ":")
		if len(parts) == 2 {
			styles2[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	for k, v := range styles1 {
		if styles2[k] != v {
			return false
		}
	}

	return true
}

func compareLineByLine(t1, t2 string) error {
	splits1, splits2 := strings.Split(t1, "\n"), strings.Split(t2, "\n")
	if len(splits1) != len(splits2) {
		return fmt.Errorf("line count mismatch")
	}

	for i := range splits1 {
		if strings.TrimSpace(splits1[i]) != strings.TrimSpace(splits2[i]) {
			return fmt.Errorf("line %d mismatch", i)
		}
	}

	return nil
}

func compareNodes(n1, n2 *html.Node) error {
	if n1 == nil || n2 == nil {
		return fmt.Errorf("n1 or n2 nil")
	}

	// Compare node type and data (e.g., tag name or text content).
	if n1.Type != n2.Type {
		return fmt.Errorf("type mismatch n1 %+v, n2 %+v", n1.Type, n2.Type)
	}

	if n1.Data == "style" {
		return nil
	}

	// if err := compareLineByLine(n1.Data, n2.Data); err != nil {
	// 	return fmt.Errorf("n1 data %s different from n2 data %s", strings.TrimSpace(n1.Data), strings.TrimSpace(n2.Data))
	// }

	// For ElementNodes, compare attributes.
	if n1.Type == html.ElementNode {
		n1.Attr = removeEmptyAttr(n1.Attr)
		n2.Attr = removeEmptyAttr(n2.Attr)

		if len(n1.Attr) != len(n2.Attr) {
			return fmt.Errorf("n1 (%s) attr (%+v) len %d different from n2 (%s) attr (%+v) len %d", n1.Data, n1.Attr, len(n1.Attr), n2.Data, n2.Attr, len(n2.Attr))
		}
		// Create a map of attributes for n2 for easy lookup.
		attrs2 := make(map[string]string)
		for _, attr := range n2.Attr {
			attrs2[attr.Key] = attr.Val
		}

		// Check if all attributes in n1 exist and have the same value in n2.
		for _, attr1 := range n1.Attr {
			val2, ok := attrs2[attr1.Key]

			if !ok {
				return fmt.Errorf("n1 (%s) key %q not existing in n2 (%s)", n1.Data, attr1.Key, n2.Data)
			}

			if attr1.Key == "style" {
				if !matchStyle(attr1.Val, val2) {
					return fmt.Errorf("n1 (%s) style val %s different in n2 (%s) %s", n1.Data, attr1.Val, n2.Data, val2)
				}
			} else {
				if attr1.Val != val2 {
					return fmt.Errorf("n1 (%s) val %s different in n2 (%s) %s", n1.Data, attr1.Val, n2.Data, val2)
				}
			}
		}
	}

	removeEmptyChildren(n1)
	removeEmptyChildren(n2)

	n1Childs, n2Childs := slices.Collect(n1.ChildNodes()), slices.Collect(n2.ChildNodes())

	for i := range len(n1Childs) {
		if err := compareNodes(n1Childs[i], n2Childs[i]); err != nil {
			return err
		}
	}

	return nil
}

func removeEmptyAttr(attrs []html.Attribute) []html.Attribute {
	attrToIgnore := []string{
		"alt",
		"style",
	}
	return slices.DeleteFunc(attrs, func(e html.Attribute) bool {
		if slices.Contains(attrToIgnore, e.Key) && e.Val == "" {
			return true
		}

		return false
	})
}

func removeEmptyChildren(n *html.Node) {
	var remove []*html.Node

	for c := range n.ChildNodes() {
		if c.Type == html.TextNode && strings.TrimSpace(c.Data) == "" {
			remove = append(remove, c)
		}
	}

	for _, c := range remove {
		n.RemoveChild(c)
	}
}
