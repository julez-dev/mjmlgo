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
	idGenerator    func() string
	dropComments   bool
	fonts          map[string]string
	ignoreIncludes bool
	printerSupport bool
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

// WithComments controls whether author comments in the document are preserved.
// The default is true, matching MJML: comments are emitted where MJML emits
// them, byte for byte. WithComments(false) drops them while parsing, exactly
// like MJML's keepComments option; comments inside raw content elements
// (mj-text, mj-raw, mj-table, ...) are part of that content and are never
// affected, and neither are the Outlook conditional comments the renderer
// generates itself.
func WithComments(keep bool) Option {
	return func(config *rendererConfig) {
		config.dropComments = !keep
	}
}

// WithFonts replaces the web-font registry MJML imports only for the families a
// document actually uses. WithFonts(nil) keeps the five defaults (Open Sans,
// Droid Sans, Lato, Roboto, Ubuntu); a non-nil map replaces them completely, so
// a default family is no longer imported unless the map lists it. An <mj-font>
// element in the document still wins for its family, like it does in MJML.
//
// Fonts are emitted in ascending family-name order, because a Go map carries no
// order of its own.
func WithFonts(fonts map[string]string) Option {
	registry := make(map[string]string, len(fonts))
	for name, href := range fonts {
		registry[name] = href
	}

	return func(config *rendererConfig) {
		// A nil map means the defaults; an explicit empty map replaces them
		// with nothing.
		if fonts == nil {
			config.fonts = nil

			return
		}

		config.fonts = registry
	}
}

// WithIgnoreIncludes leaves mj-include elements unexpanded, like MJML's
// ignoreIncludes option: the element is dropped from the output and the files it
// names are never read.
func WithIgnoreIncludes(ignore bool) Option {
	return func(config *rendererConfig) {
		config.ignoreIncludes = ignore
	}
}

// WithPrinterSupport adds MJML's printer style block to the head: the same media
// queries the document already generates, repeated inside an `@media only
// print` rule. It has no effect on a document without media queries, matching
// the reference implementation.
func WithPrinterSupport(enable bool) Option {
	return func(config *rendererConfig) {
		config.printerSupport = enable
	}
}

// Renderer renders MJML documents with the options it was created with. It is
// safe for concurrent use by multiple goroutines. All per-render state lives
// in the individual call.
type Renderer struct {
	// idGenerator is already wrapped in the lock owned by this renderer, so
	// handing it to a render context does not allocate or race.
	idGenerator func() string

	// dropComments is stored inverted so that the zero value keeps comments,
	// like a Renderer created without options.
	dropComments   bool
	fonts          map[string]string
	ignoreIncludes bool
	printerSupport bool
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

	return &Renderer{
		idGenerator: func() string {
			mu.Lock()
			defer mu.Unlock()

			return generator()
		},
		dropComments:   config.dropComments,
		fonts:          config.fonts,
		ignoreIncludes: config.ignoreIncludes,
		printerSupport: config.printerSupport,
	}
}

// Render reads the MJML document name from fsys, renders it and writes the
// resulting HTML to w. Cancellation is honoured before any work starts.
//
// The document is read through fsys, so rendering never touches the filesystem
// on its own. mj-include is resolved through the same fsys: include paths are
// relative to the directory of the file that contains them, and include cycles,
// paths escaping the filesystem root and unknown include types are hard errors.
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

	rendered, err := r.renderDocument(ctx, fsys, name, bytes.NewReader(input))
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
