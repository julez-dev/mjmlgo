package mjmlgo_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/julez-dev/mjmlgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parityDir holds the fixtures that RenderMJML and the Renderer must agree on.
const parityDir = "testdata/parity"

// fixtureFS serves fixture file contents from an in-memory filesystem.
func fixtureFS(t *testing.T, name string) fs.FS {
	t.Helper()

	input, err := os.ReadFile(filepath.Join(parityDir, name))
	require.NoError(t, err)

	return fstest.MapFS{
		name: &fstest.MapFile{Data: input},
	}
}

// TestRendererMatchesRenderMJML asserts that every entry point shares one
// pipeline: the Renderer output is byte-for-byte RenderMJML's output for the
// same document, whether the document comes from an fs.FS or from disk.
func TestRendererMatchesRenderMJML(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	renderer := mjmlgo.NewRenderer()

	for _, name := range []string{"simple.mjml", "content.mjml", "layout.mjml", "head-html-attributes.mjml"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			fixture := filepath.Join(parityDir, name)
			input, err := os.ReadFile(fixture)
			require.NoError(t, err)

			want, err := mjmlgo.RenderMJML(bytes.NewReader(input))
			require.NoError(t, err)

			fsys := fixtureFS(t, name)

			got, err := renderer.RenderString(ctx, fsys, name)
			require.NoError(t, err)
			assert.Equal(t, want, got)

			var output bytes.Buffer
			require.NoError(t, renderer.Render(ctx, fsys, name, &output))
			assert.Equal(t, want, output.String())

			raw, err := renderer.RenderBytes(ctx, fsys, name)
			require.NoError(t, err)
			assert.Equal(t, want, string(raw))

			fromDisk, err := renderer.RenderFile(ctx, fixture)
			require.NoError(t, err)
			assert.Equal(t, want, fromDisk)
		})
	}
}

func TestRendererNilFS(t *testing.T) {
	t.Parallel()

	err := mjmlgo.NewRenderer().Render(context.Background(), nil, "test.mjml", io.Discard)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil fs.FS")

	_, err = mjmlgo.NewRenderer().RenderString(context.Background(), nil, "test.mjml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil fs.FS")
}

func TestRendererCancelledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	fsys := fixtureFS(t, "simple.mjml")

	err := mjmlgo.NewRenderer().Render(ctx, fsys, "simple.mjml", io.Discard)
	require.ErrorIs(t, err, context.Canceled)

	_, err = mjmlgo.NewRenderer().RenderString(ctx, fsys, "simple.mjml")
	require.ErrorIs(t, err, context.Canceled)
}

// TestRendererIDGenerator asserts that generated ids come from the configured
// generator: a constant one makes rendering fully deterministic, while the
// default produces MJML-shaped 16-character lowercase hex ids.
func TestRendererIDGenerator(t *testing.T) {
	t.Parallel()

	const fixedID = "0123456789abcdef"

	fsys := fixtureFS(t, "carousel.mjml")
	ctx := context.Background()

	renderer := mjmlgo.NewRenderer(mjmlgo.WithIDGenerator(func() string { return fixedID }))

	first, err := renderer.RenderString(ctx, fsys, "carousel.mjml")
	require.NoError(t, err)
	second, err := renderer.RenderString(ctx, fsys, "carousel.mjml")
	require.NoError(t, err)

	assert.Equal(t, first, second, "a constant generator must make rendering deterministic")
	assert.Contains(t, first, fixedID)

	random, err := mjmlgo.NewRenderer().RenderString(ctx, fsys, "carousel.mjml")
	require.NoError(t, err)

	generatedID := regexp.MustCompile(`[0-9a-f]{16}`)
	ids := generatedID.FindAllString(random, -1)
	require.NotEmpty(t, ids, "expected generated ids in the default output")

	for _, id := range ids {
		assert.Len(t, id, 16)
		assert.Equal(t, strings.ToLower(id), id)
	}
}

// TestRendererConcurrentUse asserts that a single Renderer can be used from
// several goroutines. The generator intentionally uses unsynchronised state:
// the renderer serialises calls to it, so this is race-free.
func TestRendererConcurrentUse(t *testing.T) {
	t.Parallel()

	var next uint64

	renderer := mjmlgo.NewRenderer(mjmlgo.WithIDGenerator(func() string {
		id := fmt.Sprintf("%016x", next)
		next++

		return id
	}))

	fsys := fixtureFS(t, "navbar.mjml")

	var (
		wg    sync.WaitGroup
		errMu sync.Mutex
		errs  []error
	)

	for range 8 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			if _, err := renderer.RenderString(context.Background(), fsys, "navbar.mjml"); err != nil {
				errMu.Lock()
				errs = append(errs, err)
				errMu.Unlock()
			}
		}()
	}

	wg.Wait()
	assert.Empty(t, errs)
}

// renderSource renders an in-memory document with renderer and returns the
// resulting HTML.
func renderSource(t *testing.T, renderer *mjmlgo.Renderer, source string) string {
	t.Helper()

	fsys := fstest.MapFS{"doc.mjml": &fstest.MapFile{Data: []byte(source)}}

	html, err := renderer.RenderString(context.Background(), fsys, "doc.mjml")
	require.NoError(t, err)

	return html
}

