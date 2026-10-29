// Package mjmlgo renders MJML documents to HTML.
//
// The implementation targets MJML 5.4.1 exactly: for the supported feature
// set it produces the same document as the reference JavaScript
// implementation. Output is compact (never pretty-printed) and validation is
// fail-fast: unknown tags, invalid attributes or nesting, and missing required
// children are reported as errors instead of being warned about and ignored.
// Invalid input never produces partial output.
//
// Every validation and MJML parse rejection is a structured *Error: inspect its
// Kind, Path, Attribute and Value instead of matching the message text. I/O
// failures that are not about the document's content - an unreadable top-level
// file, a nil argument, a cancelled context - keep their own plain errors. errors.Is still finds the
// package's sentinels (ErrValidation, ErrParsingFailed, ErrIncludeCycle, ...)
// and errors.As unwraps through wrapping, so both inspection styles keep
// working. The kinds are a closed set; see the ErrorKind constants.
//
// Render writes the HTML for an in-memory document to a writer, and
// NewRenderer configures a renderer first, for example to drop author
// comments:
//
//	var out bytes.Buffer
//	err := mjmlgo.Render(&out, src)
//
//	renderer := mjmlgo.NewRenderer(mjmlgo.WithComments(false))
//	err = renderer.Render(ctx, &out, src)
//
// The output goes to the writer the caller supplies, so a bytes.Buffer that is
// Reset between renders keeps its capacity and a server allocates the output
// buffer once.
//
// A document that uses mj-include needs a filesystem: Renderer.RenderFS reads
// the top-level document through an fs.FS and resolves includes against it, so
// callers decide where documents come from and the renderer performs no
// implicit filesystem or network access. Rendering such a document without one
// is an ErrIncludeNoFS error rather than a document with holes in it;
// WithoutIncludes asks for the reference implementation's own default, where the
// element is dropped and the file it names is never opened. A Renderer is safe
// for concurrent use.
//
// WithComponents adds custom components to a renderer. A host implements
// component.Component - tag, allowed attributes with validators, defaults and
// render - and registers it; a registered tag is then a first-class element,
// with the same validation and default cascade as a built-in one. Registration
// is per renderer, never global. Implement component.ChildComponent to declare
// child elements. example/customcomponent is the worked example, and package
// component documents the contract.
//
// mj-include is resolved through the same fs.FS: paths are relative to the
// directory of the file that contains the include, and type="html", type="css"
// and the default type="mjml" are supported. Include cycles, paths that would
// leave the filesystem and unknown include types are errors.
//
// The tree the renderer walks and the components it dispatches to live in the
// public node and component packages, so that custom components can receive the
// same values. Their exported shape is frozen; their package documentation
// lists the exact set, and changing any of it is a breaking change.
//
// Renderer.RenderFile is the shorthand for a document on disk: the file's own
// directory becomes the filesystem root, so relative includes stay scoped to it.
package mjmlgo
