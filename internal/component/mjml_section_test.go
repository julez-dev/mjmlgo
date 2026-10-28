package component

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/julez-dev/mjmlgo/internal/node"
	"github.com/stretchr/testify/require"
)

func TestMJMLSection(t *testing.T) {
	t.Parallel()

	t.Run("simple", func(t *testing.T) {
		ctx := &RenderContext{
			ContainerWidth: "600px",
			MJMLStylesheet: make(map[string][]string),
		}

		parent := &node.Node{
			Type: "mj-body",
		}

		n := &node.Node{
			Type:   "mj-section",
			Parent: parent,
		}
		parent.Children = append(parent.Children, n)
		n.Children = append(n.Children, &node.Node{
			Type:   "mj-section",
			Parent: n,
		})

		var section MJMLSection
		err := InitComponent(ctx, section, n)
		require.NoError(t, err)

		b := strings.Builder{}
		err = section.Render(ctx, &b, n)
		require.NoError(t, err)

		want := "<!--[if mso | IE]><table align=\"center\" border=\"0\" cellpadding=\"0\" cellspacing=\"0\" role=\"presentation\" style=\"width:600px;\" width=\"600\">\n<tr>\n<td style=\"line-height:0px;font-size:0px;mso-line-height-rule:exactly;\">\n<![endif]-->\n<div style=\"margin:0px auto;max-width:600px;\">\n<table align=\"center\" border=\"0\" cellpadding=\"0\" cellspacing=\"0\" role=\"presentation\" style=\"width:100%;\">\n<tbody>\n<tr>\n<td style=\"direction:ltr;font-size:0px;padding:20px 0;text-align:center;\">\n<!--[if mso | IE]><table role=\"presentation\" border=\"0\" cellpadding=\"0\" cellspacing=\"0\"><![endif]-->\n<!--[if mso | IE]><tr><![endif]-->\n<!--[if mso | IE]></tr><![endif]-->\n<!--[if mso | IE]></table><![endif]-->\n</td>\n</tr>\n</tbody>\n</table>\n</div>\n<!--[if mso | IE]></td></tr></table><![endif]-->"
		require.Equal(t, want, b.String())
	})

	t.Run("full-width", func(t *testing.T) {
		ctx := &RenderContext{
			ContainerWidth: "600px",
			MJMLStylesheet: make(map[string][]string),
		}

		parent := &node.Node{
			Type: "mj-body",
		}

		n := &node.Node{
			Type:   "mj-section",
			Parent: parent,
			Attributes: []xml.Attr{
				{Name: xml.Name{Local: "full-width"}, Value: "full-width"},
			},
		}
		parent.Children = append(parent.Children, n)
		n.Children = append(n.Children, &node.Node{
			Type:   "mj-section",
			Parent: n,
		})

		var section MJMLSection
		err := InitComponent(ctx, section, n)
		require.NoError(t, err)

		b := strings.Builder{}
		err = section.Render(ctx, &b, n)
		require.NoError(t, err)

		want := "<table align=\"center\" border=\"0\" cellpadding=\"0\" cellspacing=\"0\" role=\"presentation\" style=\"width:100%;\">\n<tbody>\n<tr>\n<td><!--[if mso | IE]><table align=\"center\" border=\"0\" cellpadding=\"0\" cellspacing=\"0\" role=\"presentation\" style=\"width:600px;\" width=\"600\">\n<tr>\n<td style=\"line-height:0px;font-size:0px;mso-line-height-rule:exactly;\">\n<![endif]-->\n<div style=\"margin:0px auto;max-width:600px;\">\n<table align=\"center\" border=\"0\" cellpadding=\"0\" cellspacing=\"0\" role=\"presentation\" style=\"width:100%;\">\n<tbody>\n<tr>\n<td style=\"direction:ltr;font-size:0px;padding:20px 0;text-align:center;\">\n<!--[if mso | IE]><table role=\"presentation\" border=\"0\" cellpadding=\"0\" cellspacing=\"0\"><![endif]-->\n<!--[if mso | IE]><tr><![endif]-->\n<!--[if mso | IE]></tr><![endif]-->\n<!--[if mso | IE]></table><![endif]-->\n</td>\n</tr>\n</tbody>\n</table>\n</div>\n<!--[if mso | IE]></td></tr></table><![endif]--></td>\n</tr>\n</tbody>\n</table>\n"
		require.Equal(t, want, b.String())
	})

	t.Run("is-wrapper", func(t *testing.T) {
		ctx := &RenderContext{
			ContainerWidth: "600px",
			MJMLStylesheet: make(map[string][]string),
		}

		parent := &node.Node{
			Type: "mj-body",
		}

		n := &node.Node{
			Type:       "mj-section",
			Parent:     parent,
			Attributes: []xml.Attr{},
		}
		parent.Children = append(parent.Children, n)
		n.Children = append(n.Children, &node.Node{
			Type:   "mj-section",
			Parent: n,
		})

		var section MJMLSection
		section.IsWrapper = true
		err := InitComponent(ctx, section, n)
		require.NoError(t, err)

		b := strings.Builder{}
		err = section.Render(ctx, &b, n)
		require.NoError(t, err)

		want := "<!--[if mso | IE]><table align=\"center\" border=\"0\" cellpadding=\"0\" cellspacing=\"0\" role=\"presentation\" style=\"width:600px;\" width=\"600\">\n<tr>\n<td style=\"line-height:0px;font-size:0px;mso-line-height-rule:exactly;\">\n<![endif]-->\n<div style=\"margin:0px auto;max-width:600px;\">\n<table align=\"center\" border=\"0\" cellpadding=\"0\" cellspacing=\"0\" role=\"presentation\" style=\"width:100%;\">\n<tbody>\n<tr>\n<td style=\"direction:ltr;font-size:0px;padding:20px 0;text-align:center;\">\n<!--[if mso | IE]><table role=\"presentation\" border=\"0\" cellpadding=\"0\" cellspacing=\"0\"><![endif]-->\n<!--[if mso | IE]><tr><td width=\"600px\"><![endif]--><!--[if mso | IE]><table align=\"center\" border=\"0\" cellpadding=\"0\" cellspacing=\"0\" role=\"presentation\" style=\"width:600px;\" width=\"600\">\n<tr>\n<td style=\"line-height:0px;font-size:0px;mso-line-height-rule:exactly;\">\n<![endif]-->\n<div style=\"margin:0px auto;max-width:600px;\">\n<table align=\"center\" border=\"0\" cellpadding=\"0\" cellspacing=\"0\" role=\"presentation\" style=\"width:100%;\">\n<tbody>\n<tr>\n<td style=\"direction:ltr;font-size:0px;padding:20px 0;text-align:center;\">\n<!--[if mso | IE]><table role=\"presentation\" border=\"0\" cellpadding=\"0\" cellspacing=\"0\"><![endif]-->\n<!--[if mso | IE]><tr><![endif]-->\n<!--[if mso | IE]></tr><![endif]-->\n<!--[if mso | IE]></table><![endif]-->\n</td>\n</tr>\n</tbody>\n</table>\n</div>\n<!--[if mso | IE]></td></tr></table><![endif]--><!--[if mso | IE]></td></tr><![endif]--><!--[if mso | IE]></table><![endif]-->\n</td>\n</tr>\n</tbody>\n</table>\n</div>\n<!--[if mso | IE]></td></tr></table><![endif]-->"
		require.Equal(t, want, b.String())
	})
}

