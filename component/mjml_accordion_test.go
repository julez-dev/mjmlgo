package component

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/julez-dev/mjmlgo/v2/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMJMLAccordion(t *testing.T) {
	t.Parallel()

	t.Run("renders elements with inherited defaults and component style", func(t *testing.T) {
		t.Parallel()

		ctx := &RenderContext{Breakpoint: "480px", MJMLStylesheet: make(map[string][]string)}

		accordion := &node.Node{Type: AccordionTagName}
		element := &node.Node{Type: AccordionElementTagName, Parent: accordion}
		title := &node.Node{
			Type:    AccordionTitleTagName,
			Parent:  element,
			Content: "Question",
			Attributes: []xml.Attr{
				{Name: xml.Name{Local: "font-weight"}, Value: "bold"},
			},
		}
		text := &node.Node{Type: AccordionTextTagName, Parent: element, Content: "Answer"}
		element.Children = []*node.Node{title, text}
		accordion.Children = []*node.Node{element}

		var component MJMLAccordion
		require.NoError(t, InitComponent(ctx, component, accordion))
		require.NoError(t, InitComponent(ctx, MJMLAccordionElement{}, element))
		require.NoError(t, InitComponent(ctx, MJMLAccordionTitle{}, title))
		require.NoError(t, InitComponent(ctx, MJMLAccordionText{}, text))

		var out strings.Builder
		require.NoError(t, component.Render(ctx, &out, accordion))
		html := out.String()

		assert.Contains(t, html, `class="mj-accordion"`)
		assert.Contains(t, html, `border:2px solid black`)
		assert.Contains(t, html, `font-family:Ubuntu, Helvetica, Arial, sans-serif`)
		assert.Contains(t, html, `class="mj-accordion-title"`)
		assert.Contains(t, html, `class="mj-accordion-ico"`)
		assert.Contains(t, html, `class="mj-accordion-content"`)
		assert.Contains(t, html, `font-weight:bold`)
		assert.Contains(t, html, `>Question</td>`)
		assert.Contains(t, html, `>Answer</td>`)

		require.Len(t, ctx.ComponentHeadStyles, 1)
		assert.Equal(t, "mj-accordion", ctx.ComponentHeadStyles[0].Name)
		assert.Contains(t, ctx.ComponentHeadStyles[0].Style("480px"), "input.mj-accordion-checkbox")
	})

	t.Run("creates missing title and text children", func(t *testing.T) {
		t.Parallel()

		ctx := &RenderContext{Breakpoint: "480px", MJMLStylesheet: make(map[string][]string)}
		accordion := &node.Node{Type: AccordionTagName}
		element := &node.Node{Type: AccordionElementTagName, Parent: accordion}
		accordion.Children = []*node.Node{element}

		var component MJMLAccordion
		require.NoError(t, InitComponent(ctx, component, accordion))
		require.NoError(t, InitComponent(ctx, MJMLAccordionElement{}, element))

		var out strings.Builder
		require.NoError(t, component.Render(ctx, &out, accordion))
		html := out.String()

		assert.Equal(t, 1, strings.Count(html, `class="mj-accordion-title"`))
		assert.Equal(t, 1, strings.Count(html, `class="mj-accordion-content"`))
	})

	t.Run("icon position left puts icons before the title", func(t *testing.T) {
		t.Parallel()

		ctx := &RenderContext{Breakpoint: "480px", MJMLStylesheet: make(map[string][]string)}
		accordion := &node.Node{
			Type: AccordionTagName,
			Attributes: []xml.Attr{
				{Name: xml.Name{Local: "icon-position"}, Value: "left"},
			},
		}
		element := &node.Node{Type: AccordionElementTagName, Parent: accordion}
		element.Children = []*node.Node{
			{Type: AccordionTitleTagName, Parent: element, Content: "Question"},
			{Type: AccordionTextTagName, Parent: element, Content: "Answer"},
		}
		accordion.Children = []*node.Node{element}

		var component MJMLAccordion
		require.NoError(t, InitComponent(ctx, component, accordion))
		require.NoError(t, InitComponent(ctx, MJMLAccordionElement{}, element))
		for _, child := range element.Children {
			switch child.Type {
			case AccordionTitleTagName:
				require.NoError(t, InitComponent(ctx, MJMLAccordionTitle{}, child))
			case AccordionTextTagName:
				require.NoError(t, InitComponent(ctx, MJMLAccordionText{}, child))
			}
		}

		var out strings.Builder
		require.NoError(t, component.Render(ctx, &out, accordion))
		html := out.String()

		iconIndex := strings.Index(html, `class="mj-accordion-ico"`)
		titleIndex := strings.Index(html, `>Question</td>`)
		require.NotEqual(t, -1, iconIndex)
		require.NotEqual(t, -1, titleIndex)
		assert.Less(t, iconIndex, titleIndex)
	})
}

func TestMJMLAccordionFontFamilyResolution(t *testing.T) {
	t.Parallel()

	newContext := func() *RenderContext {
		return &RenderContext{
			Breakpoint:     "480px",
			MJMLStylesheet: make(map[string][]string),
			AttributeDefaults: map[string]AttributeMap{
				AccordionTitleTagName: {"font-family": "Roboto, sans-serif"},
			},
		}
	}

	t.Run("mj-attributes default does not count as explicit", func(t *testing.T) {
		t.Parallel()

		ctx := newContext()
		ctx.elementFontFamily = "Georgia, serif"

		title := &node.Node{
			Type: AccordionTitleTagName,
			Attributes: []xml.Attr{
				{Name: xml.Name{Local: "font-family"}, Value: "Roboto, sans-serif"},
			},
		}

		assert.Equal(t, "Georgia, serif", resolveAccordionFontFamily(ctx, title))
	})

	t.Run("explicit font-family wins", func(t *testing.T) {
		t.Parallel()

		ctx := newContext()
		ctx.elementFontFamily = "Georgia, serif"

		title := &node.Node{
			Type: AccordionTitleTagName,
			Attributes: []xml.Attr{
				{Name: xml.Name{Local: "font-family"}, Value: "monospace"},
			},
		}

		assert.Equal(t, "monospace", resolveAccordionFontFamily(ctx, title))
	})

	t.Run("falls back to accordion font", func(t *testing.T) {
		t.Parallel()

		ctx := newContext()
		ctx.accordionFontFamily = "serif"

		title := &node.Node{Type: AccordionTitleTagName}

		assert.Equal(t, "serif", resolveAccordionFontFamily(ctx, title))
	})
}
