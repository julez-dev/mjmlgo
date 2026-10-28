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
