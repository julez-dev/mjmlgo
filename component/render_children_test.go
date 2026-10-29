package component

import (
	"io"
	"strings"
	"testing"

	"github.com/julez-dev/mjmlgo/v2/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// textHTML is the exact markup mj-text renders for content with its default
// attributes (mj-text's own style; the padding default belongs to the <td> its
// container wraps it in). It is written out rather than produced by the
// component so a change to mj-text's defaults fails this expectation.
func textHTML(content string) string {
	return `<div style="font-family:Ubuntu, Helvetica, Arial, sans-serif;font-size:13px;line-height:1;` +
		`text-align:left;color:#000000;">` + content + "</div>\n"
}

// renderingComponent is a custom component that writes html and nothing else, so
// the dispatch tests can see exactly which child rendered.
type renderingComponent struct {
	tag  string
	html string
}

func (c renderingComponent) Name() string { return c.tag }

func (renderingComponent) AllowedAttributes() map[string]AttributeValidator { return nil }

func (renderingComponent) DefaultAttributes(*RenderContext) map[string]string { return nil }

func (c renderingComponent) Render(_ *RenderContext, w io.Writer, n *node.Node) error {
	_, err := io.WriteString(w, c.html)

	return err
}

// TestRenderChildrenRendersBuiltInChildren asserts a container's children are
// rendered by the built-in components in document order, with their defaults
// applied to the node in place, exactly as the built-in content containers do.
func TestRenderChildrenRendersBuiltInChildren(t *testing.T) {
	t.Parallel()

	first := &node.Node{Type: TextTagName, Content: "One"}
	second := &node.Node{Type: TextTagName, Content: "Two"}
	container := &node.Node{Type: "mj-notice", Children: []*node.Node{first, second}}

	var output strings.Builder
	require.NoError(t, RenderChildren(&RenderContext{}, &output, container))

	assert.Equal(t, textHTML("One")+textHTML("Two"), output.String())
	assert.Equal(t, "13px", first.GetAttributeValueDefault("font-size"), "defaults are applied in place")
}

// TestRenderChildrenRendersRegisteredComponents asserts children that are custom
// components render through the registry the render context carries.
func TestRenderChildrenRendersRegisteredComponents(t *testing.T) {
	t.Parallel()

	ctx := contextWith(renderingComponent{tag: "mj-chip", html: `<span class="mj-chip">One</span>`})
	container := &node.Node{
		Type:     "mj-notice",
		Children: []*node.Node{{Type: "mj-chip", Content: "One"}},
	}

	var output strings.Builder
	require.NoError(t, RenderChildren(ctx, &output, container))

	assert.Equal(t, `<span class="mj-chip">One</span>`, output.String())
}

// containerComponent is a custom container that renders its children with
// RenderChildren and nothing else, so the dispatch tests can see it recurse.
type containerComponent struct {
	tag string
}

func (c containerComponent) Name() string { return c.tag }

func (containerComponent) AllowedAttributes() map[string]AttributeValidator { return nil }

func (containerComponent) DefaultAttributes(*RenderContext) map[string]string { return nil }

func (containerComponent) AllowedChildren() []string { return []string{TextTagName} }

func (c containerComponent) Render(ctx *RenderContext, w io.Writer, n *node.Node) error {
	return RenderChildren(ctx, w, n)
}

// TestRenderChildrenRendersNestedContainers asserts a container child that is
// itself a container renders its own children, so containers nest.
func TestRenderChildrenRendersNestedContainers(t *testing.T) {
	t.Parallel()

	ctx := contextWith(containerComponent{tag: "mj-panel"})
	inner := &node.Node{Type: "mj-panel", Children: []*node.Node{{Type: TextTagName, Content: "Inside"}}}
	outer := &node.Node{Type: "mj-notice", Children: []*node.Node{inner}}

	var output strings.Builder
	require.NoError(t, RenderChildren(ctx, &output, outer))

	assert.Equal(t, textHTML("Inside"), output.String())
}

// TestRenderChildrenRendersRawChildren asserts mj-raw children are emitted
// verbatim, like the built-in containers render raw content outside a table body,
// and that this variant writes no raw-block markers: the container's children do
// not sit in a table body, so there is no position to state.
func TestRenderChildrenRendersRawChildren(t *testing.T) {
	t.Parallel()

	container := &node.Node{
		Type: "mj-notice",
		Children: []*node.Node{
			{Type: RawTagName, Content: "\n  <div class=\"raw\">raw</div>\n"},
		},
	}

	var output strings.Builder
	require.NoError(t, RenderChildren(&RenderContext{}, &output, container))

	assert.Equal(t, `<div class="raw">raw</div>`, output.String())
}

// TestRenderChildrenInTableBodyMarksRawChildren asserts the table-body variant
// states the children's position by wrapping an mj-raw child in the raw-block
// markers. They are internal transport that the renderer strips; the inline stage
// styles the block in place with or without them (custom_test.go pins that the two
// helpers produce the same document).
func TestRenderChildrenInTableBodyMarksRawChildren(t *testing.T) {
	t.Parallel()

	container := &node.Node{
		Type: "mj-notice",
		Children: []*node.Node{
			{Type: RawTagName, Content: "\n  <tr><td class=\"raw\">raw</td></tr>\n"},
			{Type: TextTagName, Content: "After"},
		},
	}

	var output strings.Builder
	require.NoError(t, RenderChildrenInTableBody(&RenderContext{}, &output, container))

	assert.Equal(t,
		RawBlockStartMarker+`<tr><td class="raw">raw</td></tr>`+RawBlockEndMarker+textHTML("After"),
		output.String())
}

// TestRenderChildrenEnforcesTheDeclaredChildList asserts RenderChildren checks a
// registered container's own AllowedChildren, not only ValidateDocument, so a
// component rendered without prior validation cannot emit a child it rejected.
func TestRenderChildrenEnforcesTheDeclaredChildList(t *testing.T) {
	t.Parallel()

	ctx := contextWith(containerComponent{tag: "mj-panel"})

	container := &node.Node{Type: "mj-panel"}
	child := &node.Node{Type: ImageTagName}
	child.Parent = container
	container.Children = []*node.Node{child}

	var output strings.Builder
	err := RenderChildren(ctx, &output, container)

	require.Error(t, err)

	var structured *Error
	require.ErrorAs(t, err, &structured)
	assert.Equal(t, KindDisallowedChild, structured.Kind)
	assert.Equal(t, []string{"mj-panel", ImageTagName}, structured.Path)
	assert.Equal(t, ImageTagName, structured.Value)
	assert.Empty(t, output.String(), "nothing of the rejected child is written")
}

// TestRenderChildrenReportsAnUnrenderableChild asserts a child no component can
// render is a broken invariant, reported instead of silently dropped.
// Registration rejects such a declared child before any document renders (see
// componentTable), so this is defensive.
func TestRenderChildrenReportsAnUnrenderableChild(t *testing.T) {
	t.Parallel()

	container := &node.Node{Type: "mj-notice", Children: []*node.Node{{Type: "mj-unknown"}}}

	var output strings.Builder
	err := RenderChildren(&RenderContext{}, &output, container)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no component for child <mj-unknown> in <mj-notice>")
}

// TestRenderChildrenRejectsTheChildrenOfALeaf asserts the helper enforces the
// leaf half of the contract too: a registered component that declares no children
// accepts none, whether or not the document was validated first.
func TestRenderChildrenRejectsTheChildrenOfALeaf(t *testing.T) {
	t.Parallel()

	ctx := contextWith(renderingComponent{tag: "mj-chip", html: "chip"})

	leaf := &node.Node{Type: "mj-chip"}
	child := &node.Node{Type: TextTagName, Content: "Inside"}
	child.Parent = leaf
	leaf.Children = []*node.Node{child}

	var output strings.Builder
	err := RenderChildren(ctx, &output, leaf)

	require.Error(t, err)

	var structured *Error
	require.ErrorAs(t, err, &structured)
	assert.Equal(t, KindDisallowedChild, structured.Kind)
	assert.Equal(t, []string{"mj-chip", TextTagName}, structured.Path)
	assert.Equal(t, TextTagName, structured.Value)
	assert.Contains(t, err.Error(), "allowed children: none")
	assert.Empty(t, output.String())
}
