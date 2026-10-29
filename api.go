package mjmlgo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"sync"

	"github.com/julez-dev/mjmlgo/component"
)

var (
	errNilFS  = errors.New("mjml: nil fs.FS")
	errNilCtx = errors.New("mjml: nil context")
)

// rendererConfig collects the configuration applied by Option values.
type rendererConfig struct {
	idGenerator    func() string
	dropComments   bool
	fonts          map[string]string
	ignoreIncludes bool
	printerSupport bool
	components     []component.Component
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
	maps.Copy(registry, fonts)

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

	// components holds the custom components registered with WithComponents,
	// indexed by tag. It is built once and never modified, so concurrent renders
	// share it. componentsErr records a registration the renderer must refuse.
	components    map[string]component.Component
	componentsErr error
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

	components, componentsErr := componentTable(config.components)

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
		components:     components,
		componentsErr:  componentsErr,
	}
}

// Render renders the MJML document in src and returns the resulting HTML.
// Cancellation is honoured before any work starts and between stages.
//
// src is the whole document, so the renderer never touches the filesystem on
// its own and documents with mj-include are rejected with ErrIncludeNoFS: use
// RenderFS to resolve includes against a filesystem.
func (r *Renderer) Render(ctx context.Context, src []byte) ([]byte, error) {
	rendered, err := r.renderSource(ctx, nil, "", src)
	if err != nil {
		return nil, err
	}

	return []byte(rendered), nil
}

// RenderFS reads the MJML document name from fsys, renders it and returns the
// resulting HTML. Cancellation is honoured before any work starts.
//
// The document is read through fsys, so rendering never touches the filesystem
// on its own. mj-include is resolved through the same fsys: include paths are
// relative to the directory of the file that contains them, and include cycles,
// paths escaping the filesystem root and unknown include types are hard errors.
func (r *Renderer) RenderFS(ctx context.Context, fsys fs.FS, name string) ([]byte, error) {
	if fsys == nil {
		return nil, errNilFS
	}

	input, err := fs.ReadFile(fsys, name)
	if err != nil {
		return nil, fmt.Errorf("mjml: read %s: %w", name, err)
	}

	rendered, err := r.renderSource(ctx, fsys, name, input)
	if err != nil {
		return nil, err
	}

	return []byte(rendered), nil
}

// RenderFile renders the MJML document at path and returns the resulting HTML.
// The document's directory is the root passed to the renderer, so relative
// references stay scoped to it.
func (r *Renderer) RenderFile(ctx context.Context, path string) ([]byte, error) {
	return r.RenderFS(ctx, os.DirFS(filepath.Dir(path)), filepath.Base(path))
}

// renderSource applies the context checks every entry point shares and runs
// the pipeline.
func (r *Renderer) renderSource(ctx context.Context, fsys fs.FS, name string, src []byte) (string, error) {
	if ctx == nil {
		return "", errNilCtx
	}

	if err := ctx.Err(); err != nil {
		return "", err
	}

	return r.renderDocument(ctx, fsys, name, bytes.NewReader(src))
}

// idFunc returns the generator to install on a render context. A nil renderer
// or a zero-value Renderer falls back to the package default.
func (r *Renderer) idFunc() func() string {
	if r == nil {
		return nil
	}

	return r.idGenerator
}
