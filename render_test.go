package mjmlgo

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/julez-dev/mjmlgo/internal/component"
	"github.com/julez-dev/mjmlgo/internal/mjmltest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

// parityDir holds the fixtures that must render identically to MJML.
const parityDir = "testdata/parity"

// invalidDir holds fixtures that must be rejected with the error recorded in
// the sibling .error.txt file.
const invalidDir = "testdata/invalid"

// TestMJMLParity renders every fixture in testdata with both this package and
// the reference JavaScript MJML implementation and compares the resulting HTML
// documents structurally.
//
// The reference version is pinned in package.json; see testdata/README.md.
// There is no baseline of accepted differences: every fixture must match.
func TestMJMLParity(t *testing.T) {
	// The reference implementation is the real mjml CLI, so this needs Node.js.
	if testing.Short() {
		t.Skip("skipping parity test against the reference MJML implementation in short mode")
	}

	fixtures := mjmlFixtures(t, parityDir)
	require.NotEmpty(t, fixtures, "no .mjml fixtures found in %s", parityDir)

	cli := mjmltest.NewCLI(t)
	t.Logf("reference implementation: MJML %s (%s)", cli.Version, cli.Source)

	reference := cli.RenderAll(t, fixtures)

	for _, fixture := range fixtures {
		input, err := os.ReadFile(fixture)
		require.NoError(t, err)

		want, ok := reference[fixture]
		require.True(t, ok, "no reference output for %s", fixture)

		got, renderErr := RenderMJML(bytes.NewReader(input))

		t.Run(filepath.ToSlash(fixture), func(t *testing.T) {
			if renderErr != nil {
				t.Fatalf("%s: rendering failed, but MJML renders it: %v", fixture, renderErr)
			}

			mismatches, err := mjmltest.CompareHTML(want, got)
			require.NoError(t, err)

			if len(mismatches) == 0 {
				return
			}

			writeFailureArtifacts(t, fixture, want, got)

			t.Fatalf(
				"%s differs from MJML %s in %d place(s):\n%s",
				fixture, cli.Version, len(mismatches), formatMismatches(mismatches),
			)
		})
	}
}

func formatMismatches(mismatches []mjmltest.Mismatch) string {
	var b strings.Builder

	for _, mismatch := range mismatches {
		b.WriteString("  - ")
		b.WriteString(mismatch.String())
		b.WriteString("\n")
	}

	return b.String()
}

// writeFailureArtifacts stores the reference and generated HTML plus their
// normalized forms so failures can be inspected with a normal diff tool.
func writeFailureArtifacts(t *testing.T, fixture, want, got string) {
	t.Helper()

	dir := filepath.Join("testdata", ".failures")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Logf("could not create failure artifact directory: %v", err)
		return
	}

	name := strings.TrimSuffix(filepath.Base(fixture), filepath.Ext(fixture))

	artifacts := map[string]string{
		name + ".want.html":          want,
		name + ".got.html":           got,
		name + ".want.canonical.txt": mjmltest.CanonicalMust(want),
		name + ".got.canonical.txt":  mjmltest.CanonicalMust(got),
	}

	for file, content := range artifacts {
		if err := os.WriteFile(filepath.Join(dir, file), []byte(content), 0o644); err != nil {
			t.Logf("could not write failure artifact %s: %v", file, err)
			continue
		}
	}

	t.Logf("wrote failure artifacts to %s", dir)
}

// TestMJMLInvalid asserts that malformed or invalid documents are rejected with
// a clear error. This implementation validates hard (fail fast), while MJML
// itself only warns and keeps rendering, so these fixtures are deliberately not
// part of the parity corpus.
func TestMJMLInvalid(t *testing.T) {
	t.Parallel()

	fixtures := mjmlFixtures(t, invalidDir)
	require.NotEmpty(t, fixtures, "no .mjml fixtures found in %s", invalidDir)

	for _, fixture := range fixtures {
		t.Run(filepath.ToSlash(fixture), func(t *testing.T) {
			t.Parallel()

			input, err := os.ReadFile(fixture)
			require.NoError(t, err)

			html, err := RenderMJML(bytes.NewReader(input))

			expectedPath := fixture + ".error.txt"
			expected, readErr := os.ReadFile(expectedPath)
			require.NoError(t, readErr, "fixture %s needs a %s with the expected error", fixture, filepath.Base(expectedPath))

			require.Error(t, err, "expected %s to be rejected, rendered:\n%s", fixture, html)
			assert.Contains(t, err.Error(), strings.TrimSpace(string(expected)))
		})
	}
}

