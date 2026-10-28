package customcomponent

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/julez-dev/mjmlgo/component"
	"github.com/julez-dev/mjmlgo/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func attributes(pairs ...string) []xml.Attr {
	attrs := make([]xml.Attr, 0, len(pairs)/2)

	for i := 0; i+1 < len(pairs); i += 2 {
		attrs = append(attrs, xml.Attr{Name: xml.Name{Local: pairs[i]}, Value: pairs[i+1]})
	}

	return attrs
}

// renderBadge runs the public contract the renderer uses: InitComponent applies
// the defaults, then Render writes the HTML.
func renderBadge(t *testing.T, pairs ...string) string {
	t.Helper()

	n := &node.Node{Type: BadgeTagName, Attributes: attributes(pairs...)}
	require.NoError(t, component.InitComponent(&component.RenderContext{}, Badge{}, n))

	var output strings.Builder
	require.NoError(t, Badge{}.Render(&component.RenderContext{}, &output, n))

	return output.String()
}

func TestBadgeRendersKnownHTML(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		`<table role="presentation" border="0" cellpadding="0" cellspacing="0">`+
			`<tbody><tr><td align="center" style="background-color:#fde68a;border-radius:4px;color:#111827;`+
			`font-family:Helvetica,Arial,sans-serif;font-size:13px;line-height:1;padding:8px 12px;">`+
			`New</td></tr></tbody></table>`,
		renderBadge(t, "text", "New", "background-color", "#fde68a", "color", "#111827"),
	)
}

func TestBadgeAppliesItsDefaults(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		`<table role="presentation" border="0" cellpadding="0" cellspacing="0">`+
			`<tbody><tr><td align="center" style="background-color:#1f2937;border-radius:4px;color:#ffffff;`+
			`font-family:Helvetica,Arial,sans-serif;font-size:13px;line-height:1;padding:8px 12px;">`+
			`Default</td></tr></tbody></table>`,
		renderBadge(t, "text", "Default"),
	)
}

func TestBadgeRendersUniversalCssClassAndEscapesText(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		`<table role="presentation" border="0" cellpadding="0" cellspacing="0" class="promo">`+
			`<tbody><tr><td align="center" style="background-color:#1f2937;border-radius:4px;color:#ffffff;`+
			`font-family:Helvetica,Arial,sans-serif;font-size:13px;line-height:1;padding:8px 12px;">`+
			`A &amp; B</td></tr></tbody></table>`,
		renderBadge(t, "text", "A & B", "css-class", "promo"),
	)
}

func TestBadgeRejectsEmptyText(t *testing.T) {
	t.Parallel()

	n := &node.Node{Type: BadgeTagName, Attributes: attributes("text", "  ")}

	err := component.InitComponent(&component.RenderContext{}, Badge{}, n)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "badge text must not be empty")
}
