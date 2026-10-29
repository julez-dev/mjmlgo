package mjmlgo

import (
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/julez-dev/mjmlgo/component"
	"github.com/julez-dev/mjmlgo/internal/mjmltest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		want, ok := reference[fixture]
		require.True(t, ok, "no reference output for %s", fixture)

		got, renderErr := renderFixture(t, fixture)

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

// fixtureRenderer renders fixtures that need no configuration. It holds no
// per-render state, so the tests can share one instance.
var fixtureRenderer = NewRenderer()

// rendererFor returns the renderer a fixture must be rendered with: the options
// its sidecar asks for, which is the same configuration the reference CLI is
// given for that fixture. A fixture without a sidecar uses the shared default
// renderer.
func rendererFor(t testing.TB, fixture string) *Renderer {
	t.Helper()

	config := mjmltest.Config(t, fixture)

	var opts []Option

	if config.KeepComments != nil {
		opts = append(opts, WithComments(*config.KeepComments))
	}

	if config.FontsSet {
		opts = append(opts, WithFonts(config.Fonts))
	}

	if config.IgnoreIncludes != nil {
		opts = append(opts, WithIgnoreIncludes(*config.IgnoreIncludes))
	}

	if config.PrinterSupport != nil {
		opts = append(opts, WithPrinterSupport(*config.PrinterSupport))
	}

	if len(opts) == 0 {
		return fixtureRenderer
	}

	return NewRenderer(opts...)
}

// renderFixture renders fixture the way a caller would: the fixture's directory
// is the root of the filesystem, so mj-include resolves relative includes inside
// the corpus. That is exactly what the reference CLI does with a file path.
func renderFixture(t testing.TB, fixture string) (string, error) {
	t.Helper()

	fsys := os.DirFS(filepath.Dir(fixture))

	var out strings.Builder

	err := rendererFor(t, fixture).RenderFS(context.Background(), &out, fsys, filepath.Base(fixture))

	return out.String(), err
}

// renderConfiguredSource renders fixture's source with the configuration its
// sidecar records, without reading anything else from disk. The benchmarks use
// it so a fixture that exists to exercise an option is measured with it.
func renderConfiguredSource(t testing.TB, fixture string, source []byte) (string, error) {
	t.Helper()

	name := filepath.Base(fixture)
	fsys := fstest.MapFS{name: &fstest.MapFile{Data: source}}

	var out strings.Builder

	err := rendererFor(t, fixture).RenderFS(context.Background(), &out, fsys, name)

	return out.String(), err
}

// renderPlain renders src through the package-level entry point, the way a
// caller with no filesystem and no options would.
func renderPlain(t testing.TB, src string) (string, error) {
	t.Helper()

	var out strings.Builder

	err := Render(&out, []byte(src))

	return out.String(), err
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

// invalidFailure is what TestMJMLInvalid expects from a fixture in addition to
// the recorded message: the structured kind and, where it is stable across
// refactors, the element path and the offending attribute and value.
type invalidFailure struct {
	kind      ErrorKind
	path      []string
	attribute string
	value     string
	// sentinel is the error errors.Is must still find through the structured
	// error, so wrapping did not break the older inspection style.
	sentinel error
}

// invalidFailures records the structured failure every invalid fixture must
// produce. A fixture without an entry fails the test, so adding an invalid
// fixture means stating what it rejects, not just what it prints.
var invalidFailures = map[string]invalidFailure{
	"auto-width.mjml": {
		kind:      KindInvalidValue,
		path:      []string{"mj-body", "mj-section", "mj-column", "mj-image"},
		attribute: "width",
		value:     "auto",
		sentinel:  ErrValidation,
	},
	"disallowed-child.mjml": {
		kind:     KindDisallowedChild,
		path:     []string{"mj-body", "mj-text"},
		value:    "mj-text",
		sentinel: ErrValidation,
	},
	"disallowed-root-child.mjml": {
		kind:     KindDisallowedChild,
		path:     []string{"mj-text"},
		value:    "mj-text",
		sentinel: ErrValidation,
	},
	"include-cycle.mjml": {
		kind:     KindParseFailure,
		path:     []string{"mj-body", "mj-include"},
		value:    "partials/cycle-a.mjml",
		sentinel: ErrIncludeCycle,
	},
	"include-escape.mjml": {
		kind:      KindParseFailure,
		path:      []string{"mj-body", "mj-include"},
		attribute: "path",
		value:     "../outside-the-corpus.mjml",
		sentinel:  ErrIncludePath,
	},
	"include-missing-file.mjml": {
		kind:     KindParseFailure,
		path:     []string{"mj-body", "mj-include"},
		value:    "partials/does-not-exist.mjml",
		sentinel: fs.ErrNotExist,
	},
	"include-malformed.mjml": {
		kind:     KindParseFailure,
		path:     []string{"mj-body", "mj-include"},
		value:    "partials/malformed.mjml",
		sentinel: ErrParsingFailed,
	},
	"include-unknown-type.mjml": {
		kind:     KindUnknownIncludeType,
		path:     []string{"mj-body", "mj-include"},
		value:    "js",
		sentinel: ErrIncludeType,
	},
	"invalid-color.mjml": {
		kind:      KindInvalidValue,
		path:      []string{"mj-body", "mj-section", "mj-column", "mj-text"},
		attribute: "color",
		value:     "chartreuseish",
		sentinel:  ErrValidation,
	},
	"malformed-xml.mjml": {
		kind:     KindParseFailure,
		path:     []string{"mjml"},
		sentinel: ErrParsingFailed,
	},
	"mismatched-tags.mjml": {
		kind:     KindParseFailure,
		path:     []string{"mj-body", "mj-section", "mj-column"},
		sentinel: ErrParsingFailed,
	},
	"missing-body.mjml": {
		kind:     KindParseFailure,
		path:     []string{"mjml"},
		sentinel: component.ErrMJMLBadlyFormatted,
	},
	"negative-font-size.mjml": {
		kind:      KindInvalidValue,
		path:      []string{"mj-body", "mj-section", "mj-column", "mj-text"},
		attribute: "font-size",
		value:     "-12px",
		sentinel:  ErrValidation,
	},
	"no-root.mjml": {
		kind:     KindParseFailure,
		path:     []string{"mjml"},
		sentinel: ErrParsingFailed,
	},
	"undeclared-mj-class.mjml": {
		kind:      KindUndefinedMJClass,
		path:      []string{"mj-body", "mj-section", "mj-column", "mj-text"},
		attribute: "mj-class",
		value:     "missing",
		sentinel:  ErrValidation,
	},
	"unknown-attribute.mjml": {
		kind:      KindUnknownAttribute,
		path:      []string{"mj-body", "mj-section", "mj-column", "mj-text"},
		attribute: "made-up",
		value:     "value",
		sentinel:  ErrValidation,
	},
	"unknown-body-tag.mjml": {
		kind:     KindUnknownElement,
		path:     []string{"mj-body", "mj-foo"},
		value:    "mj-foo",
		sentinel: ErrValidation,
	},
	"unknown-child.mjml": {
		kind:     KindUnknownElement,
		path:     []string{"mj-body", "mj-section", "mj-column", "mj-foo"},
		value:    "mj-foo",
		sentinel: ErrValidation,
	},
	"unknown-head-tag.mjml": {
		kind:     KindUnknownElement,
		path:     []string{"mj-head", "mj-foo"},
		value:    "mj-foo",
		sentinel: ErrValidation,
	},
	"unknown-root.mjml": {
		kind:     KindUnknownElement,
		path:     []string{"mj-div"},
		value:    "mj-div",
		sentinel: ErrUnknownStartingTag,
	},
}

// TestMJMLInvalid asserts that malformed or invalid documents are rejected with
// a clear error. This implementation validates hard (fail fast), while MJML
// itself only warns and keeps rendering, so these fixtures are deliberately not
// part of the parity corpus.
func TestMJMLInvalid(t *testing.T) {
	t.Parallel()

	fixtures := mjmlFixtures(t, invalidDir)
	require.NotEmpty(t, fixtures, "no .mjml fixtures found in %s", invalidDir)

	// Every recorded expectation must still have a fixture, so a deleted
	// fixture cannot leave a stale entry behind.
	for name := range invalidFailures {
		require.FileExists(t, filepath.Join(invalidDir, name), "invalidFailures entry %s has no fixture", name)
	}

	for _, fixture := range fixtures {
		t.Run(filepath.ToSlash(fixture), func(t *testing.T) {
			t.Parallel()

			html, err := renderFixture(t, fixture)

			expectedPath := fixture + ".error.txt"
			expected, readErr := os.ReadFile(expectedPath)
			require.NoError(t, readErr, "fixture %s needs a %s with the expected error", fixture, filepath.Base(expectedPath))

			require.Error(t, err, "expected %s to be rejected, rendered:\n%s", fixture, html)
			assert.Contains(t, err.Error(), strings.TrimSpace(string(expected)))

			want, ok := invalidFailures[filepath.Base(fixture)]
			require.True(t, ok, "fixture %s has no entry in invalidFailures", filepath.Base(fixture))

			var structured *Error
			require.ErrorAs(t, err, &structured, "fixture %s did not return a structured *Error", fixture)
			assert.Equal(t, want.kind, structured.Kind, "fixture %s", fixture)
			require.NotEmpty(t, structured.Path, "fixture %s returned an empty element path", fixture)

			if want.path != nil {
				assert.Equal(t, want.path, structured.Path, "fixture %s", fixture)
			}

			assert.Equal(t, want.attribute, structured.Attribute, "fixture %s", fixture)
			assert.Equal(t, want.value, structured.Value, "fixture %s", fixture)

			if want.sentinel != nil {
				assert.ErrorIs(t, err, want.sentinel, "fixture %s lost its sentinel", fixture)
			}
		})
	}
}

// TestRenderWithoutInlineStylesKeepsEmittedMarkup asserts a document with neither
// <mj-style inline> nor <mj-html-attributes> is not run through the inline stage at
// all: its markup is emitted as-is, in the emitted formatting, so nothing is
// re-serialized for a document that has nothing to inline.
func TestRenderWithoutInlineStylesKeepsEmittedMarkup(t *testing.T) {
	t.Parallel()

	for _, fixture := range []string{"simple.mjml", "layout.mjml", "social.mjml"} {
		input, err := os.ReadFile(filepath.Join(parityDir, fixture))
		require.NoError(t, err)

		rendered, err := renderPlain(t, string(input))
		require.NoError(t, err)

		// The document the components emitted starts with the template's own
		// doctype and writes its void elements as the components did; a
		// re-serialization would uppercase the doctype and close them with "/>".
		assert.True(t, strings.HasPrefix(rendered, "<!doctype html>\n<html "),
			"%s: the emitted doctype was rewritten:\n%s", fixture, rendered[:min(len(rendered), 120)])
		assert.Contains(t, rendered, `<meta http-equiv="X-UA-Compatible" content="IE=edge">`,
			"%s: a void element was re-serialized", fixture)
	}
}

// TestRenderInlineStylesIsDeterministic renders every fixture that inlines CSS
// several times and asserts the bytes are identical. The style attribute order
// used to come from Go map iteration, so repeated renders of the same document
// could differ.
func TestRenderInlineStylesIsDeterministic(t *testing.T) {
	t.Parallel()

	checked := 0

	for _, fixture := range mjmlFixtures(t, parityDir) {
		input, err := os.ReadFile(fixture)
		require.NoError(t, err)

		if !strings.Contains(string(input), `inline="inline"`) {
			continue
		}

		checked++

		first, err := renderFixture(t, fixture)
		require.NoError(t, err)

		for range 4 {
			again, err := renderFixture(t, fixture)
			require.NoError(t, err)
			require.Equal(t, first, again, "%s rendered differently on a repeated render", fixture)
		}
	}

	require.NotZero(t, checked, "no fixture in %s inlines CSS", parityDir)
}

// TestRenderMergesOutlookConditionals asserts the rendered document applies
// MJML's mergeOutlookConditionnals step: adjacent Outlook conditional blocks are
// joined so the MSO markup is not fragmented across comments.
func TestRenderMergesOutlookConditionals(t *testing.T) {
	t.Parallel()

	input, err := os.ReadFile(filepath.Join(parityDir, "layout.mjml"))
	require.NoError(t, err)

	html, err := renderPlain(t, string(input))
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

// includeTargetDir holds files that are only read through mj-include. They are
// not fixtures of their own; see testdata/README.md.
const includeTargetDir = "partials"

// mjmlFixtures returns all .mjml files below dir, sorted for stable output.
// Include targets below a "partials" directory are skipped: they are fragments or
// full documents meant to be included, not rendered on their own.
func mjmlFixtures(t testing.TB, dir string) []string {
	t.Helper()

	var fixtures []string

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if path != dir && (strings.HasPrefix(d.Name(), ".") || d.Name() == includeTargetDir) {
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

// TestRenderRejectsIncludes asserts that a document with mj-include fails loudly
// instead of dropping the included content when it is rendered without a
// filesystem.
func TestRenderRejectsIncludes(t *testing.T) {
	t.Parallel()

	const input = `<mjml><mj-body><mj-include path="partials/header.mjml" /></mj-body></mjml>`

	_, err := renderPlain(t, input)
	require.ErrorIs(t, err, ErrIncludeNoFS)

	// A document without includes renders through the same entry point.
	_, err = renderPlain(t, `<mjml><mj-body><mj-raw>raw</mj-raw></mj-body></mjml>`)
	require.NoError(t, err)
}

func TestRenderFileStartRaw(t *testing.T) {
	t.Parallel()

	const input = `<mjml>
		<mj-raw position="file-start"><!-- file-start marker --></mj-raw>
		<mj-body><mj-section><mj-column><mj-text>Body</mj-text></mj-column></mj-section></mj-body>
	</mjml>`

	html, err := renderPlain(t, input)
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

	out, err := renderPlain(t, input)
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

// BenchmarkRenderMJML measures the full pipeline (parse, render, inline CSS,
// output post-processing) per fixture. The fixture set is the parity corpus, so
// the numbers describe realistic documents rather than toy inputs.
func BenchmarkRenderMJML(b *testing.B) {
	forEachParityFixture(b, func(name, fixture string) {
		b.Run(name, func(b *testing.B) {
			input, err := os.ReadFile(fixture)
			if err != nil {
				b.Fatal(err)
			}

			render := func() error {
				return Render(io.Discard, input)
			}

			// A fixture whose sidecar configures the renderer - keepComments,
			// fonts, ignoreIncludes, printerSupport - must be measured with that
			// configuration, or the row describes a pipeline the fixture does
			// not stand for.
			if len(mjmltest.Args(b, fixture)) > 0 {
				render = func() error {
					_, err := renderConfiguredSource(b, fixture, input)

					return err
				}
			}

			if err := render(); err != nil {
				// Fixtures with mj-include cannot be rendered without a
				// filesystem. Measure the caller-visible path instead, which
				// includes reading the fixture and its includes from disk.
				render = func() error {
					_, err := renderFixture(b, fixture)

					return err
				}
			}

			if err := render(); err != nil {
				b.Skipf("fixture does not render: %v", err)
			}

			b.ReportAllocs()
			b.SetBytes(int64(len(input)))

			for b.Loop() {
				if err := render(); err != nil {
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
			html, err := renderFixture(b, fixture)
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

		html, err := renderPlain(t, source)
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

// TestRenderStripsRawBlockMarkers asserts the markers a table-body container uses
// to state where its raw children sit never reach the output, including on the
// fast path that skips the CSS inlining pass.
func TestRenderStripsRawBlockMarkers(t *testing.T) {
	t.Parallel()

	for _, fixture := range []string{"raw-in-column.mjml", "raw.mjml", "raw-in-section.mjml", "simple.mjml"} {
		input, err := os.ReadFile(filepath.Join(parityDir, fixture))
		require.NoError(t, err)

		rendered, err := renderPlain(t, string(input))
		require.NoError(t, err)

		assert.NotContains(t, rendered, "mjmlgo-raw", "%s leaked an internal marker", fixture)
	}
}
