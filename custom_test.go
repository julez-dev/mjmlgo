package mjmlgo_test

import (
	"context"
	"errors"
	"fmt"
	"html"
	"io"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/julez-dev/mjmlgo"
	"github.com/julez-dev/mjmlgo/component"
	"github.com/julez-dev/mjmlgo/example/customcomponent"
	"github.com/julez-dev/mjmlgo/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// renderWithComponents renders document with a renderer that registered the
// given components and returns the HTML.
func renderWithComponents(t *testing.T, document string, components ...component.Component) string {
	t.Helper()

	rendered, err := renderErr(t, document, components...)
	require.NoError(t, err)

	return rendered
}

// renderErr renders document and returns the HTML and the error, so a test can
// inspect the structured failure.
func renderErr(t *testing.T, document string, components ...component.Component) (string, error) {
	t.Helper()

	fsys := fstest.MapFS{"doc.mjml": &fstest.MapFile{Data: []byte(document)}}

	return mjmlgo.NewRenderer(mjmlgo.WithComponents(components...)).
		RenderString(context.Background(), fsys, "doc.mjml")
}

// structuredError requires err to be a *component.Error and returns it.
func structuredError(t *testing.T, err error) *component.Error {
	t.Helper()

	var structured *component.Error
	require.ErrorAs(t, err, &structured)

	return structured
}

// bodyContent returns the rendered content of the body div, so a test can assert
// exactly what a document produced without pinning the head boilerplate that the
// parity corpus already covers. bodyDiv is the opening tag mjml_body.go emits; a
// change to its attributes fails this helper loudly, which is intended: the
// parity corpus is what pins that markup.
func bodyContent(t *testing.T, rendered string) string {
	t.Helper()

	const bodyDiv = `<div aria-roledescription="email" dir="auto" lang="und" role="article">` + "\n"

	start := strings.Index(rendered, bodyDiv)
	require.GreaterOrEqual(t, start, 0, "rendered document has no body div")

	rest := rendered[start+len(bodyDiv):]

	end := strings.Index(rest, "</div>\n</body>")
	require.GreaterOrEqual(t, end, 0, "rendered document has no closing body div")

	return rest[:end]
}

// badgeHTML is the exact output of the example component for the attributes the
// tests below give it.
func badgeHTML(text, background, color, class string) string {
	classAttribute := ""
	if class != "" {
		classAttribute = fmt.Sprintf(` class="%s"`, class)
	}

	return `<table role="presentation" border="0" cellpadding="0" cellspacing="0"` + classAttribute + `>` +
		`<tbody><tr><td align="center" style="background-color:` + background + `;border-radius:4px;color:` + color + `;` +
		`font-family:Helvetica,Arial,sans-serif;font-size:13px;line-height:1;padding:8px 12px;">` +
		text + `</td></tr></tbody></table>`
}

// TestWithComponentsRendersCustomComponent asserts the example component's
// output exactly, at its document position: mj-badge sits directly in mj-body,
// where a content leaf is allowed, and the component's own defaults are applied.
func TestWithComponentsRendersCustomComponent(t *testing.T) {
	t.Parallel()

	const document = `<mjml><mj-body><mj-badge text="Body" /></mj-body></mjml>`

	assert.Equal(t,
		badgeHTML("Body", "#1f2937", "#ffffff", ""),
		bodyContent(t, renderWithComponents(t, document, customcomponent.Badge{})),
	)
}

// TestCustomComponentInColumn asserts a custom component is a content child of
// mj-column: it gets the column's cell wrapper, exactly like a built-in leaf.
func TestCustomComponentInColumn(t *testing.T) {
	t.Parallel()

	const document = `<mjml><mj-body><mj-section><mj-column>` +
		`<mj-badge text="New" background-color="#fde68a" color="#111827" />` +
		`</mj-column></mj-section></mj-body></mjml>`

	rendered := renderWithComponents(t, document, customcomponent.Badge{})

	assert.Contains(t, rendered, "<tr>\n<td style=\"font-size:0px;word-break:break-word;\">\n"+
		badgeHTML("New", "#fde68a", "#111827", "")+"</td>\n</tr>")
}

// TestCustomComponentInHero asserts mj-hero dispatches a registered tag into the
// hero's content cell, like it does for a built-in leaf.
func TestCustomComponentInHero(t *testing.T) {
	t.Parallel()

	const document = `<mjml><mj-body>` +
		`<mj-hero mode="fixed-height" height="120px" background-width="600px" background-height="120px">` +
		`<mj-badge text="Hero" />` +
		`</mj-hero></mj-body></mjml>`

	rendered := renderWithComponents(t, document, customcomponent.Badge{})

	assert.Contains(t, rendered, badgeHTML("Hero", "#1f2937", "#ffffff", ""))
}

// TestCustomComponentReceivesAttributeDefaults asserts the document-level
// default layers reach a custom tag: per-tag defaults from mj-attributes and
// attributes from a referenced mj-class are applied before Render runs, the
// component's own default fills what neither layer set, and the universal
// css-class survives document validation.
func TestCustomComponentReceivesAttributeDefaults(t *testing.T) {
	t.Parallel()

	const document = `<mjml><mj-head><mj-attributes>` +
		`<mj-badge text="Default label" />` +
		`<mj-class name="accent" background-color="#fde68a" />` +
		`</mj-attributes></mj-head><mj-body>` +
		`<mj-badge mj-class="accent" css-class="promo" />` +
		`</mj-body></mjml>`

	assert.Equal(t,
		badgeHTML("Default label", "#fde68a", "#ffffff", "promo"),
		bodyContent(t, renderWithComponents(t, document, customcomponent.Badge{})),
	)
}

// TestCustomComponentValidationErrors asserts every validation failure a custom
// tag can cause is a structured error carrying the element path.
func TestCustomComponentValidationErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		document  string
		kind      component.ErrorKind
		path      []string
		attribute string
		value     string
	}{
		{
			name: "undeclared attribute",
			document: `<mjml><mj-body><mj-section><mj-column>` +
				`<mj-badge text="New" made-up="value" />` +
				`</mj-column></mj-section></mj-body></mjml>`,
			kind:      component.KindUnknownAttribute,
			path:      []string{"mj-body", "mj-section", "mj-column", "mj-badge"},
			attribute: "made-up",
			value:     "value",
		},
		{
			name: "invalid declared value",
			document: `<mjml><mj-body><mj-section><mj-column>` +
				`<mj-badge text="New" background-color="chartreuseish" />` +
				`</mj-column></mj-section></mj-body></mjml>`,
			kind:      component.KindInvalidValue,
			path:      []string{"mj-body", "mj-section", "mj-column", "mj-badge"},
			attribute: "background-color",
			value:     "chartreuseish",
		},
		{
			name: "disallowed child of a leaf",
			document: `<mjml><mj-body><mj-section><mj-column>` +
				`<mj-badge text="New"><mj-text>inside</mj-text></mj-badge>` +
				`</mj-column></mj-section></mj-body></mjml>`,
			kind:  component.KindDisallowedChild,
			path:  []string{"mj-body", "mj-section", "mj-column", "mj-badge", "mj-text"},
			value: "mj-text",
		},
		{
			name: "custom component where a layout element is expected",
			document: `<mjml><mj-body><mj-section>` +
				`<mj-badge text="New" />` +
				`</mj-section></mj-body></mjml>`,
			kind:  component.KindDisallowedChild,
			path:  []string{"mj-body", "mj-section", "mj-badge"},
			value: "mj-badge",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := renderErr(t, tt.document, customcomponent.Badge{})
			require.Error(t, err)

			structured := structuredError(t, err)
			assert.Equal(t, tt.kind, structured.Kind)
			assert.Equal(t, tt.path, structured.Path)
			assert.Equal(t, tt.attribute, structured.Attribute)
			assert.Equal(t, tt.value, structured.Value)
			assert.ErrorIs(t, err, component.ErrValidation)
		})
	}
}

