package component

import (
	"strings"
	"testing"

	"github.com/julez-dev/mjmlgo/v2/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// socialWithChildren builds an mj-social node whose children are the given ones,
// with parent links filled in as the parser would.
func socialWithChildren(attrs []string, children ...*node.Node) *node.Node {
	social := &node.Node{Type: SocialTagName, Attributes: attributes(attrs...)}
	social.Children = children

	for _, child := range children {
		child.Parent = social
	}

	return social
}

// TestMJMLSocialDefaults asserts the social block's defaults are applied and an
// author value keeps its place.
func TestMJMLSocialDefaults(t *testing.T) {
	t.Parallel()

	assert.Equal(t, SocialTagName, MJMLSocial{}.Name())

	n := &node.Node{Type: SocialTagName}
	require.NoError(t, InitComponent(&RenderContext{}, MJMLSocial{}, n))

	requireAttribute(t, n, "mode", "horizontal")
	requireAttribute(t, n, "align", "center")
	requireAttribute(t, n, "icon-size", "20px")
	requireAttribute(t, n, "color", "#333333")
	requireAttribute(t, n, "font-size", "13px")
	requireAttribute(t, n, "line-height", "22px")
	requireAttribute(t, n, "padding", "10px 25px")

	author := &node.Node{Type: SocialTagName, Attributes: attributes("mode", "vertical", "icon-size", "32px")}
	require.NoError(t, InitComponent(&RenderContext{}, MJMLSocial{}, author))
	requireAttribute(t, author, "mode", "vertical")
	requireAttribute(t, author, "icon-size", "32px")
}

// TestMJMLSocialHorizontalLayout asserts the horizontal mode wraps the per-element
// tables in Outlook cells and renders raw children between them without the
// table-body markers, because that layout has no table body of its own.
func TestMJMLSocialHorizontalLayout(t *testing.T) {
	t.Parallel()

	first := &node.Node{Type: SocialElementTagName, Attributes: attributes("name", "facebook")}
	raw := &node.Node{Type: RawTagName, Content: "\n  <div class=\"separator\">raw</div>\n"}
	second := &node.Node{Type: SocialElementTagName, Attributes: attributes("name", "twitter")}
	notSocial := &node.Node{Type: TextTagName, Content: "not a social element"}

	social := socialWithChildren([]string{"align", "right"}, first, raw, second, notSocial)

	out, err := renderComponent(t, &RenderContext{}, MJMLSocial{}, social)
	require.NoError(t, err)

	assert.True(t, strings.HasPrefix(out, `<!--[if mso | IE]><table align="right"`), "the layout opens with an Outlook table")
	assert.True(t, strings.HasSuffix(out, "<!--[if mso | IE]></tr></table><![endif]-->"), "and closes the Outlook table")
	assert.Equal(t, 2, strings.Count(out, "<!--[if mso | IE]><td><![endif]-->"), "each social element gets its own Outlook cell")

	assert.Contains(t, out, `<div class="separator">raw</div>`, "raw children render at their position")
	assert.NotContains(t, out, RawBlockStartMarker)
	assert.NotContains(t, out, RawBlockEndMarker)

	assert.NotContains(t, out, "not a social element", "other children are ignored, not rendered")

	elementTable := `<table align="right" border="0" cellpadding="0" cellspacing="0" role="presentation" style="float:none;display:inline-table;">`
	assert.Equal(t, 2, strings.Count(out, elementTable), "each element table inherits the block's alignment")
}

// TestMJMLSocialVerticalLayout asserts the vertical mode renders one table body
// and protects raw children with the table-body markers, since they sit directly
// inside that body.
func TestMJMLSocialVerticalLayout(t *testing.T) {
	t.Parallel()

	raw := &node.Node{Type: RawTagName, Content: "<tr><td class=\"raw-row\">raw</td></tr>"}
	element := &node.Node{Type: SocialElementTagName, Attributes: attributes("name", "github")}
	social := socialWithChildren([]string{"mode", "vertical", "align", "right"}, raw, element)

	out, err := renderComponent(t, &RenderContext{}, MJMLSocial{}, social)
	require.NoError(t, err)

	assert.True(t, strings.HasPrefix(out, "<table "), "vertical mode is a plain table")
	assert.Contains(t, out, "<tbody>")
	assert.True(t, strings.HasSuffix(out, "</tbody></table>"))
	assert.NotContains(t, out, "<!--[if mso | IE]><td>", "the vertical layout has no per-element Outlook cells")
	assert.NotContains(t, out, `align="right"`, "the vertical layout ignores the block alignment")

	assert.Contains(t, out, "<tbody>"+RawBlockStartMarker+`<tr><td class="raw-row">raw</td></tr>`+RawBlockEndMarker)
}

// TestMJMLSocialPropagatesIconSizing asserts icon-size and icon-height set on the
// social block reach an element that does not set them, so one block sizes all
// its icons.
func TestMJMLSocialPropagatesIconSizing(t *testing.T) {
	t.Parallel()

	element := &node.Node{Type: SocialElementTagName, Attributes: attributes("name", "facebook")}
	social := socialWithChildren([]string{"icon-size", "30px", "icon-height", "40px"}, element)

	out, err := renderComponent(t, &RenderContext{}, MJMLSocial{}, social)
	require.NoError(t, err)

	requireAttribute(t, element, "icon-size", "30px")
	requireAttribute(t, element, "icon-height", "40px")

	assert.Contains(t, out, `width="30"`, "the img keeps only the numeric width")
	assert.Contains(t, out, "width:30px;")
	assert.Contains(t, out, "height:40px;", "icon-height wins over the square icon-size fallback")
}

// TestMJMLSocialPropagatesTextStyle asserts the block's shared colours, font and
// radius reach its elements, and that inner-padding becomes the element's
// padding.
func TestMJMLSocialPropagatesTextStyle(t *testing.T) {
	t.Parallel()

	element := &node.Node{Type: SocialElementTagName, Attributes: attributes("name", "twitter"), Content: "Twitter"}
	social := socialWithChildren([]string{
		"icon-size", "24px",
		"color", "#123456",
		"font-size", "16px",
		"border-radius", "8px",
		"inner-padding", "2px 6px",
	}, element)

	out, err := renderComponent(t, &RenderContext{}, MJMLSocial{}, social)
	require.NoError(t, err)

	requireAttribute(t, element, "padding", "2px 6px")
	assert.Contains(t, out, "padding:2px 6px")
	assert.Contains(t, out, "color:#123456")
	assert.Contains(t, out, "font-size:16px")
	assert.Contains(t, out, "border-radius:8px")
}

// TestMJMLSocialReportsInvalidElementAttributes asserts the block validates each
// element while rendering, so a bad element value surfaces as an error instead of
// silently rendering with a default.
func TestMJMLSocialReportsInvalidElementAttributes(t *testing.T) {
	t.Parallel()

	// align is not one of the attributes the block propagates, so the element's
	// own invalid value reaches InitComponent instead of a parent default.
	element := &node.Node{Type: SocialElementTagName, Attributes: attributes("name", "facebook", "align", "justify")}
	social := socialWithChildren(nil, element)

	_, err := renderComponent(t, &RenderContext{}, MJMLSocial{}, social)
	requireInvalidValue(t, err, "align", "justify")
}
