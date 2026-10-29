package mjmlgo

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"maps"
	"strings"

	"github.com/julez-dev/mjmlgo/component"
	"github.com/julez-dev/mjmlgo/node"
)

// ErrUnknownStartingTag reports a document whose root element is not <mjml>.
var ErrUnknownStartingTag = errors.New("mjml: unknown starting tag")

// defaultRenderer backs RenderMJML. It has no mutable per-render state, so a
// single shared instance is safe for concurrent use.
var defaultRenderer = NewRenderer()

// RenderMJML renders the MJML document read from input and returns the
// resulting HTML.
//
// Deprecated: use NewRenderer and Renderer.Render, Renderer.RenderString,
// Renderer.RenderBytes or Renderer.RenderFile instead. RenderMJML is a thin
// wrapper around the default Renderer; it has no filesystem, so documents that
// use mj-include are rejected with an error.
func RenderMJML(input io.Reader) (string, error) {
	return defaultRenderer.renderDocument(context.Background(), nil, "", input)
}

// renderDocument is the single rendering pipeline shared by every entry point.
// fsys and name identify the document for mj-include resolution; fsys is nil
// when the caller has no filesystem of its own.
//
// Cancellation is checked between stages: parsing, include expansion, rendering
// and inlining all happen in memory, so a cancelled context cannot interrupt
// them part-way, but a caller is not left waiting for the remaining stages.
func (r *Renderer) renderDocument(callCtx context.Context, fsys fs.FS, name string, input io.Reader) (string, error) {
	if err := callCtx.Err(); err != nil {
		return "", err
	}

	// A renderer whose custom components could not be registered renders
	// nothing: the failure is about its configuration, not about this document,
	// so it is reported identically for every document.
	if r.componentsErr != nil {
		return "", r.componentsErr
	}

	emitted, err := r.renderMarkup(callCtx, fsys, name, input)
	if err != nil {
		return "", err
	}

	rendered := emitted.markup

	// The inline stage patches the emitted markup in place (see inlineCSS): a
	// document with neither <mj-style inline> nor <mj-html-attributes> has
	// nothing to apply, so its markup is already final.
	if len(emitted.ctx.InlineStyles) > 0 || len(emitted.ctx.HTMLAttributes) > 0 {
		var inlined strings.Builder
		if err := inlineCSS(emitted.ctx, strings.NewReader(rendered), &inlined); err != nil {
			return "", err
		}

		rendered = inlined.String()
	}

	// A container that renders its children directly in a table body marks raw
	// content with the raw-block comments, so the position the author wrote it in
	// stays explicit. The markers are internal transport and never reach the
	// caller: the inline stage matches the emitted markup itself and needs no
	// protection, so stripping them is all that is left of that machinery.
	rendered = component.StripRawBlockMarkers(rendered)

	// MJML merges adjacent Outlook conditional blocks after rendering, so
	// contiguous MSO fragments form a single conditional comment.
	html := component.MergeOutlookConditionals(rendered)

	if beforeDoctype := fileStartRawContent(emitted.doc); beforeDoctype != "" {
		return beforeDoctype + "\n" + html, nil
	}

	return html, nil
}

// emittedMarkup is the document as the component tree emitted it, before the
// inline stage and the post-processing steps: the markup itself, the context the
// head filled in (the inline stylesheets, the <mj-html-attributes> selectors and
// everything else a caller can inspect) and the parsed document, which still
// holds the position="file-start" raw content.
type emittedMarkup struct {
	markup string
	ctx    *component.RenderContext
	doc    *node.Node
}

// renderMarkup runs the front half of the pipeline: parse, include expansion and
// the component render. Its result is what the inline stage works on.
func (r *Renderer) renderMarkup(callCtx context.Context, fsys fs.FS, name string, input io.Reader) (emittedMarkup, error) {
	if err := callCtx.Err(); err != nil {
		return emittedMarkup{}, err
	}

	doc, err := parse(input, !r.dropComments)
	if err != nil {
		return emittedMarkup{}, err
	}

	if doc.Type != "mjml" {
		return emittedMarkup{}, component.NewError(KindUnknownElement, []string{doc.Type}, "", doc.Type, ErrUnknownStartingTag)
	}

	if err := callCtx.Err(); err != nil {
		return emittedMarkup{}, err
	}

	if r.ignoreIncludes {
		stripIncludes(doc)
	} else if err := resolveIncludes(callCtx, fsys, name, doc, !r.dropComments); err != nil {
		return emittedMarkup{}, err
	}

	if err := callCtx.Err(); err != nil {
		return emittedMarkup{}, err
	}

	var buff strings.Builder
	mjml := component.MJML{}

	ctx := &component.RenderContext{
		MJMLStylesheet: make(map[string][]string),
		Fonts:          r.fontRegistry(),
		// A caller-provided registry replaces the built-in one instead of
		// being extended with it.
		FontsConfigured: r.fonts != nil,
		PrinterSupport:  r.printerSupport,
		IDGenerator:     r.idFunc(),
		// The table belongs to the renderer and is immutable after
		// construction, so concurrent renders may share it.
		CustomComponents: r.components,
	}
	if err := component.InitComponent(ctx, mjml, doc); err != nil {
		return emittedMarkup{}, err
	}

	if err := mjml.Render(ctx, &buff, doc); err != nil {
		return emittedMarkup{}, err
	}

	if err := callCtx.Err(); err != nil {
		return emittedMarkup{}, err
	}

	return emittedMarkup{markup: buff.String(), ctx: ctx, doc: doc}, nil
}

// fontRegistry returns the web-font registry to seed the render context with.
// The renderer's map is copied because rendering registers <mj-font> families in
// the context and a Renderer may be used concurrently.
func (r *Renderer) fontRegistry() map[string]string {
	if r.fonts == nil {
		return make(map[string]string)
	}

	registry := make(map[string]string, len(r.fonts))
	maps.Copy(registry, r.fonts)

	return registry
}
