package mjmlgo

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/julez-dev/mjmlgo/internal/node"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	t.Parallel()

	t.Run("mj-raw", func(t *testing.T) {
		const input = `<mjml>
			<mj-body>
				<mj-raw>Raw </div></mj-raw>
			</mj-body>
		</mjml>
		`

		n, err := parse(strings.NewReader(input))
		require.NoError(t, err)

		var rawContent string
		for _, v := range n.Children {
			if v.Type == "mj-body" {
				for _, v2 := range v.Children {
					if v2.Type == "mj-raw" {
						rawContent = v2.Content
					}
				}
			}
		}

		require.Equal(t, "Raw </div>", rawContent)
	})

	t.Run("mj-raw-with-attributes", func(t *testing.T) {
		const input = `<mjml>
			<mj-body>
				<mj-raw position="file-start" class="raw-class"><div>raw content</div></mj-raw>
			</mj-body>
		</mjml>`

		n, err := parse(strings.NewReader(input))
		require.NoError(t, err)

		body := findChild(t, n, "mj-body")
		raw := findChild(t, body, "mj-raw")

		position, ok := raw.GetAttributeValue("position")
		require.True(t, ok)
		require.Equal(t, "file-start", position)

		class, ok := raw.GetAttributeValue("class")
		require.True(t, ok)
		require.Equal(t, "raw-class", class)

		require.Equal(t, "<div>raw content</div>", raw.Content)
	})

	t.Run("file-start-raw-content", func(t *testing.T) {
		const input = `<mjml>
			<mj-raw position="file-start"><!-- first --></mj-raw>
			<mj-raw position="file-start"><!-- second --></mj-raw>
			<mj-raw><div>in body flow</div></mj-raw>
			<mj-body></mj-body>
		</mjml>`

		n, err := parse(strings.NewReader(input))
		require.NoError(t, err)

		require.Equal(t, "<!-- first -->\n<!-- second -->", fileStartRawContent(n))
	})

	t.Run("raw-content-is-kept-verbatim", func(t *testing.T) {
		const input = `<mjml><mj-body><mj-section><mj-column>` +
			`<mj-text><p>x</p><br><!--c--><br></mj-text>` +
			`<mj-raw><div data-x="1">a &amp; b</div></mj-raw>` +
			`</mj-column></mj-section></mj-body></mjml>`

		n, err := parse(strings.NewReader(input))
		require.NoError(t, err)

		contents := map[string]string{}
		walkNodes(n, func(node *node.Node) {
			if node.Type == "mj-text" || node.Type == "mj-raw" {
				contents[node.Type] = node.Content
			}
		})

		require.Equal(t, `<p>x</p><br><!--c--><br>`, contents["mj-text"])
		require.Equal(t, `<div data-x="1">a &amp; b</div>`, contents["mj-raw"])
	})

	t.Run("mj-end-tags", func(t *testing.T) {
		const input = `<mjml><mj-text><h1>Test</h1></mj-text></mjml>`

		n, err := parse(strings.NewReader(input))
		require.NoError(t, err)

		var rawContent string
		for _, v := range n.Children {
			if v.Type == "mj-text" {
				rawContent = v.Content
			}
		}

		require.Equal(t, "<h1>Test</h1>", rawContent)
	})
}

// findChild returns the first direct child of n with the given tag name.
func findChild(t *testing.T, n *node.Node, tag string) *node.Node {
	t.Helper()

	for _, child := range n.Children {
		if child.Type == tag {
			return child
		}
	}

	t.Fatalf("no <%s> child of <%s>", tag, n.Type)
	return nil
}

// BenchmarkParse isolates the MJML parser.
func BenchmarkParse(b *testing.B) {
	forEachParityFixture(b, func(name, fixture string) {
		b.Run(name, func(b *testing.B) {
			input, err := os.ReadFile(fixture)
			if err != nil {
				b.Fatal(err)
			}

			b.ReportAllocs()
			b.SetBytes(int64(len(input)))

			for b.Loop() {
				if _, err := parse(bytes.NewReader(input)); err != nil {
					b.Skipf("fixture does not parse: %v", err)
				}
			}
		})
	})
}

