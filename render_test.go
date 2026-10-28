package mjmlgo

import (
	"bytes"
	"context"
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

	return rendererFor(t, fixture).RenderString(context.Background(), fsys, filepath.Base(fixture))
}

// renderConfiguredSource renders fixture's source with the configuration its
// sidecar records, without reading anything else from disk. The benchmarks use
// it so a fixture that exists to exercise an option is measured with it.
func renderConfiguredSource(t testing.TB, fixture string, source []byte) (string, error) {
	t.Helper()

	name := filepath.Base(fixture)
	fsys := fstest.MapFS{name: &fstest.MapFile{Data: source}}

	return rendererFor(t, fixture).RenderString(context.Background(), fsys, name)
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

// TestRenderWithoutInlineStylesKeepsMarkup asserts the fast path: a document
// with neither <mj-style inline> nor <mj-html-attributes> is not round-tripped
// through the HTML parser, but still normalizes to the same tree as the version
// the round trip would produce.
func TestRenderWithoutInlineStylesKeepsMarkup(t *testing.T) {
	t.Parallel()

	for _, fixture := range []string{"simple.mjml", "layout.mjml", "social.mjml"} {
		input, err := os.ReadFile(filepath.Join(parityDir, fixture))
		require.NoError(t, err)

		rendered, err := RenderMJML(bytes.NewReader(input))
		require.NoError(t, err)

		// The inliner skips documents it does not have to touch, so the round
		// trip is written out here: this test is what keeps the skipped
		// parse/render from hiding a formatting difference.
		parsed, err := html.Parse(strings.NewReader(rendered))
		require.NoError(t, err)

		var roundTripped strings.Builder
		require.NoError(t, html.Render(&roundTripped, parsed))

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

// TestRenderMJMLRejectsIncludes asserts that the deprecated entry point, which
// has no filesystem of its own, fails loudly instead of dropping included content
// on the floor.
func TestRenderMJMLRejectsIncludes(t *testing.T) {
	t.Parallel()

	const input = `<mjml><mj-body><mj-include path="partials/header.mjml" /></mj-body></mjml>`

	_, err := RenderMJML(strings.NewReader(input))
	require.ErrorIs(t, err, ErrIncludeNoFS)

	// A document without includes renders through the same entry point as before.
	_, err = RenderMJML(strings.NewReader(`<mjml><mj-body><mj-raw>raw</mj-raw></mj-body></mjml>`))
	require.NoError(t, err)
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

	t.Run("adds declarations to matching nodes", func(t *testing.T) {
		t.Parallel()

		const input = `<html><body><p class="p-class">Hello</p></body></html>`

		var out bytes.Buffer
		require.NoError(t, inlineCSS(inlineContext(inlineSheet(".p-class",
			component.Style{Property: "font-size", Value: "22px", Important: true})),
			strings.NewReader(input), &out))

		require.Equal(t, `<html><head></head><body><p class="p-class" style="font-size:22px;">Hello</p></body></html>`, out.String())
	})

	t.Run("important overrides an existing declaration", func(t *testing.T) {
		t.Parallel()

		const input = `<html><body><p class="p-class" style="font-size: 20px;font-weight:300">Hello</p></body></html>`

		var out bytes.Buffer
		require.NoError(t, inlineCSS(inlineContext(inlineSheet(".p-class",
			component.Style{Property: "font-size", Value: "22px", Important: true},
			component.Style{Property: "font-weight", Value: "400"})),
			strings.NewReader(input), &out))

		// The existing declaration order is preserved, the important rule wins
		// and the plain rule loses against the node's own declaration.
		require.Equal(t, `<html><head></head><body><p class="p-class" style="font-size:22px;font-weight:300;">Hello</p></body></html>`, out.String())
	})

	t.Run("existing declaration order is preserved", func(t *testing.T) {
		t.Parallel()

		const input = `<html><body><p class="p-class" style="z-index:1;color:red">Hello</p></body></html>`

		var out bytes.Buffer
		require.NoError(t, inlineCSS(inlineContext(inlineSheet(".p-class",
			component.Style{Property: "border", Value: "0"},
			component.Style{Property: "color", Value: "green"})),
			strings.NewReader(input), &out))

		require.Equal(t, `<html><head></head><body><p class="p-class" style="z-index:1;color:red;border:0;">Hello</p></body></html>`, out.String())
	})

	t.Run("unmatched nodes are untouched", func(t *testing.T) {
		t.Parallel()

		// The markup is deliberately not canonical HTML (unquoted attribute
		// values, a bare <br>): a node no rule touches must not be reserialized.
		const input = `<html><body><p style=color:red class=other>a<br></p><div class=no-style>b</div></body></html>`

		var out bytes.Buffer
		require.NoError(t, inlineCSS(inlineContext(inlineSheet(".p-class",
			component.Style{Property: "font-size", Value: "22px"})),
			strings.NewReader(input), &out))

		require.Equal(t, input, out.String())
		require.NotContains(t, out.String(), "font-size")
	})

	t.Run("unparsable selectors are skipped", func(t *testing.T) {
		t.Parallel()

		const input = `<html><body><p class="p-class">Hello</p></body></html>`

		ctx := inlineContext(
			inlineSheet("!! not a selector", component.Style{Property: "color", Value: "red"}),
			inlineSheet(".p-class", component.Style{Property: "font-weight", Value: "bold"}),
		)

		var out bytes.Buffer
		require.NoError(t, inlineCSS(ctx, strings.NewReader(input), &out))

		require.Equal(t, `<html><head></head><body><p class="p-class" style="font-weight:bold;">Hello</p></body></html>`, out.String())
	})

	t.Run("layout declarations mirror to legacy attributes", func(t *testing.T) {
		t.Parallel()

		const input = `<html><body><table class="t"><tr><td class="c">x</td></tr></table><div class="d">y</div></body></html>`

		ctx := inlineContext(
			inlineSheet(".c",
				component.Style{Property: "width", Value: "50%"},
				component.Style{Property: "text-align", Value: "right"},
				component.Style{Property: "vertical-align", Value: "bottom"},
				component.Style{Property: "background-color", Value: "#000"}),
			inlineSheet(".d", component.Style{Property: "width", Value: "100px"}),
		)

		var out bytes.Buffer
		require.NoError(t, inlineCSS(ctx, strings.NewReader(input), &out))

		rendered := out.String()

		// px widths become numeric attributes, percentages stay as they are.
		require.Contains(t, rendered, `<td class="c" width="50%" align="right" valign="bottom" bgcolor="#000" style="width:50%;text-align:right;vertical-align:bottom;background-color:#000;">`)
		require.Contains(t, rendered, `<div class="d" width="100" style="width:100px;">`)
	})
}

// TestInlineCSSDeterminism asserts a render is byte-for-byte reproducible: the
// style attribute order used to come from Go map iteration, so two renders of
// the same document could differ.
func TestInlineCSSDeterminism(t *testing.T) {
	t.Parallel()

	input := `<html><body>` +
		`<p class="a b c" style="color:red;font-size:12px">x</p>` +
		`<div class="a" style="padding:0;margin:0">y</div>` +
		`</body></html>`

	ctx := inlineContext(inlineSheet(".a, .b, .c",
		component.Style{Property: "background-color", Value: "#fff"},
		component.Style{Property: "border", Value: "0"},
		component.Style{Property: "text-align", Value: "left"},
		component.Style{Property: "vertical-align", Value: "top"},
		component.Style{Property: "width", Value: "100px"},
		component.Style{Property: "font-weight", Value: "bold"},
	))

	var first bytes.Buffer
	require.NoError(t, inlineCSS(ctx, strings.NewReader(input), &first))

	for range 20 {
		var again bytes.Buffer
		require.NoError(t, inlineCSS(ctx, strings.NewReader(input), &again))
		require.Equal(t, first.String(), again.String())
	}
}

// inlineContext builds a render context with the given inline stylesheets.
func inlineContext(sheets ...component.Stylesheet) *component.RenderContext {
	return &component.RenderContext{InlineStyles: sheets}
}

// inlineSheet builds a stylesheet with one rule for selector and declarations.
// The selector is compiled the way the head parser compiles it.
func inlineSheet(selector string, declarations ...component.Style) component.Stylesheet {
	return component.Stylesheet{Rules: []component.Rule{newInlineRule(selector, declarations...)}}
}

func newInlineRule(selector string, declarations ...component.Style) component.Rule {
	rule := component.Rule{Selectors: selector, Declarations: declarations}
	rule.CompileSelectors()

	return rule
}

// TestStyleDecls locks down the inline style merge: declaration order is
// preserved, new properties are appended in application order, and only an
// important declaration overrides an existing property.
func TestStyleDecls(t *testing.T) {
	t.Parallel()

	t.Run("appends new properties after the existing ones", func(t *testing.T) {
		t.Parallel()

		var decls styleDecls
		require.NoError(t, parseStyleAttribute(&decls, "z-index: 1; color: red"))

		require.True(t, decls.set(component.Style{Property: "background-color", Value: "#fff"}))
		require.True(t, decls.set(component.Style{Property: "border", Value: "0"}))

		require.Equal(t, "z-index:1;color:red;background-color:#fff;border:0;", decls.String())
	})

	t.Run("a plain declaration keeps the existing value", func(t *testing.T) {
		t.Parallel()

		var decls styleDecls
		require.NoError(t, parseStyleAttribute(&decls, "color:red"))

		require.False(t, decls.set(component.Style{Property: "color", Value: "green"}))
		require.Equal(t, "color:red;", decls.String())
	})

	t.Run("an important declaration overrides in place", func(t *testing.T) {
		t.Parallel()

		var decls styleDecls
		require.NoError(t, parseStyleAttribute(&decls, "color:red;font-size:12px"))

		require.True(t, decls.set(component.Style{Property: "color", Value: "green", Important: true}))
		require.Equal(t, "color:green;font-size:12px;", decls.String())
	})

	t.Run("a repeated property keeps its first position and the last value", func(t *testing.T) {
		t.Parallel()

		var decls styleDecls
		require.NoError(t, parseStyleAttribute(&decls, "color:red;font-size:12px;color:blue"))

		require.Equal(t, "color:blue;font-size:12px;", decls.String())
	})

	t.Run("lookup reads the merged value", func(t *testing.T) {
		t.Parallel()

		var decls styleDecls
		require.NoError(t, parseStyleAttribute(&decls, "width: 100px;"))

		value, ok := decls.lookup("width")
		require.True(t, ok)
		require.Equal(t, "100px", value)

		_, ok = decls.lookup("height")
		require.False(t, ok)
	})

	t.Run("empty declarations serialize to the empty string", func(t *testing.T) {
		t.Parallel()

		require.Equal(t, "", styleDecls(nil).String())
	})

	t.Run("an invalid declaration is rejected", func(t *testing.T) {
		t.Parallel()

		var decls styleDecls

		require.Error(t, parseStyleAttribute(&decls, "color"))
		require.Error(t, parseStyleAttribute(&decls, "color:"))
	})

	t.Run("reset keeps the backing array", func(t *testing.T) {
		t.Parallel()

		decls := make(styleDecls, 0, 4)
		require.NoError(t, parseStyleAttribute(&decls, "color:red;font-size:12px"))

		buffer := &decls[0]
		decls.reset()

		require.Empty(t, decls)

		// Appending again reuses the same backing array, so no allocation happens
		// per matched node.
		require.NoError(t, parseStyleAttribute(&decls, "color:blue"))
		require.Same(t, buffer, &decls[0])
	})
}

// TestWriteRestoredRawBlocks asserts the raw block placeholders are restored in a
// single pass, including for documents that carry none.
func TestWriteRestoredRawBlocks(t *testing.T) {
	t.Parallel()

	t.Run("no blocks writes the document as it is", func(t *testing.T) {
		t.Parallel()

		var out bytes.Buffer
		require.NoError(t, writeRestoredRawBlocks(&out, "<p>plain</p>", nil))
		require.Equal(t, "<p>plain</p>", out.String())
	})

	t.Run("replaces every placeholder", func(t *testing.T) {
		t.Parallel()

		document := "a" + rawBlockPlaceholderPrefix + "0" + rawBlockPlaceholderSuffix +
			"b" + rawBlockPlaceholderPrefix + "1" + rawBlockPlaceholderSuffix + "c"

		var out bytes.Buffer
		require.NoError(t, writeRestoredRawBlocks(&out, document, []string{"<div>", "</div>"}))
		require.Equal(t, "a<div>b</div>c", out.String())
	})

	t.Run("leaves an unknown index alone", func(t *testing.T) {
		t.Parallel()

		document := "a" + rawBlockPlaceholderPrefix + "7" + rawBlockPlaceholderSuffix + "b"

		var out bytes.Buffer
		require.NoError(t, writeRestoredRawBlocks(&out, document, []string{"<div>"}))
		require.Equal(t, document, out.String())
	})

	t.Run("leaves a prefix without a suffix alone", func(t *testing.T) {
		t.Parallel()

		document := "a" + rawBlockPlaceholderPrefix + "0"

		var out bytes.Buffer
		require.NoError(t, writeRestoredRawBlocks(&out, document, []string{"<div>"}))
		require.Equal(t, document, out.String())
	})
}

// TestProtectRawBlocks asserts raw blocks are replaced by placeholders in one
// pass and a document without raw blocks is not copied.
func TestProtectRawBlocks(t *testing.T) {
	t.Parallel()

	t.Run("no raw blocks", func(t *testing.T) {
		t.Parallel()

		const document = "<p>no raw blocks</p>"

		protected, blocks := protectRawBlocks(document)
		require.Equal(t, document, protected)
		require.Empty(t, blocks)
	})

	t.Run("protects and restores raw blocks in order", func(t *testing.T) {
		t.Parallel()

		document := `<p>before</p>` +
			component.RawBlockStartMarker + `<div class="raw">one</div>` + component.RawBlockEndMarker +
			`<p>between</p>` +
			component.RawBlockStartMarker + `<span>two</span>` + component.RawBlockEndMarker

		// The markers themselves are internal: the restored document is the input
		// with the marker pairs removed.
		restoredDocument := `<p>before</p><div class="raw">one</div><p>between</p><span>two</span>`

		protected, blocks := protectRawBlocks(document)
		require.Equal(t, []string{`<div class="raw">one</div>`, `<span>two</span>`}, blocks)
		require.NotContains(t, protected, component.RawBlockStartMarker)

		var restored bytes.Buffer
		require.NoError(t, writeRestoredRawBlocks(&restored, protected, blocks))
		require.Equal(t, restoredDocument, restored.String())
	})
}

// TestInlineCSSStylesRawBlocks asserts protected raw content is parsed as a
// fragment - in a context that fits its own shape - and receives the inline
// declarations and the <mj-html-attributes> selectors, while the markers keep it
// at its position.
func TestInlineCSSStylesRawBlocks(t *testing.T) {
	t.Parallel()

	// A marked block sits directly in a <tbody>, which is where the markers are
	// required. Each block is still parsed in the context its own first element
	// needs, so both a bare row and a nested table survive.
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "bare row stays a row",
			raw:  `<tr><td class="cell">raw row</td></tr>`,
			want: `<tr><td class="cell" data-x="1" style="color:red;">raw row</td></tr>`,
		},
		{
			name: "nested table keeps its table element",
			raw:  `<table><tr><td class="cell">raw cell</td></tr></table>`,
			want: `<table><tbody><tr><td class="cell" data-x="1" style="color:red;">raw cell</td></tr></tbody></table>`,
		},
		{
			name: "generic markup is untouched around the styled node",
			raw:  `<div class="wrapper"><p class="cell">text</p></div>`,
			want: `<div class="wrapper"><p class="cell" data-x="1" style="color:red;">text</p></div>`,
		},
		{
			// A table child is not body content: a table body context would
			// turn the caption into plain text.
			name: "caption stays an element",
			raw:  `<caption class="cell">caption</caption>`,
			want: `<caption class="cell" data-x="1" style="color:red;">caption</caption>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			document := `<html><body><table><tbody>` +
				component.RawBlockStartMarker + tt.raw + component.RawBlockEndMarker +
				`<tr><td>content</td></tr>` +
				`</tbody></table></body></html>`

			ctx := inlineContext(inlineSheet(".cell", component.Style{Property: "color", Value: "red"}))
			ctx.HTMLAttributes = []component.HTMLAttributeSelector{
				{Path: ".cell", Attributes: []component.HTMLAttribute{{Name: "data-x", Value: "1"}}},
			}

			var out bytes.Buffer
			require.NoError(t, inlineCSS(ctx, strings.NewReader(document), &out))

			require.Equal(t, `<html><body><table><tbody>`+
				tt.want+
				`<tr><td>content</td></tr>`+
				`</tbody></table></body></html>`, out.String())
		})
	}
}

// TestRawBlockFragmentContext locks down the context choice: it is the block's
// own first element that decides, not the table body the block sits in.
func TestRawBlockFragmentContext(t *testing.T) {
	t.Parallel()

	tests := []struct {
		block string
		want  string
	}{
		{block: `<tr><td>x</td></tr>`, want: "tbody"},
		{block: `<TD class="cell">x</TD>`, want: "tbody"},
		{block: `<th scope="col">x</th>`, want: "tbody"},
		{block: `<tbody><tr><td>x</td></tr></tbody>`, want: "table"},
		{block: `<thead><tr><th>x</th></tr></thead>`, want: "table"},
		{block: `<tfoot><tr><td>x</td></tr></tfoot>`, want: "table"},
		{block: `<caption>caption</caption>`, want: "table"},
		{block: `<colgroup><col span="2"></colgroup>`, want: "table"},
		{block: `<col span="2">`, want: "table"},
		// A nested table is flow content: "in table body" mode would treat its
		// start tag as an end tag and delete it.
		{block: `<table><tr><td>x</td></tr></table>`, want: "div"},
		{block: `<div><table><tr><td>x</td></tr></table></div>`, want: "div"},
		// Text, comments and doctypes do not decide the context.
		{block: `<!-- <tr><td>commented out</td></tr> --><div>x</div>`, want: "div"},
		{block: `plain text with a < b`, want: "div"},
		{block: ``, want: "div"},
	}

	for _, tt := range tests {
		t.Run(tt.block, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, rawBlockFragmentContext(tt.block).Data)
		})
	}
}

// TestInlineCSSKeepsUnmatchedRawBlocks asserts a block that no rule and no
// selector applies to keeps its exact bytes: comment-only blocks - the common
// case - never get re-serialized.
func TestInlineCSSKeepsUnmatchedRawBlocks(t *testing.T) {
	t.Parallel()

	const raw = `<!--     a comment with 5 spaces     --><div>plain</div>`
	document := `<html><body><table><tbody>` +
		component.RawBlockStartMarker + raw + component.RawBlockEndMarker +
		`<tr><td class="other">content</td></tr>` +
		`</tbody></table></body></html>`

	ctx := inlineContext(inlineSheet(".other", component.Style{Property: "color", Value: "red"}))

	var out bytes.Buffer
	require.NoError(t, inlineCSS(ctx, strings.NewReader(document), &out))

	require.Contains(t, out.String(), raw)
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

			render := func() error {
				_, err := RenderMJML(bytes.NewReader(input))

				return err
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

	selectors := []component.HTMLAttributeSelector{
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
	}

	require.True(t, applyHTMLAttributes(selectors, doc))

	// The pass reports no change when it sets the same values again.
	require.False(t, applyHTMLAttributes(selectors, doc))

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