// namedComponent is a component whose only interesting property is its tag, for
// the registration tests.
type namedComponent struct {
	tag string
}

func (c namedComponent) Name() string { return c.tag }

func (namedComponent) AllowedAttributes() map[string]component.AttributeValidator { return nil }

func (namedComponent) DefaultAttributes(*component.RenderContext) map[string]string { return nil }

func (namedComponent) Render(*component.RenderContext, io.Writer, *node.Node) error { return nil }

// childNamedComponent is a namedComponent that declares children, for the
// registration tests that are about a container's declared child list.
type childNamedComponent struct {
	namedComponent
	children []string
}

func (c childNamedComponent) AllowedChildren() []string { return c.children }

// TestWithComponentsRejectsInvalidRegistration asserts a renderer refuses a
// component it cannot register, with a structured error instead of a plain one.
// The failure is about the renderer's configuration, so it applies to every
// document, including one that never uses the tag, and its path names the tag.
func TestWithComponentsRejectsInvalidRegistration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		renderer *mjmlgo.Renderer
		path     []string
		value    string
	}{
		{
			name:     "duplicate tag",
			renderer: mjmlgo.NewRenderer(mjmlgo.WithComponents(customcomponent.Badge{}, customcomponent.Badge{})),
			path:     []string{customcomponent.BadgeTagName},
			value:    customcomponent.BadgeTagName,
		},
		{
			name: "duplicate tag across option calls",
			renderer: mjmlgo.NewRenderer(
				mjmlgo.WithComponents(customcomponent.Badge{}),
				mjmlgo.WithComponents(customcomponent.Badge{}),
			),
			path:  []string{customcomponent.BadgeTagName},
			value: customcomponent.BadgeTagName,
		},
		{
			name:     "tag shadowing a built-in element",
			renderer: mjmlgo.NewRenderer(mjmlgo.WithComponents(namedComponent{tag: component.TextTagName})),
			path:     []string{component.TextTagName},
			value:    component.TextTagName,
		},
		{
			name:     "tag shadowing the include element",
			renderer: mjmlgo.NewRenderer(mjmlgo.WithComponents(namedComponent{tag: component.IncludeTagName})),
			path:     []string{component.IncludeTagName},
			value:    component.IncludeTagName,
		},
		{
			name:     "empty tag",
			renderer: mjmlgo.NewRenderer(mjmlgo.WithComponents(namedComponent{tag: ""})),
			path:     []string{component.MJMLTagName},
		},
		{
			name:     "nil component",
			renderer: mjmlgo.NewRenderer(mjmlgo.WithComponents(nil)),
			path:     []string{component.MJMLTagName},
		},
		{
			name:     "typed nil component",
			renderer: mjmlgo.NewRenderer(mjmlgo.WithComponents((*customcomponent.Badge)(nil))),
			path:     []string{component.MJMLTagName},
		},
		{
			name: "container declaring a head element",
			renderer: mjmlgo.NewRenderer(mjmlgo.WithComponents(
				childNamedComponent{
					namedComponent: namedComponent{tag: "mj-panel"},
					children:       []string{component.TitleTagName},
				},
			)),
			path:  []string{"mj-panel"},
			value: component.TitleTagName,
		},
		{
			name: "container declaring an unknown element",
			renderer: mjmlgo.NewRenderer(mjmlgo.WithComponents(
				childNamedComponent{
					namedComponent: namedComponent{tag: "mj-panel"},
					children:       []string{"mj-nope"},
				},
			)),
			path:  []string{"mj-panel"},
			value: "mj-nope",
		},
		{
			name: "container declaring a layout element",
			renderer: mjmlgo.NewRenderer(mjmlgo.WithComponents(
				childNamedComponent{
					namedComponent: namedComponent{tag: "mj-panel"},
					children:       []string{component.SectionTagName},
				},
			)),
			path:  []string{"mj-panel"},
			value: component.SectionTagName,
		},
	}

	const document = `<mjml><mj-body><mj-section><mj-column>` +
		`<mj-text>built-in only</mj-text>` +
		`</mj-column></mj-section></mj-body></mjml>`

	fsys := fstest.MapFS{"doc.mjml": &fstest.MapFile{Data: []byte(document)}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := tt.renderer.RenderString(context.Background(), fsys, "doc.mjml")
			require.Error(t, err)

			structured := structuredError(t, err)
			assert.Equal(t, component.KindInvalidComponent, structured.Kind)
			assert.Equal(t, tt.path, structured.Path)
			assert.Equal(t, tt.value, structured.Value)
			assert.NotErrorIs(t, err, component.ErrValidation, "registration is not document validation")
		})
	}
}

