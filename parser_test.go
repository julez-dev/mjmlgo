package mjmlgo

import (
	"bytes"
	"os"
	"strings"
	"testing"

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
