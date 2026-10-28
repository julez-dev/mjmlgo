package component

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUsedFonts(t *testing.T) {
	t.Parallel()

	t.Run("builtin fonts only when used, in registry order", func(t *testing.T) {
		t.Parallel()

		ctx := &RenderContext{}
		ctx.ensureFontRegistry()

		body := `<div style="font-family:Ubuntu, Helvetica, Arial, sans-serif;"></div>` +
			`<div style="font-family:Lato, sans-serif;"></div>`

		require.Equal(t, []string{"Lato", "Ubuntu"}, ctx.usedFonts(body, nil))
	})

	t.Run("unused mj-font is not emitted", func(t *testing.T) {
		t.Parallel()

		ctx := &RenderContext{}
		ctx.ensureFontRegistry()
		ctx.addFont("Custom", "https://example.com/custom.css")

		body := `<div style="font-family:Ubuntu, Helvetica, Arial, sans-serif;"></div>`

		require.Equal(t, []string{"Ubuntu"}, ctx.usedFonts(body, nil))
	})

	t.Run("used mj-font is emitted after the builtin fonts", func(t *testing.T) {
		t.Parallel()

		ctx := &RenderContext{}
		ctx.ensureFontRegistry()
		ctx.addFont("Custom", "https://example.com/custom.css")

		body := `<div style="font-family:Custom, sans-serif;"></div>` +
			`<div style="font-family:Ubuntu, Helvetica, Arial, sans-serif;"></div>`

		require.Equal(t, []string{"Ubuntu", "Custom"}, ctx.usedFonts(body, nil))
	})

	t.Run("font usage in inline stylesheets is detected", func(t *testing.T) {
		t.Parallel()

		ctx := &RenderContext{}
		ctx.ensureFontRegistry()

		inline := []Stylesheet{{
			Rules: []Rule{{
				Selectors:    ".custom",
				Declarations: []Style{{Property: "font-family", Value: "Roboto, sans-serif"}},
			}},
		}}

		require.Equal(t, []string{"Roboto"}, ctx.usedFonts("", inline))
	})
}