// TestCustomComponentsArePerRenderer asserts there is no global registry: the
// same document is accepted by a configured renderer and rejected by an
// unconfigured one in the same process.
func TestCustomComponentsArePerRenderer(t *testing.T) {
	t.Parallel()

	const document = `<mjml><mj-body><mj-section><mj-column>` +
		`<mj-badge text="New" />` +
		`</mj-column></mj-section></mj-body></mjml>`

	assert.Contains(t, renderWithComponents(t, document, customcomponent.Badge{}), ">New</td>")

	_, err := renderErr(t, document)
	require.Error(t, err)

	structured := structuredError(t, err)
	assert.Equal(t, component.KindUnknownElement, structured.Kind)
	assert.Equal(t, "mj-badge", structured.Value)
}

// chipComponent is a tiny leaf component that a custom container renders.
type chipComponent struct{}

func (chipComponent) Name() string { return "mj-chip" }

func (chipComponent) AllowedAttributes() map[string]component.AttributeValidator {
	return map[string]component.AttributeValidator{"label": nil}
}

func (chipComponent) DefaultAttributes(*component.RenderContext) map[string]string { return nil }

func (chipComponent) Render(_ *component.RenderContext, w io.Writer, n *node.Node) error {
	_, err := fmt.Fprintf(w, `<span class="mj-chip">%s</span>`, html.EscapeString(n.GetAttributeValueDefault("label")))

	return err
}