// TestRenderWithoutInlineStylesKeepsMarkup asserts the fast path: a document
// with neither <mj-style inline> nor <mj-html-attributes> is not round-tripped
// through the HTML parser, but still normalizes to the same tree as the inlined
// variant would.
func TestRenderWithoutInlineStylesKeepsMarkup(t *testing.T) {
	t.Parallel()

	for _, fixture := range []string{"simple.mjml", "layout.mjml", "social.mjml"} {
		input, err := os.ReadFile(filepath.Join(parityDir, fixture))
		require.NoError(t, err)

		rendered, err := RenderMJML(bytes.NewReader(input))
		require.NoError(t, err)

		var roundTripped strings.Builder
		require.NoError(t, inlineCSS(&component.RenderContext{}, strings.NewReader(rendered), &roundTripped))

		want, err := mjmltest.ParseHTML(roundTripped.String())
		require.NoError(t, err)

		got, err := mjmltest.ParseHTML(rendered)
		require.NoError(t, err)

		assert.Empty(t, mjmltest.Compare(want, got), "%s: fast path changed the normalized document", fixture)
	}
}

// TestRenderStripsRawBlockMarkers asserts the markers used internally to protect
// raw column content never reach the output, including on the fast path that
// skips the CSS inlining pass.
func TestRenderStripsRawBlockMarkers(t *testing.T) {
	t.Parallel()

	for _, fixture := range []string{"raw-in-column.mjml", "raw.mjml", "raw-in-section.mjml"} {
		input, err := os.ReadFile(filepath.Join(parityDir, fixture))
		require.NoError(t, err)

		html, err := RenderMJML(bytes.NewReader(input))
		require.NoError(t, err)

		assert.NotContains(t, html, "mjmlgo-raw", "%s leaked an internal marker", fixture)
	}
}

// TestRenderMergesOutlookConditionals asserts the rendered document applies
// MJML's mergeOutlookConditionnals step: adjacent Outlook conditional blocks are
// joined so the MSO markup is not fragmented across comments.
func TestRenderMergesOutlookConditionals(t *testing.T) {
	t.Parallel()

	input, err := os.ReadFile(filepath.Join(parityDir, "layout.mjml"))
	require.NoError(t, err)

	html, err := RenderMJML(bytes.NewReader(input))
	require.NoError(t, err)

	assert.NotContains(t, html, "<![endif]--><!--[if mso | IE]>")
	assert.NotContains(t, html, "<![endif]--><![endif]-->")

	// Nested MSO tables must still be closed: merging must not swallow markers.
	assert.Contains(t, html, "<!--[if mso | IE]></td></tr></table><![endif]-->")
}

// forEachParityFixture runs fn for every parity fixture, named after its path
// relative to the parity directory.
func forEachParityFixture(tb testing.TB, fn func(name, fixture string)) {
	for _, fixture := range mjmlFixtures(tb, parityDir) {
		fn(strings.TrimPrefix(filepath.ToSlash(fixture), parityDir+"/"), fixture)
	}
}

// maxFuzzInput bounds fuzz input size so one case cannot dominate the run.
const maxFuzzInput = 1 << 20

// checkFuzzInput skips oversized fuzz inputs.
func checkFuzzInput(t *testing.T, source string) {
	t.Helper()

	if len(source) > maxFuzzInput {
		t.Skip("input too large for a fuzz case")
	}
}

// mjmlFixtures returns all .mjml files below dir, sorted for stable output.
func mjmlFixtures(t testing.TB, dir string) []string {
	t.Helper()

	var fixtures []string

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if path != dir && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}

			return nil
		}

		if filepath.Ext(path) == ".mjml" {
			fixtures = append(fixtures, path)
		}

		return nil
	})
	require.NoError(t, err)

	slices.Sort(fixtures)

	return fixtures
}

