package mjmltest

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompare(t *testing.T) {
	t.Parallel()

	t.Run("equal", func(t *testing.T) {
		t.Parallel()

		got := parseTestNode(t, `<html><body><p class="a">Hi</p></body></html>`)
		want := parseTestNode(t, `<html><body><p class="a">Hi</p></body></html>`)

		assert.Empty(t, Compare(want, got))
	})

	t.Run("attribute difference", func(t *testing.T) {
		t.Parallel()

		want := parseTestNode(t, `<html><body><div width="600"></div></body></html>`)
		got := parseTestNode(t, `<html><body><div width="800"></div></body></html>`)

		mismatches := Compare(want, got)
		require.Len(t, mismatches, 1)

		assert.Equal(t, "/#document/html[1]/body[1]/div[1]", mismatches[0].Path)
		assert.Equal(t, `attribute "width" mismatch: want "600", got "800"`, mismatches[0].Detail)
	})

	t.Run("missing and unexpected node", func(t *testing.T) {
		t.Parallel()

		want := parseTestNode(t, `<html><body><p>Hi</p><p>Ho</p></body></html>`)
		got := parseTestNode(t, `<html><body><p>Hi</p></body></html>`)

		mismatches := Compare(want, got)
		require.Len(t, mismatches, 1)
		assert.Contains(t, mismatches[0].Detail, "child count mismatch: want 2, got 1")
	})

	t.Run("mismatch limit", func(t *testing.T) {
		t.Parallel()

		var want, got strings.Builder
		want.WriteString("<html><body>")
		got.WriteString("<html><body>")

		for i := range MaxMismatches + 10 {
			want.WriteString(`<div id="`)
			want.WriteString(strconv.Itoa(i))
			want.WriteString(`" class="want"></div>`)
			got.WriteString(`<div id="`)
			got.WriteString(strconv.Itoa(i))
			got.WriteString(`" class="got"></div>`)
		}

		want.WriteString("</body></html>")
		got.WriteString("</body></html>")

		mismatches := Compare(parseTestNode(t, want.String()), parseTestNode(t, got.String()))
		assert.LessOrEqual(t, len(mismatches), MaxMismatches)
	})
}

func TestCompareHTML(t *testing.T) {
	t.Parallel()

	_, err := CompareHTML("not html <", "also not html >")
	require.NoError(t, err, "malformed markup is parsed leniently")
}

func parseTestNode(t *testing.T, document string) *Node {
	t.Helper()

	node, err := ParseHTML(document)
	require.NoError(t, err)

	return node
}