// panelComponent is a ChildComponent: it declares mj-chip as its only allowed
// child and renders that child itself, through the exported context and node it
// receives. Dispatching built-in children into a custom tag is ticket T4.
type panelComponent struct{}

func (panelComponent) Name() string { return "mj-panel" }

func (panelComponent) AllowedAttributes() map[string]component.AttributeValidator { return nil }

func (panelComponent) DefaultAttributes(*component.RenderContext) map[string]string { return nil }

func (panelComponent) AllowedChildren() []string { return []string{"mj-chip"} }

func (panelComponent) Render(ctx *component.RenderContext, w io.Writer, n *node.Node) error {
	_, _ = io.WriteString(w, `<div class="mj-panel">`)

	for _, child := range n.Children {
		childComponent, ok := ctx.CustomComponents[child.Type]
		if !ok {
			return fmt.Errorf("no component registered for <%s>", child.Type)
		}

		if err := component.InitComponent(ctx, childComponent, child); err != nil {
			return err
		}

		if err := childComponent.Render(ctx, w, child); err != nil {
			return err
		}
	}

	_, err := io.WriteString(w, `</div>`)

	return err
}

// TestCustomComponentAllowedChildren asserts a custom component's declared child
// list is enforced in both directions, and that a ChildComponent renders the
// children it accepted itself.
func TestCustomComponentAllowedChildren(t *testing.T) {
	t.Parallel()

	t.Run("a declared child is accepted and rendered", func(t *testing.T) {
		t.Parallel()

		const document = `<mjml><mj-body><mj-section><mj-column>` +
			`<mj-panel><mj-chip label="One" /><mj-chip label="Two" /></mj-panel>` +
			`</mj-column></mj-section></mj-body></mjml>`

		rendered := renderWithComponents(t, document, panelComponent{}, chipComponent{})

		assert.Contains(t, rendered,
			`<div class="mj-panel"><span class="mj-chip">One</span><span class="mj-chip">Two</span></div>`)
	})

	t.Run("an undeclared child is rejected", func(t *testing.T) {
		t.Parallel()

		const document = `<mjml><mj-body><mj-section><mj-column>` +
			`<mj-panel><mj-image src="a.png" /></mj-panel>` +
			`</mj-column></mj-section></mj-body></mjml>`

		_, err := renderErr(t, document, panelComponent{}, chipComponent{})
		require.Error(t, err)

		structured := structuredError(t, err)
		assert.Equal(t, component.KindDisallowedChild, structured.Kind)
		assert.Equal(t, []string{"mj-body", "mj-section", "mj-column", "mj-panel", "mj-image"}, structured.Path)
		assert.Equal(t, "mj-image", structured.Value)
	})
}

