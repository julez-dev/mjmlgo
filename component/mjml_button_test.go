package component

import (
	"testing"

	"github.com/julez-dev/mjmlgo/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMJMLButtonDefaults asserts the markup a bare button produces: a paragraph,
// not a link, with MJML's default colours, padding and vertical alignment.
func TestMJMLButtonDefaults(t *testing.T) {
	t.Parallel()

	assert.Equal(t, ButtonTagName, MJMLButton{}.Name())

	out, err := renderComponent(t, &RenderContext{ContainerWidth: testContainerWidth}, MJMLButton{}, &node.Node{Content: "Click"})
	require.NoError(t, err)

	assert.Contains(t, out, "<p ", "a button without href is a paragraph")
	assert.NotContains(t, out, "<a ")
	assert.NotContains(t, out, "target=", "link attributes only exist on a link")
	assert.Contains(t, out, ">Click</p>")

	assert.Contains(t, out, `valign="middle"`)
	assert.Contains(t, out, `bgcolor="#414141"`)
	assert.Contains(t, out, "background:#414141")
	assert.Contains(t, out, "border:none")
	assert.Contains(t, out, "border-radius:3px")
	assert.Contains(t, out, "mso-padding-alt:10px 25px")
	assert.Contains(t, out, "padding:10px 25px")
	assert.NotContains(t, out, "width:", "a button without a width sets none")
}

// TestMJMLButtonLink asserts a button with an href becomes an anchor carrying the
// author's link attributes and MJML's default target.
func TestMJMLButtonLink(t *testing.T) {
	t.Parallel()

	n := &node.Node{
		Attributes: attributes(
			"href", "https://example.com/",
			"name", "cta",
			"rel", "nofollow",
			"title", "Open the site",
		),
		Content: "Visit",
	}

	out, err := renderComponent(t, &RenderContext{ContainerWidth: testContainerWidth}, MJMLButton{}, n)
	require.NoError(t, err)

	assert.Contains(t, out, "<a ")
	assert.Contains(t, out, `href="https://example.com/"`)
	assert.Contains(t, out, `name="cta"`)
	assert.Contains(t, out, `rel="nofollow"`)
	assert.Contains(t, out, `title="Open the site"`)
	assert.Contains(t, out, `target="_blank"`)
	assert.Contains(t, out, ">Visit</a>")
}

// TestMJMLButtonWidthAccountsForPaddingAndBorders asserts a pixel width is the
// button's outer width: the anchor's width subtracts the inner padding and both
// borders, so the box stays inside the requested width. A width in percent is
// left to the client.
func TestMJMLButtonWidthAccountsForPaddingAndBorders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		attrs   []string
		want    string
		notWant string
	}{
		{
			name:  "inner padding is subtracted",
			attrs: []string{"width", "200px", "inner-padding", "10px 20px"},
			want:  "border-radius:3px;width:160px;",
		},
		{
			name: "the directional borders win over the shorthand",
			attrs: []string{
				"width", "200px",
				"inner-padding", "10px 20px",
				"border", "5px solid black",
				"border-left", "1px solid black",
				"border-right", "12px solid black",
			},
			want: "border-radius:3px;width:147px;",
		},
		{
			name:  "the container padding does not shrink the anchor",
			attrs: []string{"width", "200px", "padding", "100px", "inner-padding", "10px 20px"},
			want:  "border-radius:3px;width:160px;",
		},
		{
			name:    "a percentage width is not converted",
			attrs:   []string{"width", "50%"},
			notWant: "border-radius:3px;width:",
		},
		{
			name:    "no width sets none",
			attrs:   nil,
			notWant: "border-radius:3px;width:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			out, err := renderComponent(t, &RenderContext{ContainerWidth: testContainerWidth}, MJMLButton{}, &node.Node{
				Attributes: attributes(tt.attrs...),
				Content:    "Click",
			})
			require.NoError(t, err)

			if tt.want != "" {
				assert.Contains(t, out, tt.want)
			}

			if tt.notWant != "" {
				assert.NotContains(t, out, tt.notWant)
			}
		})
	}
}

// TestMJMLButtonInnerPaddingDrivesBothPaddings asserts inner-padding is used for
// the padding the client sees and for the Outlook fallback on the cell.
func TestMJMLButtonInnerPaddingDrivesBothPaddings(t *testing.T) {
	t.Parallel()

	out, err := renderComponent(t, &RenderContext{ContainerWidth: testContainerWidth}, MJMLButton{}, &node.Node{
		Attributes: attributes("inner-padding", "12px 34px"),
		Content:    "Click",
	})
	require.NoError(t, err)

	assert.Contains(t, out, "padding:12px 34px")
	assert.Contains(t, out, "mso-padding-alt:12px 34px")
}

// TestMJMLButtonBackground asserts the background colour reaches the cell's
// bgcolor attribute and a hex shortcut is expanded.
func TestMJMLButtonBackground(t *testing.T) {
	t.Parallel()

	t.Run("a hex shortcut is expanded", func(t *testing.T) {
		t.Parallel()

		out, err := renderComponent(t, &RenderContext{ContainerWidth: testContainerWidth}, MJMLButton{}, &node.Node{
			Attributes: attributes("background-color", "#f00"),
			Content:    "Click",
		})
		require.NoError(t, err)

		assert.Contains(t, out, `bgcolor="#ff0000"`)
		assert.Contains(t, out, "background:#ff0000")
	})
}

// TestMJMLButtonVerticalAlign asserts the author's vertical alignment reaches
// the cell's valign.
func TestMJMLButtonVerticalAlign(t *testing.T) {
	t.Parallel()

	out, err := renderComponent(t, &RenderContext{ContainerWidth: testContainerWidth}, MJMLButton{}, &node.Node{
		Attributes: attributes("vertical-align", "top"),
		Content:    "Click",
	})
	require.NoError(t, err)

	assert.Contains(t, out, `valign="top"`)
}

// TestMJMLButtonValidatesAuthorValues asserts InitComponent checks the author's
// values before defaults are applied, so a bad value cannot hide behind the
// button's own default for the same attribute.
func TestMJMLButtonValidatesAuthorValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		attrs    []string
		badAttr  string
		badValue string
	}{
		{name: "unknown alignment", attrs: []string{"align", "justify"}, badAttr: "align", badValue: "justify"},
		{name: "width in em", attrs: []string{"width", "10em"}, badAttr: "width", badValue: "10em"},
		{name: "unknown colour", attrs: []string{"background-color", "chartreuseish"}, badAttr: "background-color", badValue: "chartreuseish"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			n := &node.Node{Attributes: attributes(tt.attrs...)}

			requireInvalidValue(t, InitComponent(&RenderContext{}, MJMLButton{}, n), tt.badAttr, tt.badValue)
		})
	}

	t.Run("a valid author value is kept over the default", func(t *testing.T) {
		t.Parallel()

		n := &node.Node{Attributes: attributes("background-color", "#123456")}
		require.NoError(t, InitComponent(&RenderContext{}, MJMLButton{}, n))
		requireAttribute(t, n, "background-color", "#123456")
	})
}

// TestMJMLButtonReportsABrokenContainerWidth asserts a context that cannot be
// measured is an error instead of a silently wrong width.
func TestMJMLButtonReportsABrokenContainerWidth(t *testing.T) {
	t.Parallel()

	n := &node.Node{Attributes: attributes("width", "200px"), Content: "Click"}

	_, err := renderComponent(t, &RenderContext{ContainerWidth: "wide"}, MJMLButton{}, n)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid container width")
}