func TestRenderFileStartRaw(t *testing.T) {
	t.Parallel()

	const input = `<mjml>
		<mj-raw position="file-start"><!-- file-start marker --></mj-raw>
		<mj-body><mj-section><mj-column><mj-text>Body</mj-text></mj-column></mj-section></mj-body>
	</mjml>`

	html, err := RenderMJML(strings.NewReader(input))
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(strings.ToLower(html), "<!-- file-start marker -->\n<!doctype html>"), "file-start raw must precede the doctype, got:\n%s", html)
}

func TestRenderRawInColumnKeepsPosition(t *testing.T) {
	t.Parallel()

	const input = `<mjml>
		<mj-body>
			<mj-section>
				<mj-column>
					<mj-text>first text</mj-text>
					<mj-raw><div class="raw-between">raw markup</div></mj-raw>
					<mj-text>second text</mj-text>
				</mj-column>
			</mj-section>
		</mj-body>
	</mjml>`

	out, err := RenderMJML(strings.NewReader(input))
	require.NoError(t, err)

	tbody := strings.Index(out, "<tbody>")
	require.NotEqual(t, -1, tbody, "missing <tbody>")

	firstTR := strings.Index(out[tbody:], "<tr>")
	require.NotEqual(t, -1, firstTR, "missing first row")
	firstTR += tbody

	raw := strings.Index(out, `<div class="raw-between">raw markup</div>`)
	require.NotEqual(t, -1, raw, "missing raw markup")

	secondTR := strings.Index(out[raw:], "<tr>")
	require.NotEqual(t, -1, secondTR, "missing second row")
	secondTR += raw

	require.Less(t, tbody, firstTR)
	require.Less(t, firstTR, raw, "raw markup must follow the first row")
	require.Less(t, raw, secondTR, "raw markup must precede the second row")
}

func TestInlineCSS(t *testing.T) {
	t.Parallel()

	t.Run("simple", func(t *testing.T) {
		const input = `<html><body><p class="p-class">Hello</p></body></html>`
		ctx := &component.RenderContext{
			InlineStyles: []component.Stylesheet{
				{
					Rules: []component.Rule{
						{
							Selectors: ".p-class",
							Declarations: []component.Style{
								{Property: "font-size", Value: "22px", Important: true},
							}},
					},
				},
			},
		}

		var out bytes.Buffer
		err := inlineCSS(ctx, strings.NewReader(input), &out)
		require.NoError(t, err)

		require.Equal(t, "<html><head></head><body><p class=\"p-class\" style=\"font-size:22px;\">Hello</p></body></html>", out.String())
	})

	t.Run("important-override", func(t *testing.T) {
		const input = `<html><body><p class="p-class" style="font-size: 20px;font-weight:300">Hello</p></body></html>`
		ctx := &component.RenderContext{
			InlineStyles: []component.Stylesheet{
				{
					Rules: []component.Rule{
						{
							Selectors: ".p-class",
							Declarations: []component.Style{
								{Property: "font-size", Value: "22px", Important: true},
								{Property: "font-weight", Value: "400"},
							}},
					},
				},
			},
		}

		var out bytes.Buffer
		err := inlineCSS(ctx, strings.NewReader(input), &out)
		require.NoError(t, err)

		randomOrder := []string{
			"<html><head></head><body><p class=\"p-class\" style=\"font-size:22px;font-weight:300;\">Hello</p></body></html>",
			"<html><head></head><body><p class=\"p-class\" style=\"font-weight:300;font-size:22px;\">Hello</p></body></html>",
		}

		assert.True(t, slices.Contains(randomOrder, out.String()), "returned value should be one of the possible")
	})
}

// BenchmarkRenderMJML measures the full pipeline (parse, render, inline CSS,
// output post-processing) per fixture. The fixture set is the parity corpus, so
// the numbers describe realistic documents rather than toy inputs.
//
// Budgets live in benchmarks/BUDGETS.md.
func BenchmarkRenderMJML(b *testing.B) {
	forEachParityFixture(b, func(name, fixture string) {
		b.Run(name, func(b *testing.B) {
			input, err := os.ReadFile(fixture)
			if err != nil {
				b.Fatal(err)
			}

			if _, err := RenderMJML(bytes.NewReader(input)); err != nil {
				b.Skipf("fixture does not render: %v", err)
			}

			b.ReportAllocs()
			b.SetBytes(int64(len(input)))

			for b.Loop() {
				if _, err := RenderMJML(bytes.NewReader(input)); err != nil {
					b.Fatal(err)
				}
			}
		})
	})
}

