package mjmlgo

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/julez-dev/mjmlgo/node"
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
