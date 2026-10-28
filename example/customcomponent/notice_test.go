package customcomponent

import (
	"strings"
	"testing"

	"github.com/julez-dev/mjmlgo/component"
	"github.com/julez-dev/mjmlgo/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// noticeHTML is the exact markup Notice writes for its children: the wrapper
// with the notice's defaults, and the children rendered by their own components
// inside it.
func noticeHTML(children string) string {
	return `<table role="presentation" border="0" cellpadding="0" cellspacing="0" class="mj-notice mj-notice-fluid">` +
		`<tbody><tr><td align="left" style="background-color:#fef3c7;border:1px solid #f59e0b;border-radius:6px;` +
		`font-family:Helvetica,Arial,sans-serif;font-size:13px;line-height:1;padding:12px 16px;">` +
		children + `</td></tr></tbody></table>`
}

// renderNotice runs the public contract the renderer uses: InitComponent applies
// the defaults, then Render writes the wrapper around the children it renders
// with component.RenderChildren.
func renderNotice(t *testing.T, ctx *component.RenderContext, pairs ...string) (string, *node.Node) {
	t.Helper()

	n := &node.Node{Type: NoticeTagName, Attributes: attributes(pairs...)}
	n.Children = []*node.Node{
		{Type: component.TextTagName, Content: "One"},
		{Type: component.TextTagName, Content: "Two"},
	}

	require.NoError(t, component.InitComponent(ctx, Notice{}, n))

	var output strings.Builder
	require.NoError(t, Notice{}.Render(ctx, &output, n))

	return output.String(), n
}

// TestNoticeWrapsItsRenderedChildren asserts the container's exact HTML: its own
// wrapper around the children, each rendered with its own component's defaults.
func TestNoticeWrapsItsRenderedChildren(t *testing.T) {
	t.Parallel()

	rendered, _ := renderNotice(t, &component.RenderContext{})

	assert.Equal(t, noticeHTML(
		`<div style="font-family:Ubuntu, Helvetica, Arial, sans-serif;font-size:13px;line-height:1;`+
			`text-align:left;color:#000000;">One</div>`+"\n"+
			`<div style="font-family:Ubuntu, Helvetica, Arial, sans-serif;font-size:13px;line-height:1;`+
			`text-align:left;color:#000000;">Two</div>`+"\n",
	), rendered)
}

// TestNoticeAuthorAttributesWinOverItsDefaults asserts an attribute the author
// wrote survives and the component's defaults fill in the rest. The document's
// mj-attributes/mj-class cascade runs in the renderer, before Render, and is
// covered by the root package's tests.
func TestNoticeAuthorAttributesWinOverItsDefaults(t *testing.T) {
	t.Parallel()

	rendered, _ := renderNotice(t, &component.RenderContext{}, "background-color", "#111827")

	assert.Contains(t, rendered, "background-color:#111827;border:1px solid #f59e0b;")
}

// TestNoticeContributesHeadStyles asserts the three head mechanisms are
// registered: a named component block, a breakpoint media-query class and the
// fluid-on-mobile class.
func TestNoticeContributesHeadStyles(t *testing.T) {
	t.Parallel()

	ctx := &component.RenderContext{}
	renderNotice(t, ctx)

	require.Len(t, ctx.ComponentHeadStyles, 1)
	assert.Equal(t, NoticeTagName, ctx.ComponentHeadStyles[0].Name)
	assert.Equal(t,
		"    @media only screen and (min-width:480px) {\n        .mj-notice { padding: 16px !important; }\n    }\n",
		ctx.ComponentHeadStyles[0].Style("480px"))

	assert.Equal(t, []string{"mj-notice-inline"}, ctx.MJMLStyleOrder)
	assert.Equal(t, []string{"display: inline-block !important"}, ctx.MJMLStylesheet["mj-notice-inline"])
}

// TestNoticeRejectsAnUndeclaredChild asserts Notice's child list is enforced by
// the renderer's document validation, with the element path in the error.
func TestNoticeRejectsAnUndeclaredChild(t *testing.T) {
	t.Parallel()

	ctx := &component.RenderContext{CustomComponents: map[string]component.Component{NoticeTagName: Notice{}}}
	panel := &node.Node{Type: NoticeTagName}
	image := &node.Node{Type: component.ImageTagName}
	panel.Children = []*node.Node{image}
	image.Parent = panel

	root := &node.Node{Type: component.MJMLTagName}
	body := &node.Node{Type: component.BodyTagName}
	section := &node.Node{Type: component.SectionTagName}
	column := &node.Node{Type: component.ColumnTagName}
	root.Children = []*node.Node{body}
	body.Children = []*node.Node{section}
	section.Children = []*node.Node{column}
	column.Children = []*node.Node{panel}
	panel.Parent = column
	section.Parent = body
	body.Parent = root
	column.Parent = section

	err := component.ValidateDocument(ctx, root)
	require.Error(t, err)

	var structured *component.Error
	require.ErrorAs(t, err, &structured)
	assert.Equal(t, component.KindDisallowedChild, structured.Kind)
	assert.Equal(t, []string{"mj-body", "mj-section", "mj-column", NoticeTagName, component.ImageTagName}, structured.Path)
	assert.Equal(t, component.ImageTagName, structured.Value)
}
