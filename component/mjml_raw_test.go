package component

import (
	"strings"
	"testing"

	"github.com/julez-dev/mjmlgo/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMJMLRawRender asserts raw content is written as the author wrote it: the
// surrounding whitespace is trimmed, the payload itself is untouched, and
// markup that looks like CDATA delimiters is not interpreted.
func TestMJMLRawRender(t *testing.T) {
	t.Parallel()

	assert.Equal(t, RawTagName, MJMLRaw{}.Name())

	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "surrounding whitespace is trimmed",
			content: "\n\t  <div class=\"raw\">kept  spacing</div>\n",
			want:    `<div class="raw">kept  spacing</div>`,
		},
		{
			name:    "cdata delimiters in the payload survive",
			content: "<![CDATA[<div>not markup</div>]]>",
			want:    "<![CDATA[<div>not markup</div>]]>",
		},
		{
			name:    "a closing cdata sequence inside the payload survives",
			content: "<div>a ]]> b</div>",
			want:    "<div>a ]]> b</div>",
		},
		{
			name:    "entities are not decoded or re-encoded",
			content: "<div>a &amp; b</div>",
			want:    "<div>a &amp; b</div>",
		},
		{
			name:    "empty content renders nothing",
			content: "  \n",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var out strings.Builder
			require.NoError(t, MJMLRaw{}.Render(&RenderContext{}, &out, &node.Node{Content: tt.content}))
			assert.Equal(t, tt.want, out.String())
		})
	}
}

// TestRenderRawChildMarksTheTableBodyVariant asserts the two raw-child helpers
// differ only by the markers: bare content outside a table body, marked content
// directly inside one.
func TestRenderRawChildMarksTheTableBodyVariant(t *testing.T) {
	t.Parallel()

	const content = "\n  <tr><td>raw row</td></tr>\n"
	n := &node.Node{Type: RawTagName, Content: content}

	t.Run("outside a table body", func(t *testing.T) {
		t.Parallel()

		var out strings.Builder
		require.NoError(t, renderRawChild(&RenderContext{}, &out, n))
		assert.Equal(t, "<tr><td>raw row</td></tr>", out.String())
	})

	t.Run("inside a table body", func(t *testing.T) {
		t.Parallel()

		var out strings.Builder
		require.NoError(t, renderRawChildInTableBody(&RenderContext{}, &out, n))
		assert.Equal(t,
			RawBlockStartMarker+"<tr><td>raw row</td></tr>"+RawBlockEndMarker,
			out.String())
	})
}

// TestRenderRawChildValidatesTheNode asserts raw children go through the same
// attribute validation as any other component, so an invalid position is caught
// during rendering too, not only by document validation.
func TestRenderRawChildValidatesTheNode(t *testing.T) {
	t.Parallel()

	t.Run("a valid position renders", func(t *testing.T) {
		t.Parallel()

		n := &node.Node{Type: RawTagName, Attributes: attributes("position", "file-start"), Content: "<div>x</div>"}

		var out strings.Builder
		require.NoError(t, renderRawChild(&RenderContext{}, &out, n))
		assert.Equal(t, "<div>x</div>", out.String())
	})

	t.Run("an unknown position is rejected", func(t *testing.T) {
		t.Parallel()

		n := &node.Node{Type: RawTagName, Attributes: attributes("position", "body-end"), Content: "<div>x</div>"}

		var out strings.Builder
		err := renderRawChild(&RenderContext{}, &out, n)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not one of")
		assert.Empty(t, out.String(), "nothing is written for a rejected raw child")
	})
}

// TestStripRawBlockMarkers asserts marker removal is complete and idempotent, and
// that html without markers is returned unchanged.
func TestStripRawBlockMarkers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "both markers are removed",
			in:   RawBlockStartMarker + "<tr><td>raw</td></tr>" + RawBlockEndMarker,
			want: "<tr><td>raw</td></tr>",
		},
		{
			name: "every occurrence is removed",
			in: "a" + RawBlockStartMarker + "b" + RawBlockEndMarker +
				RawBlockStartMarker + "c" + RawBlockEndMarker + "d",
			want: "abcd",
		},
		{
			name: "a lone start marker is removed",
			in:   "a" + RawBlockStartMarker + "b",
			want: "ab",
		},
		{
			name: "a lone end marker is removed",
			in:   "a" + RawBlockEndMarker + "b",
			want: "ab",
		},
		{
			name: "html without markers is unchanged",
			in:   "<table><tbody><tr><td>raw</td></tr></tbody></table>",
			want: "<table><tbody><tr><td>raw</td></tr></tbody></table>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := StripRawBlockMarkers(tt.in)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.want, StripRawBlockMarkers(got), "stripping is idempotent")
		})
	}
}