func TestMJMLSectionOutlookSeats(t *testing.T) {
	t.Parallel()

	newSection := func(t *testing.T, children ...*node.Node) (*RenderContext, *node.Node) {
		t.Helper()

		ctx := &RenderContext{
			ContainerWidth: "600px",
			MJMLStylesheet: make(map[string][]string),
		}
		parent := &node.Node{Type: "mj-body"}
		n := &node.Node{Type: "mj-section", Parent: parent}
		parent.Children = append(parent.Children, n)

		for _, child := range children {
			child.Parent = n
			n.Children = append(n.Children, child)
		}

		return ctx, n
	}

	t.Run("column seat uses the column width", func(t *testing.T) {
		ctx, n := newSection(t,
			&node.Node{Type: ColumnTagName, Attributes: []xml.Attr{{Name: xml.Name{Local: "width"}, Value: "30%"}}},
			&node.Node{Type: ColumnTagName, Attributes: []xml.Attr{{Name: xml.Name{Local: "width"}, Value: "70%"}}},
		)

		var section MJMLSection
		require.NoError(t, InitComponent(ctx, section, n))

		b := strings.Builder{}
		require.NoError(t, section.Render(ctx, &b, n))

		out := b.String()
		require.Contains(t, out, `style="vertical-align:top;width:180px;"`)
		require.Contains(t, out, `style="vertical-align:top;width:420px;"`)
	})

	t.Run("group gets its own Outlook seat", func(t *testing.T) {
		group := &node.Node{
			Type: GroupTagName,
			Children: []*node.Node{
				{Type: ColumnTagName, Parent: &node.Node{}},
			},
		}
		group.Children[0].Parent = group

		ctx, n := newSection(t, group)

		var section MJMLSection
		require.NoError(t, InitComponent(ctx, section, n))

		b := strings.Builder{}
		require.NoError(t, section.Render(ctx, &b, n))

		require.Contains(t, b.String(), `<!--[if mso | IE]><td style="width:600px;"><![endif]-->`)
	})
}