// FuzzParse asserts the parser never panics and either returns an error or a
// root node; it deliberately does not require MJML validity.
func FuzzParse(f *testing.F) {
	f.Add(`<mjml><mj-body><mj-raw><div></mj-raw></mj-body></mjml>`)
	f.Add(`<mjml><mj-body><mj-text><h1>Test</h1></mj-text></mj-body></mjml>`)
	f.Add(`<mjml><mj-body><mj-section attr="value"><mj-column /></mj-section></mj-body></mjml>`)
	f.Add(`<mjml></mjml><mjml></mjml>`)
	f.Add(`<mjml><mj-body>`)
	f.Add(`not mjml`)

	f.Fuzz(func(t *testing.T, source string) {
		checkFuzzInput(t, source)

		root, err := parse(bytes.NewReader([]byte(source)))
		if err != nil {
			return
		}

		if root == nil {
			t.Fatalf("parse returned nil root without an error for input %q", source)
		}
	})
}

// walkNodes calls fn for n and all its descendants.
func walkNodes(n *node.Node, fn func(*node.Node)) {
	fn(n)

	for _, child := range n.Children {
		walkNodes(child, fn)
	}
}

// includeTestFS builds an in-memory filesystem from slash-separated paths to
// file contents.
func includeTestFS(files map[string]string) fstest.MapFS {
	fsys := make(fstest.MapFS, len(files))
	for name, content := range files {
		fsys[name] = &fstest.MapFile{Data: []byte(content)}
	}

	return fsys
}

// parseWithIncludes parses document and expands its mj-include elements against
// fsys, exactly like the renderer does.
func parseWithIncludes(t *testing.T, fsys fs.FS, name, document string) (*node.Node, error) {
	t.Helper()

	root, err := parse(strings.NewReader(document))
	require.NoError(t, err)

	return root, resolveIncludes(context.Background(), fsys, name, root)
}

// describeTree lists the tag names of n and its descendants in document order.
func describeTree(n *node.Node) string {
	var types []string

	walkNodes(n, func(node *node.Node) {
		types = append(types, node.Type)
	})

	return strings.Join(types, " ")
}

// TestIncludeResolution covers how mj-include finds its files: relative to the
// including file, nested, through glob patterns, and with every level resolved
// against the fs.FS the caller passed.
func TestIncludeResolution(t *testing.T) {
	t.Parallel()

	const document = `<mjml><mj-body><mj-include path="partials/header.mjml" /></mj-body></mjml>`

	t.Run("relative-to-the-including-file", func(t *testing.T) {
		t.Parallel()

		fsys := includeTestFS(map[string]string{
			"partials/header.mjml":      `<mj-section><mj-column><mj-include path="deep/leaf.mjml" /></mj-column></mj-section>`,
			"partials/deep/leaf.mjml":   `<mj-text>Leaf</mj-text>`,
			"partials/deep/deeper.mjml": `<mj-text>Deeper</mj-text>`,
		})

		root, err := parseWithIncludes(t, fsys, "main.mjml", document)
		require.NoError(t, err)

		// A nested include resolves against its own directory, not the
		// directory of the document that started the render.
		require.Equal(t,
			"mjml mj-head mj-body mj-section mj-column mj-text",
			describeTree(root),
		)

		body := findChild(t, root, "mj-body")
		section := findChild(t, body, "mj-section")
		require.Same(t, body, section.Parent, "spliced nodes must point at the node that replaced the include's parent")
	})

	t.Run("document-in-a-subdirectory", func(t *testing.T) {
		t.Parallel()

		// The rendered document lives in a subdirectory, so its includes
		// resolve below it, while a partial may still reach a shared sibling
		// one level up.
		const subDocument = `<mjml><mj-body><mj-include path="partials/header.mjml" /></mj-body></mjml>`

		fsys := includeTestFS(map[string]string{
			"pages/partials/header.mjml": `<mj-section><mj-column><mj-include path="../../../shared/footer.mjml" /></mj-column></mj-section>`,
		})

		_, err := parseWithIncludes(t, fsys, "pages/main.mjml", subDocument)
		require.ErrorIs(t, err, ErrIncludePath, "a partial may not escape the filesystem root")
	})

	t.Run("shared-sibling-via-parent-directory", func(t *testing.T) {
		t.Parallel()

		const subDocument = `<mjml><mj-body><mj-include path="partials/header.mjml" /></mj-body></mjml>`

		fsys := includeTestFS(map[string]string{
			"pages/partials/header.mjml": `<mj-section><mj-column><mj-include path="../shared/footer.mjml" /></mj-column></mj-section>`,
			"pages/shared/footer.mjml":   `<mj-text>Shared</mj-text>`,
		})

		root, err := parseWithIncludes(t, fsys, "pages/main.mjml", subDocument)
		require.NoError(t, err)

		require.Equal(t,
			"mjml mj-head mj-body mj-section mj-column mj-text",
			describeTree(root),
		)
	})

	t.Run("glob-matches-in-lexical-order", func(t *testing.T) {
		t.Parallel()

		fsys := includeTestFS(map[string]string{
			"partials/c-third.mjml":  `<mj-text>third</mj-text>`,
			"partials/a-first.mjml":  `<mj-text>first</mj-text>`,
			"partials/b-second.mjml": `<mj-text>second</mj-text>`,
			"partials/skipped.txt":   `<mj-text>not mjml</mj-text>`,
		})

		const globDocument = `<mjml><mj-body><mj-include path="partials/*.mjml" /></mj-body></mjml>`

		root, err := parseWithIncludes(t, fsys, "main.mjml", globDocument)
		require.NoError(t, err)

		var texts []string

		walkNodes(root, func(node *node.Node) {
			if node.Type == "mj-text" {
				texts = append(texts, node.Content)
			}
		})

		require.Equal(t, []string{"first", "second", "third"}, texts)
	})

	t.Run("glob-without-matches", func(t *testing.T) {
		t.Parallel()

		const globDocument = `<mjml><mj-body><mj-include path="partials/*.mjml" /></mj-body></mjml>`

		_, err := parseWithIncludes(t, includeTestFS(map[string]string{
			"partials/header.txt": "nothing to include",
		}), "main.mjml", globDocument)

		require.ErrorIs(t, err, ErrIncludePath)
		require.Contains(t, err.Error(), "matches no files")
	})

	t.Run("same-file-included-twice", func(t *testing.T) {
		t.Parallel()

		const document = `<mjml><mj-body>` +
			`<mj-include path="partials/header.mjml" />` +
			`<mj-include path="partials/header.mjml" />` +
			`</mj-body></mjml>`

		fsys := includeTestFS(map[string]string{
			"partials/header.mjml": `<mj-section><mj-column><mj-text>Header</mj-text></mj-column></mj-section>`,
		})

		root, err := parseWithIncludes(t, fsys, "main.mjml", document)
		require.NoError(t, err, "the same file may be included twice as long as it is not its own ancestor")

		require.Equal(t,
			"mjml mj-head mj-body mj-section mj-column mj-text mj-section mj-column mj-text",
			describeTree(root),
		)
	})

	t.Run("document-without-includes-is-untouched", func(t *testing.T) {
		t.Parallel()

		const plain = `<mjml><mj-body><mj-section><mj-column><mj-text>No includes</mj-text></mj-column></mj-section></mj-body></mjml>`

		// A nil filesystem is fine as long as nothing includes anything.
		root, err := parseWithIncludes(t, nil, "main.mjml", plain)
		require.NoError(t, err)

		require.Equal(t, "mjml mj-head mj-body mj-section mj-column mj-text", describeTree(root))
	})
}

