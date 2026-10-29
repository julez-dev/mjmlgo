package mjmlgo_test

import (
	"context"
	"fmt"
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

// parityDir holds the fixtures the entry points must agree on.
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

// TestEntryPointsShareOnePipeline asserts that every entry point - the package
// function, the memory renderer and the filesystem renderer - produces the same
// bytes for the same document.
func TestEntryPointsShareOnePipeline(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	renderer := mjmlgo.NewRenderer()

	for _, name := range []string{"simple.mjml", "content.mjml", "layout.mjml", "head-html-attributes.mjml"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			fixture := filepath.Join(parityDir, name)
			input, err := os.ReadFile(fixture)
			require.NoError(t, err)

			want, err := mjmlgo.Render(input)
			require.NoError(t, err)

			got, err := renderer.Render(ctx, input)
			require.NoError(t, err)
			assert.Equal(t, want, got)

			fsys := fixtureFS(t, name)

			fromFS, err := renderer.RenderFS(ctx, fsys, name)
			require.NoError(t, err)
			assert.Equal(t, want, fromFS)

			fromDisk, err := renderer.RenderFile(ctx, fixture)
			require.NoError(t, err)
			assert.Equal(t, want, fromDisk)
		})
	}
}

func TestRendererNilFS(t *testing.T) {
	t.Parallel()

	_, err := mjmlgo.NewRenderer().RenderFS(context.Background(), nil, "test.mjml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil fs.FS")
}

func TestRendererCancelledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	renderer := mjmlgo.NewRenderer()

	_, err := renderer.RenderFS(ctx, fixtureFS(t, "simple.mjml"), "simple.mjml")
	require.ErrorIs(t, err, context.Canceled)

	_, err = renderer.Render(ctx, []byte(`<mjml><mj-body /></mjml>`))
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

	first, err := renderFrom(renderer, ctx, fsys, "carousel.mjml")
	require.NoError(t, err)
	second, err := renderFrom(renderer, ctx, fsys, "carousel.mjml")
	require.NoError(t, err)

	assert.Equal(t, first, second, "a constant generator must make rendering deterministic")
	assert.Contains(t, first, fixedID)

	random, err := renderFrom(mjmlgo.NewRenderer(), ctx, fsys, "carousel.mjml")
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
		wg.Go(func() {
			if _, err := renderer.RenderFS(context.Background(), fsys, "navbar.mjml"); err != nil {
				errMu.Lock()
				errs = append(errs, err)
				errMu.Unlock()
			}
		})
	}

	wg.Wait()
	assert.Empty(t, errs)
}

// renderFrom renders name out of fsys and converts the result for assertions
// written against HTML strings.
func renderFrom(renderer *mjmlgo.Renderer, ctx context.Context, fsys fs.FS, name string) (string, error) {
	rendered, err := renderer.RenderFS(ctx, fsys, name)

	return string(rendered), err
}

// renderSource renders an in-memory document with renderer and returns the
// resulting HTML.
func renderSource(t *testing.T, renderer *mjmlgo.Renderer, source string) string {
	t.Helper()

	html, err := renderFrom(renderer, context.Background(), fixtureFSNamed("doc.mjml", source), "doc.mjml")
	require.NoError(t, err)

	return html
}

// fixtureFSNamed serves source as the only file of an in-memory filesystem.
func fixtureFSNamed(name, source string) fs.FS {
	return fstest.MapFS{name: &fstest.MapFile{Data: []byte(source)}}
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

	_, err := mjmlgo.NewRenderer().RenderFS(context.Background(), fsys, "doc.mjml")
	require.Error(t, err, "without ignoreIncludes the missing include is an error")

	html, err := renderFrom(mjmlgo.NewRenderer(mjmlgo.WithIgnoreIncludes(true)),
		context.Background(), fsys, "doc.mjml")
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
