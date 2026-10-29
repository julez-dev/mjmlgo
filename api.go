package mjmlgo

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/julez-dev/mjmlgo/component"
)

var (
	errNilFS     = errors.New("mjml: nil fs.FS")
	errNilCtx    = errors.New("mjml: nil context")
	errNilWriter = errors.New("mjml: nil io.Writer")
)

// rendererConfig collects the configuration applied by Option values.
type rendererConfig struct {
	idGenerator    func() string
	dropComments   bool
	fonts          []Font
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

// Font is one entry of the web-font registry: the family a document names in a
// font-family attribute, and the URL of the stylesheet that provides it.
type Font struct {
	// Family is the name documents refer to, e.g. "Inter". It is matched
	// case-sensitively against the families a document mentions. An entry with
	// an empty Family is ignored.
	Family string
	// Href is the stylesheet URL, imported the way MJML imports its defaults.
	Href string
}

// WithFonts replaces the web-font registry MJML imports only for the families a
// document actually uses. WithFonts(nil) keeps the five defaults (Open Sans,
// Droid Sans, Lato, Roboto, Ubuntu); a non-nil slice replaces them completely, so
// a default family is no longer imported unless the slice lists it. An <mj-font>
// element in the document still wins for its family, like it does in MJML.
//
// Fonts are emitted in the order given, which is the order the reference
// implementation emits a registry it received as a JavaScript object. A family
// listed twice keeps its first position and its last URL, like assigning to an
// object property does. FontsFromMap builds the slice from a map for callers
// whose registry has no order worth preserving.
func WithFonts(fonts []Font) Option {
	registry := slices.Clone(fonts)

	return func(config *rendererConfig) {
		// A nil slice means the defaults; an empty one replaces them with
		// nothing.
		if fonts == nil {
			config.fonts = nil

			return
		}

		config.fonts = registry
	}
}

// FontsFromMap converts a family-to-URL map into a registry ordered by family
// name. It is the bridge for callers holding a map: a Go map has no order of its
// own, so this states the order instead of leaving it to chance. FontsFromMap(nil)
// keeps the default registry, like WithFonts(nil).
func FontsFromMap(fonts map[string]string) []Font {
	if fonts == nil {
		return nil
	}

	registry := make([]Font, 0, len(fonts))
	for family, href := range fonts {
		registry = append(registry, Font{Family: family, Href: href})
	}

	slices.SortFunc(registry, func(a, b Font) int { return strings.Compare(a.Family, b.Family) })

	return registry
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
	dropComments bool
	// fonts is the caller's registry in emission order, or nil for the built-in
	// one.
	fonts          []Font
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

// Render renders the MJML document in src and writes the resulting HTML to w.
// Cancellation is honoured before any work starts and between stages.
//
// src is the whole document, so the renderer never touches the filesystem on
// its own and documents with mj-include are rejected with ErrIncludeNoFS: use
// RenderFS to resolve includes against a filesystem.
//
// A bytes.Buffer can be reused across renders: its Reset keeps the capacity, so
// a server can grow the output buffer once and hand the same one to every call.
// strings.Builder cannot be reused for that - its Reset drops the buffer.
func (r *Renderer) Render(ctx context.Context, w io.Writer, src []byte) error {
	return r.renderSource(ctx, w, nil, "", src)
}

// RenderFS reads the MJML document name from fsys, renders it and writes the
// resulting HTML to w. Cancellation is honoured before any work starts.
//
// The document is read through fsys, so rendering never touches the filesystem
// on its own. mj-include is resolved through the same fsys: include paths are
// relative to the directory of the file that contains them, and include cycles,
// paths escaping the filesystem root and unknown include types are hard errors.
func (r *Renderer) RenderFS(ctx context.Context, w io.Writer, fsys fs.FS, name string) error {
	if fsys == nil {
		return errNilFS
	}

	input, err := fs.ReadFile(fsys, name)
	if err != nil {
		return fmt.Errorf("mjml: read %s: %w", name, err)
	}

	return r.renderSource(ctx, w, fsys, name, input)
}

// RenderFile renders the MJML document at path and writes the resulting HTML to
// w. The document's directory is the root passed to the renderer, so relative
// references stay scoped to it.
func (r *Renderer) RenderFile(ctx context.Context, w io.Writer, path string) error {
	return r.RenderFS(ctx, w, os.DirFS(filepath.Dir(path)), filepath.Base(path))
}

// renderSource applies the checks every entry point shares, runs the pipeline
// and writes its result out in one go.
func (r *Renderer) renderSource(ctx context.Context, w io.Writer, fsys fs.FS, name string, src []byte) error {
	if ctx == nil {
		return errNilCtx
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	if w == nil {
		return errNilWriter
	}

	rendered, err := r.renderDocument(ctx, fsys, name, src)
	if err != nil {
		return err
	}

	_, err = io.WriteString(w, rendered)

	return err
}

// idFunc returns the generator to install on a render context. A nil renderer
// or a zero-value Renderer falls back to the package default.
func (r *Renderer) idFunc() func() string {
	if r == nil {
		return nil
	}

	return r.idGenerator
}
