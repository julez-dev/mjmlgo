package mjmlgo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/julez-dev/mjmlgo/internal/component"
)

var (
	errNilFS     = errors.New("mjml: nil fs.FS")
	errNilWriter = errors.New("mjml: nil io.Writer")
	errNilCtx    = errors.New("mjml: nil context")
)

// rendererConfig collects the configuration applied by Option values.
type rendererConfig struct {
	idGenerator func() string
}

// Option configures a Renderer.
type Option func(*rendererConfig)

// WithIDGenerator overrides the source of the opaque ids MJML assigns to
// navbar hamburger labels and carousels. The default is 16 lowercase hex
// characters from crypto/rand, like the reference implementation.
//
// A constant generator makes rendering deterministic. The generator is called
// while the renderer holds its lock, so it does not need to be safe for
// concurrent use itself.
func WithIDGenerator(gen func() string) Option {
	return func(config *rendererConfig) {
		if gen != nil {
			config.idGenerator = gen
		}
	}
}

// Renderer renders MJML documents with the options it was created with. It is
// safe for concurrent use by multiple goroutines. All per-render state lives
// in the individual call.
type Renderer struct {
	// idGenerator is already wrapped in the lock owned by this renderer, so
	// handing it to a render context does not allocate or race.
	idGenerator func() string
}

// NewRenderer returns a Renderer configured with the given options.
func NewRenderer(opts ...Option) *Renderer {
	config := rendererConfig{idGenerator: component.DefaultIDGenerator}

	for _, opt := range opts {
		if opt != nil {
			opt(&config)
		}
	}

	if config.idGenerator == nil {
		config.idGenerator = component.DefaultIDGenerator
	}

	generator := config.idGenerator
	var mu sync.Mutex

	return &Renderer{idGenerator: func() string {
		mu.Lock()
		defer mu.Unlock()

		return generator()
	}}
}

// Render reads the MJML document name from fsys, renders it and writes the
// resulting HTML to w. Cancellation is honoured before any work starts.
//
// The document is read through fsys, so rendering never touches the filesystem
// on its own. mj-include is not implemented yet and is rejected with an
// explicit error rather than being silently dropped.
func (r *Renderer) Render(ctx context.Context, fsys fs.FS, name string, w io.Writer) error {
	if ctx == nil {
		return errNilCtx
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	if fsys == nil {
		return errNilFS
	}

	if w == nil {
		return errNilWriter
	}

	input, err := fs.ReadFile(fsys, name)
	if err != nil {
		return fmt.Errorf("mjml: read %s: %w", name, err)
	}

	rendered, err := r.renderDocument(bytes.NewReader(input))
	if err != nil {
		return err
	}

	_, err = io.WriteString(w, rendered)

	return err
}

// RenderString reads and renders the MJML document name from fsys, returning
// the resulting HTML.
func (r *Renderer) RenderString(ctx context.Context, fsys fs.FS, name string) (string, error) {
	var output strings.Builder

	if err := r.Render(ctx, fsys, name, &output); err != nil {
		return "", err
	}

	return output.String(), nil
}

// RenderBytes reads and renders the MJML document name from fsys, returning
// the resulting HTML.
func (r *Renderer) RenderBytes(ctx context.Context, fsys fs.FS, name string) ([]byte, error) {
	var output bytes.Buffer

	if err := r.Render(ctx, fsys, name, &output); err != nil {
		return nil, err
	}

	return output.Bytes(), nil
}

// RenderFile renders the MJML document at path and returns the resulting HTML.
// The document's directory is the root passed to the renderer, so relative
// references stay scoped to it.
func (r *Renderer) RenderFile(ctx context.Context, path string) (string, error) {
	return r.RenderString(ctx, os.DirFS(filepath.Dir(path)), filepath.Base(path))
}

// idFunc returns the generator to install on a render context. A nil renderer
// or a zero-value Renderer falls back to the package default.
func (r *Renderer) idFunc() func() string {
	if r == nil {
		return nil
	}

	return r.idGenerator
}