// TestComponentRenderErrorIsPropagated asserts a custom component's render
// failure stops the render and reaches the caller.
func TestComponentRenderErrorIsPropagated(t *testing.T) {
	t.Parallel()

	const document = `<mjml><mj-body><mj-section><mj-column>` +
		`<mj-badge text="New" />` +
		`</mj-column></mj-section></mj-body></mjml>`

	_, err := renderErr(t, document, failingComponent{})
	require.Error(t, err)
	assert.ErrorIs(t, err, errCustomRender)
}

var errCustomRender = errors.New("custom render failed")

// failingComponent is a custom component whose Render always fails.
type failingComponent struct{}

func (failingComponent) Name() string { return customcomponent.BadgeTagName }

// AllowedAttributes also covers the nil-validator rule: the entry accepts every
// value instead of panicking.
func (failingComponent) AllowedAttributes() map[string]component.AttributeValidator {
	return map[string]component.AttributeValidator{"text": nil}
}

func (failingComponent) DefaultAttributes(*component.RenderContext) map[string]string {
	return nil
}

func (failingComponent) Render(*component.RenderContext, io.Writer, *node.Node) error {
	return errCustomRender
}

// noticeHTML is the exact wrapper the example container renders around its
// children.
func noticeHTML(children string) string {
	return `<table role="presentation" border="0" cellpadding="0" cellspacing="0" class="mj-notice mj-notice-fluid">` +
		`<tbody><tr><td align="left" style="background-color:#fef3c7;border:1px solid #f59e0b;border-radius:6px;` +
		`font-family:Helvetica,Arial,sans-serif;font-size:13px;line-height:1;padding:12px 16px;">` +
		children + `</td></tr></tbody></table>`
}

// textHTML is the exact markup mj-text renders with its defaults.
func textHTML(content string) string {
	return `<div style="font-family:Ubuntu, Helvetica, Arial, sans-serif;font-size:13px;line-height:1;` +
		`text-align:left;color:#000000;">` + content + "</div>\n"
}

// TestCustomContainerRendersChildrenAndWrapper asserts the exact column cell of
// a custom container at its document position: the container's own wrapper
// around the built-in child, rendered by that child's own component.
func TestCustomContainerRendersChildrenAndWrapper(t *testing.T) {
	t.Parallel()

	const document = `<mjml><mj-body><mj-section><mj-column>` +
		`<mj-notice><mj-text>Hello</mj-text></mj-notice>` +
		`</mj-column></mj-section></mj-body></mjml>`

	assert.Contains(t,
		bodyContent(t, renderWithComponents(t, document, customcomponent.Notice{})),
		"<tr>\n<td style=\"font-size:0px;word-break:break-word;\">\n"+
			noticeHTML(textHTML("Hello"))+
			"</td>\n</tr>",
	)
}