// TestIncludeTypes covers the four ways an include can contribute content: as an
// MJML fragment, as a full MJML document, verbatim HTML and CSS in the head.
func TestIncludeTypes(t *testing.T) {
	t.Parallel()

	t.Run("mjml-fragment-inside-a-column", func(t *testing.T) {
		t.Parallel()

		const document = `<mjml><mj-body><mj-section><mj-column>` +
			`<mj-include path="partials/text.mjml" />` +
			`</mj-column></mj-section></mj-body></mjml>`

		fsys := includeTestFS(map[string]string{
			"partials/text.mjml": `<mj-text>From a fragment</mj-text><mj-divider />`,
		})

		root, err := parseWithIncludes(t, fsys, "main.mjml", document)
		require.NoError(t, err)

		require.Equal(t,
			"mjml mj-head mj-body mj-section mj-column mj-text mj-divider",
			describeTree(root),
		)
	})

	t.Run("mjml-fragment-inside-the-head", func(t *testing.T) {
		t.Parallel()

		const document = `<mjml><mj-head><mj-include path="partials/head.mjml" /></mj-head>` +
			`<mj-body><mj-section><mj-column><mj-text>Body</mj-text></mj-column></mj-section></mj-body></mjml>`

		fsys := includeTestFS(map[string]string{
			"partials/head.mjml": `<mj-style>.from-head-include { color: #0000ff; }</mj-style>`,
		})

		root, err := parseWithIncludes(t, fsys, "main.mjml", document)
		require.NoError(t, err)

		head := findChild(t, root, "mj-head")
		style := findChild(t, head, "mj-style")

		require.Equal(t, ".from-head-include { color: #0000ff; }", style.Content)
		require.Same(t, head, style.Parent)
	})

	t.Run("full-document", func(t *testing.T) {
		t.Parallel()

		const document = `<mjml><mj-head><mj-title>T</mj-title></mj-head>` +
			`<mj-body><mj-include path="partials/page.mjml" /></mj-body></mjml>`

		fsys := includeTestFS(map[string]string{
			"partials/page.mjml": `<mjml>
				<mj-head>
					<mj-preview>Included preview</mj-preview>
					<mj-style>.included { color: #00ff00; }</mj-style>
				</mj-head>
				<mj-body>
					<mj-section><mj-column><mj-text>Included body</mj-text></mj-column></mj-section>
				</mj-body>
			</mjml>`,
		})

		root, err := parseWithIncludes(t, fsys, "main.mjml", document)
		require.NoError(t, err)

		// The included document's head is merged into the rendered document's
		// head, its body children take the position of the include.
		require.Equal(t,
			"mjml mj-head mj-title mj-preview mj-style mj-body mj-section mj-column mj-text",
			describeTree(root),
		)
	})

	t.Run("html", func(t *testing.T) {
		t.Parallel()

		const document = `<mjml><mj-body><mj-include path="partials/footer.html" type="html" /></mj-body></mjml>`

		fsys := includeTestFS(map[string]string{
			"partials/footer.html": `<div class="footer">Verbatim <span>markup</span></div>`,
		})

		root, err := parseWithIncludes(t, fsys, "main.mjml", document)
		require.NoError(t, err)

		body := findChild(t, root, "mj-body")
		raw := findChild(t, body, "mj-raw")

		require.Equal(t, `<div class="footer">Verbatim <span>markup</span></div>`, raw.Content)
	})

	t.Run("css-order-in-the-head", func(t *testing.T) {
		t.Parallel()

		// MJML appends the styles of type="css" includes to the end of the head
		// they belong to, after that head's own children, and a document's head
		// children come before the styles of the document that includes it.
		const document = `<mjml>
			<mj-head>
				<mj-style>.doc-a {}</mj-style>
				<mj-include path="partials/head.css" type="css" />
				<mj-style>.doc-c {}</mj-style>
			</mj-head>
			<mj-body>
				<mj-include path="partials/page.mjml" />
				<mj-include path="partials/body.css" type="css" />
			</mj-body>
		</mjml>`

		fsys := includeTestFS(map[string]string{
			"partials/head.css": `.head {}`,
			"partials/body.css": `.body {}`,
			"partials/page.mjml": `<mjml>
				<mj-head>
					<mj-style>.page-a {}</mj-style>
					<mj-include path="page-head.css" type="css" />
				</mj-head>
				<mj-body>
					<mj-section><mj-column><mj-text>Page</mj-text></mj-column></mj-section>
					<mj-include path="page-body.css" type="css" />
				</mj-body>
			</mjml>`,
			"partials/page-head.css": `.page-head {}`,
			"partials/page-body.css": `.page-body {}`,
		})

		root, err := parseWithIncludes(t, fsys, "main.mjml", document)
		require.NoError(t, err)

		var styles []string

		walkNodes(root, func(n *node.Node) {
			if n.Type == "mj-style" {
				styles = append(styles, n.Content)
			}
		})

		require.Equal(t, []string{
			".doc-a {}",
			".doc-c {}",
			".page-a {}",
			".page-head {}",
			".page-body {}",
			".head {}",
			".body {}",
		}, styles)
	})

	t.Run("css", func(t *testing.T) {
		t.Parallel()

		const document = `<mjml><mj-body>` +
			`<mj-include path="partials/theme.css" type="css" />` +
			`<mj-include path="partials/inline.css" type="css" css-inline="inline" />` +
			`</mj-body></mjml>`

		fsys := includeTestFS(map[string]string{
			"partials/theme.css":  `.theme { color: #ff0000; }`,
			"partials/inline.css": `.inlined { font-size: 16px; }`,
		})

		root, err := parseWithIncludes(t, fsys, "main.mjml", document)
		require.NoError(t, err)

		head := findChild(t, root, "mj-head")
		require.Equal(t, "mjml mj-head mj-style mj-style mj-body", describeTree(root))

		require.Equal(t, `.theme { color: #ff0000; }`, head.Children[0].Content)

		inline, ok := head.Children[1].GetAttributeValue("inline")
		require.True(t, ok)
		require.Equal(t, "inline", inline)

		// A css include contributes nothing at its own position.
		require.Empty(t, findChild(t, root, "mj-body").Children)
	})
}