func TestMJMLSectionBorderRadius(t *testing.T) {
	t.Parallel()

	ctx := &RenderContext{
		ContainerWidth: "600px",
		MJMLStylesheet: make(map[string][]string),
	}
	parent := &node.Node{Type: "mj-body"}
	n := &node.Node{
		Type:       "mj-section",
		Parent:     parent,
		Attributes: []xml.Attr{{Name: xml.Name{Local: "border-radius"}, Value: "10px"}},
	}
	parent.Children = append(parent.Children, n)

	var section MJMLSection
	require.NoError(t, InitComponent(ctx, section, n))

	b := strings.Builder{}
	require.NoError(t, section.Render(ctx, &b, n))

	out := b.String()
	require.Contains(t, out, `overflow:hidden;`)
	require.Contains(t, out, `border-collapse:separate;`)
	require.NotContains(t, out, `<table align="center" border="0" cellpadding="0" cellspacing="0" role="presentation" style="border-radius:10px;width:100%;">`)
	require.Contains(t, out, `border-radius:10px;direction:ltr;`)
}

func TestMJMLSectionWrapperGap(t *testing.T) {
	t.Parallel()

	wrapper := &node.Node{
		Type:       WrapperTagName,
		Attributes: []xml.Attr{{Name: xml.Name{Local: "gap"}, Value: "20px"}},
	}
	first := &node.Node{Type: SectionTagName, Parent: wrapper}
	second := &node.Node{
		Type:   SectionTagName,
		Parent: wrapper,
		Attributes: []xml.Attr{
			{Name: xml.Name{Local: "background-color"}, Value: "#cccccc"},
		},
	}
	wrapper.Children = []*node.Node{first, second}

	require.Equal(t, "20px", sectionGap(first))
	require.True(t, isFirstSibling(first))
	require.False(t, isFirstSibling(second))

	ctx := &RenderContext{
		ContainerWidth: "600px",
		MJMLStylesheet: make(map[string][]string),
	}

	var section MJMLSection
	require.NoError(t, InitComponent(ctx, section, second))

	b := strings.Builder{}
	require.NoError(t, section.Render(ctx, &b, second))

	out := b.String()
	require.Contains(t, out, `style="width:600px;padding-top:20px;"`)
	require.Contains(t, out, `margin-top:20px;`)
	require.NotContains(t, out, `bgcolor="#cccccc"`)
}

func TestMJMLWrapperChildOutlookClass(t *testing.T) {
	t.Parallel()

	ctx := &RenderContext{
		ContainerWidth: "600px",
		MJMLStylesheet: make(map[string][]string),
	}
	parent := &node.Node{Type: "mj-body", Children: []*node.Node{}}
	wrapper := &node.Node{
		Type:       WrapperTagName,
		Parent:     parent,
		Attributes: []xml.Attr{{Name: xml.Name{Local: "gap"}, Value: "20px"}},
	}
	parent.Children = append(parent.Children, wrapper)

	first := &node.Node{
		Type:       SectionTagName,
		Parent:     wrapper,
		Attributes: []xml.Attr{{Name: xml.Name{Local: "css-class"}, Value: "my-section"}},
	}
	wrapper.Children = append(wrapper.Children, first)

	var section MJMLSection
	section.IsWrapper = true
	require.NoError(t, InitComponent(ctx, section, wrapper))

	b := strings.Builder{}
	require.NoError(t, section.Render(ctx, &b, wrapper))

	require.Contains(t, b.String(), `class="my-section-outlook"`)
}