// BenchmarkParityComparison isolates the test harness cost (normalize both
// documents and compare them), which gates how large the parity corpus can get.
func BenchmarkParityComparison(b *testing.B) {
	forEachParityFixture(b, func(name, fixture string) {
		b.Run(name, func(b *testing.B) {
			input, err := os.ReadFile(fixture)
			if err != nil {
				b.Fatal(err)
			}

			html, err := RenderMJML(bytes.NewReader(input))
			if err != nil {
				b.Skipf("fixture does not render: %v", err)
			}

			b.ReportAllocs()

			for b.Loop() {
				if _, err := mjmltest.CompareHTML(html, html); err != nil {
					b.Fatal(err)
				}
			}
		})
	})
}

// FuzzRenderMJML asserts that the renderer never panics on arbitrary input and
// that anything it accepts produces a well-formed HTML document. Malformed input
// is expected to return an error (this implementation fails fast).
func FuzzRenderMJML(f *testing.F) {
	f.Add(`<mjml><mj-body><mj-section><mj-column><mj-text>Hi</mj-text></mj-column></mj-section></mj-body></mjml>`)
	f.Add(`<mjml><mj-body></mj-body></mjml>`)
	f.Add(`<mjml lang="de" dir="rtl"><mj-body><mj-section><mj-column><mj-text font-size="12px">x</mj-text></mj-column></mj-section></mj-body></mjml>`)
	f.Add(`<mjml><mj-head><mj-attributes><mj-all font-family="Arial" /></mj-attributes></mj-head><mj-body /></mjml>`)
	f.Add(`<mjml><mj-body><mj-section><mj-column><mj-raw><div>x</div></mj-raw></mj-column></mj-section></mj-body></mjml>`)
	f.Add(`<mjml><mj-body><mj-section><mj-column><mj-text`)
	f.Add(``)

	for _, fixture := range []string{"testdata/parity/simple.mjml", "testdata/parity/content.mjml", "testdata/parity/head.mjml"} {
		input, err := os.ReadFile(fixture)
		if err != nil {
			f.Fatal(err)
		}

		f.Add(string(input))
	}

	f.Fuzz(func(t *testing.T, source string) {
		checkFuzzInput(t, source)

		html, err := RenderMJML(strings.NewReader(source))
		if err != nil {
			return
		}

		if html == "" {
			t.Fatalf("rendered empty output for input %q", source)
		}

		if !strings.Contains(strings.ToLower(html), "<!doctype html>") {
			t.Fatalf("output is not an HTML document: %q", html)
		}
	})
}

func TestApplyHTMLAttributes(t *testing.T) {
	t.Parallel()

	const document = `<html><head><title class="text">head</title></head><body>` +
		`<div class="text"><div>first</div></div>` +
		`<div class="text"><div>second</div></div>` +
		`</body></html>`

	doc, err := html.Parse(strings.NewReader(document))
	require.NoError(t, err)

	applyHTMLAttributes([]component.HTMLAttributeSelector{
		{
			Path:       ".text div",
			Attributes: []component.HTMLAttribute{{Name: "data-id", Value: "42"}},
		},
		{
			Path:       ".missing",
			Attributes: []component.HTMLAttribute{{Name: "data-missing", Value: "never"}},
		},
		{
			Path:       ".text div",
			Attributes: []component.HTMLAttribute{{Name: "data-extra", Value: "second"}},
		},
	}, doc)

	var out bytes.Buffer
	require.NoError(t, html.Render(&out, doc))
	rendered := out.String()

	// Every match of the repeated selector gets both attributes.
	require.Equal(t, 2, strings.Count(rendered, `data-id="42"`))
	require.Equal(t, 2, strings.Count(rendered, `data-extra="second"`))

	// A selector matching nothing is a no-op.
	require.NotContains(t, rendered, "data-missing")

	// The post-render pass only targets rendered body content, like MJML.
	require.Contains(t, rendered, `<title class="text">head</title>`)
}