// mediaQueryDocument produces the column media queries printerSupport repeats in
// its print block.
const mediaQueryDocument = `<mjml><mj-body><mj-section>` +
	`<mj-column width="50%"><mj-text>a</mj-text></mj-column>` +
	`</mj-section></mj-body></mjml>`

// TestWithComments asserts the authoring default and the opt-out: comments are
// preserved by default, dropped on request, and comments inside an ending tag
// are content and are always kept.
func TestWithComments(t *testing.T) {
	t.Parallel()

	const document = `<mjml><mj-body><!-- between components -->` +
		`<mj-section><mj-column><mj-text>text<!-- inside a text --></mj-text></mj-column></mj-section>` +
		`</mj-body></mjml>`

	defaultOutput := renderSource(t, mjmlgo.NewRenderer(), document)
	assert.Contains(t, defaultOutput, "<!-- between components -->")
	assert.Contains(t, defaultOutput, "<!-- inside a text -->")

	dropped := renderSource(t, mjmlgo.NewRenderer(mjmlgo.WithComments(false)), document)
	assert.NotContains(t, dropped, "<!-- between components -->")
	assert.Contains(t, dropped, "<!-- inside a text -->", "comments inside an ending tag are content")

	kept := renderSource(t, mjmlgo.NewRenderer(mjmlgo.WithComments(true)), document)
	assert.Contains(t, kept, "<!-- between components -->")
}

// TestWithFonts asserts that a custom registry replaces the MJML default one
// instead of extending it, that WithFonts(nil) means the defaults, and that an
// <mj-font> element still wins for its family.
func TestWithFonts(t *testing.T) {
	t.Parallel()

	const document = `<mjml><mj-head>` +
		`<mj-font name="Document Font" href="https://example.com/document.css" />` +
		`</mj-head><mj-body><mj-section><mj-column>` +
		`<mj-text font-family="Custom Sans, sans-serif">custom</mj-text>` +
		`<mj-text font-family="Document Font, serif">document</mj-text>` +
		`<mj-text font-family="Lato, sans-serif">default</mj-text>` +
		`</mj-column></mj-section></mj-body></mjml>`

	custom := renderSource(t, mjmlgo.NewRenderer(mjmlgo.WithFonts(map[string]string{
		"Custom Sans": "https://example.com/custom-sans.css",
	})), document)

	assert.Contains(t, custom, "https://example.com/custom-sans.css")
	assert.Contains(t, custom, "https://example.com/document.css", "<mj-font> wins for its family")
	assert.NotContains(t, custom, "fonts.googleapis.com/css?family=Lato", "the default registry is replaced")

	defaults := renderSource(t, mjmlgo.NewRenderer(mjmlgo.WithFonts(nil)), document)
	assert.Contains(t, defaults, "fonts.googleapis.com/css?family=Lato")
	assert.NotContains(t, defaults, "https://example.com/custom-sans.css")

	empty := renderSource(t, mjmlgo.NewRenderer(mjmlgo.WithFonts(map[string]string{})), document)
	assert.NotContains(t, empty, "fonts.googleapis.com", "an empty registry replaces the defaults with nothing")
	assert.Contains(t, empty, "https://example.com/document.css")
}

// TestWithIgnoreIncludes asserts that includes are left unexpanded: the element
// disappears, the included file is never read, and the document still renders.
func TestWithIgnoreIncludes(t *testing.T) {
	t.Parallel()

	const document = `<mjml><mj-body><mj-include path="partials/body.mjml" />` +
		`<mj-section><mj-column><mj-text>local</mj-text></mj-column></mj-section>` +
		`</mj-body></mjml>`

	fsys := fstest.MapFS{"doc.mjml": &fstest.MapFile{Data: []byte(document)}}

	_, err := mjmlgo.NewRenderer().RenderString(context.Background(), fsys, "doc.mjml")
	require.Error(t, err, "without ignoreIncludes the missing include is an error")

	html, err := mjmlgo.NewRenderer(mjmlgo.WithIgnoreIncludes(true)).
		RenderString(context.Background(), fsys, "doc.mjml")
	require.NoError(t, err)

	assert.Contains(t, html, "local")
	assert.NotContains(t, html, "partials/body.mjml")
}

// TestWithPrinterSupport asserts the option only adds MJML's print block, and
// only when the document generates media queries.
func TestWithPrinterSupport(t *testing.T) {
	t.Parallel()

	defaultOutput := renderSource(t, mjmlgo.NewRenderer(), mediaQueryDocument)
	assert.NotContains(t, defaultOutput, "@media only print")
	assert.Contains(t, defaultOutput, "@media only screen", "the base media queries are always emitted")

	printOutput := renderSource(t, mjmlgo.NewRenderer(mjmlgo.WithPrinterSupport(true)), mediaQueryDocument)
	assert.Contains(t, printOutput, "@media only print")
	assert.Contains(t, printOutput, ".mj-column-per-50", "the print block reuses the base media queries")

	// A document without media queries - no column, so no width class - gets no
	// print block, like MJML.
	const plainDocument = `<mjml><mj-body><mj-raw><div>raw only</div></mj-raw></mj-body></mjml>`
	assert.NotContains(t,
		renderSource(t, mjmlgo.NewRenderer(mjmlgo.WithPrinterSupport(true)), plainDocument),
		"@media only print")
}