// TestCustomContainerAppliesChildDefaults asserts a container's built-in child
// takes part in the document's default cascade: the per-tag defaults from
// mj-attributes are applied to the child before it renders.
func TestCustomContainerAppliesChildDefaults(t *testing.T) {
	t.Parallel()

	const document = `<mjml><mj-head><mj-attributes>` +
		`<mj-text color="#123456" />` +
		`</mj-attributes></mj-head><mj-body><mj-section><mj-column>` +
		`<mj-notice><mj-text>Hello</mj-text></mj-notice>` +
		`</mj-column></mj-section></mj-body></mjml>`

	rendered := renderWithComponents(t, document, customcomponent.Notice{})

	assert.Contains(t, rendered, "text-align:left;color:#123456;\">Hello</div>")
}

// TestCustomContainerHeadCSSIsDeterministic asserts the CSS a container
// contributes is emitted once per document - the named block and the class rules
// are de-duplicated across instances - and that two renders of the same document
// produce the same bytes.
func TestCustomContainerHeadCSSIsDeterministic(t *testing.T) {
	t.Parallel()

	const document = `<mjml><mj-body><mj-section><mj-column>` +
		`<mj-notice><mj-text>One</mj-text></mj-notice>` +
		`<mj-notice><mj-text>Two</mj-text></mj-notice>` +
		`</mj-column></mj-section></mj-body></mjml>`

	first := renderWithComponents(t, document, customcomponent.Notice{})
	second := renderWithComponents(t, document, customcomponent.Notice{})

	assert.Equal(t, first, second, "repeated renders must be byte-identical")
	assert.Equal(t, 1, strings.Count(first, ".mj-notice { padding: 16px !important; }"), "the named block is emitted once")
	assert.Equal(t, 2, strings.Count(first, ".mj-notice-inline {"),
		"the media-query class is emitted once, in the base and Thunderbird variants")
	assert.Equal(t, 1, strings.Count(first, "table.mj-notice-fluid { width: 100% !important; }"),
		"the fluid-on-mobile rule is emitted once")
	assert.Contains(t, first, "@media only screen and (max-width:479px) {")
	assert.Contains(t, first, "@media only screen and (min-width:480px) {")
}

// TestCustomContainerRejectsUndeclaredChild asserts a child outside a
// container's declared child list fails document validation with the element
// path, exactly like a built-in container's disallowed child.
func TestCustomContainerRejectsUndeclaredChild(t *testing.T) {
	t.Parallel()

	const document = `<mjml><mj-body><mj-section><mj-column>` +
		`<mj-notice><mj-image src="a.png" /></mj-notice>` +
		`</mj-column></mj-section></mj-body></mjml>`

	_, err := renderErr(t, document, customcomponent.Notice{})
	require.Error(t, err)

	structured := structuredError(t, err)
	assert.Equal(t, component.KindDisallowedChild, structured.Kind)
	assert.Equal(t, []string{"mj-body", "mj-section", "mj-column", "mj-notice", "mj-image"}, structured.Path)
	assert.Equal(t, "mj-image", structured.Value)
	assert.ErrorIs(t, err, component.ErrValidation)
	assert.Contains(t, err.Error(), "allowed children: mj-text")
}

// rawTableComponent is a container that places its children directly in a table
// body. marked selects the helper that states that position in the markup
// (RenderChildrenInTableBody) over the one that does not (RenderChildren), so a
// test can render the same document through both.
type rawTableComponent struct {
	marked bool
}

func (rawTableComponent) Name() string { return "mj-raw-table" }

func (rawTableComponent) AllowedAttributes() map[string]component.AttributeValidator { return nil }

func (rawTableComponent) DefaultAttributes(*component.RenderContext) map[string]string {
	return nil
}

func (rawTableComponent) AllowedChildren() []string { return []string{component.RawTagName} }

func (c rawTableComponent) Render(ctx *component.RenderContext, w io.Writer, n *node.Node) error {
	_, _ = io.WriteString(w, `<table class="raw-table"><tbody>`)

	renderChildren := component.RenderChildren
	if c.marked {
		renderChildren = component.RenderChildrenInTableBody
	}

	if err := renderChildren(ctx, w, n); err != nil {
		return err
	}

	_, err := io.WriteString(w, `</tbody></table>`)

	return err
}

