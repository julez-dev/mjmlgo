package mjmlgo

import (
	"bytes"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/julez-dev/mjmlgo/component"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRender(t *testing.T) {
	input, err := os.ReadFile("input_s.mjml")
	require.NoError(t, err)

	out, err := RenderMJML(bytes.NewReader(input))
	require.NoError(t, err)

	os.WriteFile("out.html", []byte(out), 0644)
	_ = out
	//t.Log(out)
}

func TestInlineCSS(t *testing.T) {
	t.Parallel()

	t.Run("simple", func(t *testing.T) {
		const input = `<html><body><p class="p-class">Hello</p></body></html>`
		ctx := &component.RenderContext{
			InlineStyles: []component.Stylesheet{
				{
					Rules: []component.Rule{
						{
							Selectors: ".p-class",
							Declarations: []component.Style{
								{Property: "font-size", Value: "22px", Important: true},
							}},
					},
				},
			},
		}

		var out bytes.Buffer
		err := inlineCSS(ctx, strings.NewReader(input), &out)
		require.NoError(t, err)

		require.Equal(t, "<html><head></head><body><p class=\"p-class\" style=\"font-size:22px;\">Hello</p></body></html>", out.String())
	})

	t.Run("important-override", func(t *testing.T) {
		const input = `<html><body><p class="p-class" style="font-size: 20px;font-weight:300">Hello</p></body></html>`
		ctx := &component.RenderContext{
			InlineStyles: []component.Stylesheet{
				{
					Rules: []component.Rule{
						{
							Selectors: ".p-class",
							Declarations: []component.Style{
								{Property: "font-size", Value: "22px", Important: true},
								{Property: "font-weight", Value: "400"},
							}},
					},
				},
			},
		}

		var out bytes.Buffer
		err := inlineCSS(ctx, strings.NewReader(input), &out)
		require.NoError(t, err)

		randomOrder := []string{
			"<html><head></head><body><p class=\"p-class\" style=\"font-size:22px;font-weight:300;\">Hello</p></body></html>",
			"<html><head></head><body><p class=\"p-class\" style=\"font-weight:300;font-size:22px;\">Hello</p></body></html>",
		}

		assert.True(t, slices.Contains(randomOrder, out.String()), "returned value should be one of the possible")
	})
}
