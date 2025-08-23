package mjmlgo

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/julez-dev/mjmlgo/component"
	"github.com/stretchr/testify/require"
)

func TestRender(t *testing.T) {
	input, err := os.ReadFile("input_big.mjml")
	require.NoError(t, err)

	out, err := RenderMJML(bytes.NewReader(input))
	require.NoError(t, err)

	os.WriteFile("out.html", []byte(out), 0644)
	_ = out
	//t.Log(out)
}

func TestInlineCSS(t *testing.T) {
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

	require.Equal(t, "<html><head></head><body><p class=\"p-class\" style=\"font-size: 22px !important;\">Hello</p></body></html>", out.String())
}