// TestCustomContainerKeepsRawChildrenInPlace asserts a container that puts its
// children in a table body keeps mj-raw content where the container placed it,
// with an inline stylesheet in play, and that stating the position with the
// raw-block markers or without them produces the same document: the markers are
// transport, not a styling mechanism (the inline stage styles a raw block where it
// stands).
func TestCustomContainerKeepsRawChildrenInPlace(t *testing.T) {
	t.Parallel()

	const document = `<mjml><mj-head><mj-style inline="inline">.raw { color: red; }</mj-style></mj-head>` +
		`<mj-body><mj-section><mj-column>` +
		`<mj-raw-table><mj-raw><tr><td class="raw">raw</td></tr></mj-raw></mj-raw-table>` +
		`</mj-column></mj-section></mj-body></mjml>`

	marked := renderWithComponents(t, document, rawTableComponent{marked: true})
	unmarked := renderWithComponents(t, document, rawTableComponent{})

	assert.Equal(t, marked, unmarked, "the raw-child helper must not change the document")

	// The raw row stays inside the container's table, and the inliner styles it
	// where it stands.
	assert.Contains(t, marked,
		`<table class="raw-table"><tbody><tr><td class="raw" style="color:red;">raw</td></tr></tbody></table>`)

	// The marker itself is stripped from the output.
	assert.NotContains(t, marked, component.RawBlockStartMarker)
	assert.NotContains(t, marked, component.RawBlockEndMarker)
}

// divContainerComponent is a container that wraps its children in a plain <div>.
// marked selects the table-body helper over the plain one, so a test can render
// the same document through both: the position is not a table body either way, and
// the choice must not change the document.
type divContainerComponent struct {
	marked bool
}

func (divContainerComponent) Name() string { return "mj-div-box" }

func (divContainerComponent) AllowedAttributes() map[string]component.AttributeValidator {
	return nil
}

func (divContainerComponent) DefaultAttributes(*component.RenderContext) map[string]string {
	return nil
}

func (divContainerComponent) AllowedChildren() []string { return []string{component.RawTagName} }

func (c divContainerComponent) Render(ctx *component.RenderContext, w io.Writer, n *node.Node) error {
	_, _ = io.WriteString(w, `<div class="wrap">`)

	renderChildren := component.RenderChildren
	if c.marked {
		renderChildren = component.RenderChildrenInTableBody
	}

	if err := renderChildren(ctx, w, n); err != nil {
		return err
	}

	_, err := io.WriteString(w, `</div>`)

	return err
}

// TestCustomContainerRawOutsideTableIsNotMarked asserts a container that does not
// use a table body renders raw content without fabricating table structure around
// it, and that the helper it chooses does not change the result: a bare <td> must
// not come back wrapped in an invented <tr> either way.
func TestCustomContainerRawOutsideTableIsNotMarked(t *testing.T) {
	t.Parallel()

	const document = `<mjml><mj-head><mj-style inline="inline">.raw { color: red; }</mj-style></mj-head>` +
		`<mj-body><mj-section><mj-column>` +
		`<mj-div-box><mj-raw><td class="raw">in-container</td></mj-raw></mj-div-box>` +
		`</mj-column></mj-section></mj-body></mjml>`

	rendered := renderWithComponents(t, document, divContainerComponent{})
	marked := renderWithComponents(t, document, divContainerComponent{marked: true})

	assert.Equal(t, rendered, marked, "the raw-child helper must not change the document")

	start := strings.Index(rendered, `<div class="wrap">`)
	require.GreaterOrEqual(t, start, 0, "the container wrapper is rendered")
	end := strings.Index(rendered[start:], "</div>")
	require.GreaterOrEqual(t, end, 0, "the container wrapper is closed")

	block := rendered[start : start+end]

	assert.NotContains(t, block, "<tr>", "no table row is fabricated around the raw cell")
	assert.NotContains(t, block, component.RawBlockStartMarker)
	assert.Contains(t, rendered, "in-container", "the raw content is not dropped")
}