// TestIncludeErrors covers the hard errors: cycle detection, the depth cap,
// paths that leave the filesystem, missing files and malformed includes.
func TestIncludeErrors(t *testing.T) {
	t.Parallel()

	const document = `<mjml><mj-body><mj-include path="%s" /></mj-body></mjml>`

	t.Run("self-inclusion", func(t *testing.T) {
		t.Parallel()

		fsys := includeTestFS(map[string]string{
			"partials/self.mjml": `<mj-include path="self.mjml" />`,
		})

		_, err := parseWithIncludes(t, fsys, "main.mjml",
			`<mjml><mj-body><mj-include path="partials/self.mjml" /></mj-body></mjml>`)

		require.ErrorIs(t, err, ErrIncludeCycle)
		require.Contains(t, err.Error(), "partials/self.mjml -> partials/self.mjml")
	})

	t.Run("indirect-cycle", func(t *testing.T) {
		t.Parallel()

		fsys := includeTestFS(map[string]string{
			"partials/a.mjml": `<mj-include path="b.mjml" />`,
			"partials/b.mjml": `<mj-include path="a.mjml" />`,
		})

		_, err := parseWithIncludes(t, fsys, "main.mjml",
			`<mjml><mj-body><mj-include path="partials/a.mjml" /></mj-body></mjml>`)

		require.ErrorIs(t, err, ErrIncludeCycle)
		require.Contains(t, err.Error(),
			"partials/a.mjml -> partials/b.mjml -> partials/a.mjml")
	})

	t.Run("document-includes-itself", func(t *testing.T) {
		t.Parallel()

		fsys := includeTestFS(map[string]string{
			"main.mjml": `<mjml><mj-body><mj-include path="main.mjml" /></mj-body></mjml>`,
		})

		_, err := parseWithIncludes(t, fsys, "main.mjml",
			`<mjml><mj-body><mj-include path="main.mjml" /></mj-body></mjml>`)

		require.ErrorIs(t, err, ErrIncludeCycle)
	})

	t.Run("depth-limit", func(t *testing.T) {
		t.Parallel()

		chain := func(levels int) fstest.MapFS {
			files := make(map[string]string, levels)

			for level := 1; level < levels; level++ {
				files[fmt.Sprintf("partials/level-%d.mjml", level)] =
					fmt.Sprintf(`<mj-include path="level-%d.mjml" />`, level+1)
			}

			files[fmt.Sprintf("partials/level-%d.mjml", levels)] = `<mj-text>Leaf</mj-text>`

			return includeTestFS(files)
		}

		const doc = `<mjml><mj-body><mj-include path="partials/level-1.mjml" /></mj-body></mjml>`

		_, err := parseWithIncludes(t, chain(maxIncludeDepth), "main.mjml", doc)
		require.NoError(t, err, "includes nested exactly maxIncludeDepth deep must work")

		_, err = parseWithIncludes(t, chain(maxIncludeDepth+1), "main.mjml", doc)
		require.ErrorIs(t, err, ErrIncludeDepth)
	})

	t.Run("escaping-paths", func(t *testing.T) {
		t.Parallel()

		for _, includePath := range []string{
			"../outside.mjml",
			"partials/../../outside.mjml",
			"/absolute.mjml",
			"",
		} {
			t.Run(includePath, func(t *testing.T) {
				t.Parallel()

				fsys := includeTestFS(map[string]string{
					"partials/partial.mjml": `<mj-text>Partial</mj-text>`,
				})

				_, err := parseWithIncludes(t, fsys, "main.mjml", fmt.Sprintf(document, includePath))
				require.ErrorIs(t, err, ErrIncludePath)
			})
		}
	})

	t.Run("missing-file", func(t *testing.T) {
		t.Parallel()

		_, err := parseWithIncludes(t, includeTestFS(nil), "main.mjml",
			fmt.Sprintf(document, "partials/absent.mjml"))

		require.ErrorIs(t, err, fs.ErrNotExist)
		require.Contains(t, err.Error(), "mj-include partials/absent.mjml")
	})

	t.Run("unknown-type", func(t *testing.T) {
		t.Parallel()

		fsys := includeTestFS(map[string]string{
			"partials/text.mjml": `<mj-text>Text</mj-text>`,
		})

		_, err := parseWithIncludes(t, fsys, "main.mjml",
			`<mjml><mj-body><mj-include path="partials/text.mjml" type="js" /></mj-body></mjml>`)

		require.ErrorIs(t, err, ErrIncludeType)
		require.Contains(t, err.Error(), `"js"`)
	})

	t.Run("include-with-content", func(t *testing.T) {
		t.Parallel()

		fsys := includeTestFS(map[string]string{
			"partials/text.mjml": `<mj-text>Text</mj-text>`,
		})

		_, err := parseWithIncludes(t, fsys, "main.mjml",
			`<mjml><mj-body><mj-include path="partials/text.mjml"><mj-text>Dropped</mj-text></mj-include></mj-body></mjml>`)

		require.ErrorIs(t, err, ErrParsingFailed)
	})
}
