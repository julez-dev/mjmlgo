package component

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/julez-dev/mjmlgo/v2/node"
	"github.com/stretchr/testify/require"
)

func TestMJMLColumnRender(t *testing.T) {
	t.Parallel()

	t.Run("no-gutter", func(t *testing.T) {
		ctx := &RenderContext{
			ContainerWidth: "600px",
			MJMLStylesheet: make(map[string][]string),
		}

		p := &node.Node{}
		n := &node.Node{
			Parent: p,
		}
		p.Children = append(p.Children, n)
		p.Children = append(p.Children, &node.Node{
			Parent: n,
		})

		var column MJMLColumn
		err := InitComponent(ctx, column, n)
		require.NoError(t, err)

		b := strings.Builder{}
		err = column.Render(ctx, &b, n)
		require.NoError(t, err)

		want := "<div class=\"mj-column-per-50 mj-outlook-group-fix\" style=\"font-size:0px;text-align:left;direction:ltr;display:inline-block;vertical-align:top;width:100%;\">\n<table border=\"0\" cellpadding=\"0\" cellspacing=\"0\" role=\"presentation\" style=\"vertical-align:top;\" width=\"100%\">\n<tbody>\n</tbody>\n</table>\n</div>\n"
		require.Equal(t, want, b.String())
	})

	t.Run("with-gutter", func(t *testing.T) {
		ctx := &RenderContext{
			ContainerWidth: "600px",
			MJMLStylesheet: make(map[string][]string),
		}

		p := &node.Node{}
		n := &node.Node{
			Attributes: []xml.Attr{
				{Name: xml.Name{Local: "padding"}, Value: "11px"},
			},
			Parent: p,
		}
		p.Children = append(p.Children, n)
		p.Children = append(p.Children, &node.Node{
			Parent: n,
		})

		var column MJMLColumn
		err := InitComponent(ctx, column, n)
		require.NoError(t, err)

		b := strings.Builder{}
		err = column.Render(ctx, &b, n)
		require.NoError(t, err)

		want := "<div class=\"mj-column-per-50 mj-outlook-group-fix\" style=\"font-size:0px;text-align:left;direction:ltr;display:inline-block;vertical-align:top;width:100%;\">\n<table border=\"0\" cellpadding=\"0\" cellspacing=\"0\" role=\"presentation\" width=\"100%\">\n<tbody>\n<tr>\n<td style=\"padding:11px;vertical-align:top;\">\n<table border=\"0\" cellpadding=\"0\" cellspacing=\"0\" role=\"presentation\" width=\"100%\">\n<tbody>\n</tbody>\n</table>\n</td>\n</tr>\n</tbody>\n</table>\n</div>\n"
		require.Equal(t, want, b.String())
	})

	t.Run("no-gutter-mobile-width", func(t *testing.T) {
		ctx := &RenderContext{
			ContainerWidth: "600px",
			MJMLStylesheet: make(map[string][]string),
		}

		p := &node.Node{}
		n := &node.Node{
			Attributes: []xml.Attr{
				{Name: xml.Name{Local: "mobileWidth"}, Value: "mobileWidth"},
			},
			Parent: p,
		}
		p.Children = append(p.Children, n)
		p.Children = append(p.Children, &node.Node{
			Parent: n,
		})

		var column MJMLColumn
		err := InitComponent(ctx, column, n)
		require.NoError(t, err)

		b := strings.Builder{}
		err = column.Render(ctx, &b, n)
		require.NoError(t, err)

		want := "<div class=\"mj-column-per-50 mj-outlook-group-fix\" style=\"font-size:0px;text-align:left;direction:ltr;display:inline-block;vertical-align:top;width:50%;\">\n<table border=\"0\" cellpadding=\"0\" cellspacing=\"0\" role=\"presentation\" style=\"vertical-align:top;\" width=\"100%\">\n<tbody>\n</tbody>\n</table>\n</div>\n"
		require.Equal(t, want, b.String())
	})

	t.Run("raw-between-children", func(t *testing.T) {
		ctx := &RenderContext{
			ContainerWidth: "600px",
			MJMLStylesheet: make(map[string][]string),
		}

		parent := &node.Node{}
		n := &node.Node{Parent: parent}
		parent.Children = append(parent.Children, n)

		n.Children = append(n.Children,
			&node.Node{Type: TextTagName, Parent: n, Content: "first text"},
			&node.Node{Type: RawTagName, Parent: n, Content: `<div class="raw-between">raw markup</div>`},
			&node.Node{Type: TextTagName, Parent: n, Content: "second text"},
		)

		var column MJMLColumn
		err := InitComponent(ctx, column, n)
		require.NoError(t, err)

		b := strings.Builder{}
		err = column.Render(ctx, &b, n)
		require.NoError(t, err)

		out := b.String()
		tbody := strings.Index(out, "<tbody>")
		require.NotEqual(t, -1, tbody, "missing <tbody>")

		firstTR := strings.Index(out[tbody:], "<tr>")
		require.NotEqual(t, -1, firstTR, "missing first row")
		firstTR += tbody

		firstClose := strings.Index(out[firstTR:], "</tr>")
		require.NotEqual(t, -1, firstClose, "missing first row close")
		firstClose += firstTR

		raw := strings.Index(out, `<div class="raw-between">raw markup</div>`)
		require.NotEqual(t, -1, raw, "missing raw markup")

		secondTR := strings.Index(out[firstClose:], "<tr>")
		require.NotEqual(t, -1, secondTR, "missing second row")
		secondTR += firstClose

		require.Less(t, tbody, firstTR, "first <tr> must follow <tbody>")
		require.Less(t, firstClose, raw, "raw markup must follow the first row")
		require.Less(t, raw, secondTR, "raw markup must precede the second row")
	})
}

func TestMJMLColumnChildContainerWidth(t *testing.T) {
	t.Parallel()

	n := &node.Node{
		Attributes: []xml.Attr{
			{Name: xml.Name{Local: "width"}, Value: "50%"},
			{Name: xml.Name{Local: "padding"}, Value: "10px 25px"},
			{Name: xml.Name{Local: "inner-border-left"}, Value: "5px solid #000"},
			{Name: xml.Name{Local: "inner-border-right"}, Value: "5px solid #000"},
		},
		Parent: &node.Node{
			Children: []*node.Node{{Type: ColumnTagName}},
		},
	}

	var column MJMLColumn
	width, err := column.childContainerWidth(&RenderContext{ContainerWidth: "600px"}, n)
	require.NoError(t, err)
	// 600 * 50% - (2*25 + 2*5) = 240
	require.Equal(t, "240px", width)
}

func TestMJMLColumnBorderRadiusStyles(t *testing.T) {
	t.Parallel()

	n := &node.Node{
		Attributes: []xml.Attr{
			{Name: xml.Name{Local: "border-radius"}, Value: "50px"},
			{Name: xml.Name{Local: "inner-border-radius"}, Value: "40px"},
			{Name: xml.Name{Local: "padding"}, Value: "50px"},
			{Name: xml.Name{Local: "border"}, Value: "5px solid #000"},
			{Name: xml.Name{Local: "inner-border"}, Value: "5px solid #666"},
		},
		Parent: &node.Node{Children: []*node.Node{{Type: ColumnTagName}}},
	}

	var column MJMLColumn

	outer := column.outerTableStyle(n).InlineString()
	require.Contains(t, outer, "border-radius:50px;")
	require.Contains(t, outer, "border-collapse:separate;")

	inner := column.tableStyle(n).InlineString()
	require.Contains(t, inner, "border-radius:40px;")
	require.Contains(t, inner, "border-collapse:separate;")
	require.Contains(t, inner, "border:5px solid #666;")
}

func TestMJMLColumnGutterKeepsBackground(t *testing.T) {
	t.Parallel()

	ctx := &RenderContext{
		ContainerWidth: "600px",
		MJMLStylesheet: make(map[string][]string),
	}
	parent := &node.Node{Children: []*node.Node{}}
	n := &node.Node{
		Parent: parent,
		Attributes: []xml.Attr{
			{Name: xml.Name{Local: "background-color"}, Value: "#fafafa"},
			{Name: xml.Name{Local: "padding"}, Value: "10px"},
		},
	}
	parent.Children = append(parent.Children, n)

	var column MJMLColumn
	require.NoError(t, InitComponent(ctx, column, n))

	b := strings.Builder{}
	require.NoError(t, column.Render(ctx, &b, n))

	require.Contains(t, b.String(), `background-color:#fafafa;`)
	require.Contains(t, b.String(), `padding:10px;`)
}

func TestColumnClass(t *testing.T) {
	t.Parallel()

	t.Run("percentage", func(t *testing.T) {
		n := &node.Node{
			Attributes: []xml.Attr{},
			Parent: &node.Node{
				Children: []*node.Node{
					{Type: ColumnTagName},
					{Type: ColumnTagName},
				},
			},
		}
		ctx := &RenderContext{
			MJMLStylesheet: make(map[string][]string),
		}

		v, err := MJMLColumn{}.columnClass(ctx, n)
		require.NoError(t, err)
		require.Equal(t, "mj-column-per-50", v)

		m, exists := ctx.MJMLStylesheet["mj-column-per-50"]
		require.True(t, exists, "key mj-column-per-50 should be in map")
		require.Len(t, m, 2)
		require.Equal(t, "width: 50% !important", m[0])
		require.Equal(t, "max-width: 50%", m[1])
	})

	t.Run("px", func(t *testing.T) {
		n := &node.Node{
			Attributes: []xml.Attr{
				{Name: xml.Name{Local: "width"}, Value: "200px"},
			},
			Parent: &node.Node{
				Children: []*node.Node{
					{Type: ColumnTagName},
					{Type: ColumnTagName},
				},
			},
		}
		ctx := &RenderContext{
			MJMLStylesheet: make(map[string][]string),
		}

		v, err := MJMLColumn{}.columnClass(ctx, n)
		require.NoError(t, err)
		require.Equal(t, "mj-column-px-200", v)

		m, exists := ctx.MJMLStylesheet["mj-column-px-200"]
		require.True(t, exists, "key mj-column-px-200 should be in map")
		require.Len(t, m, 2)
		require.Equal(t, "width: 200px !important", m[0])
		require.Equal(t, "max-width: 200px", m[1])
	})
}

func TestColumnMobileWidth(t *testing.T) {
	t.Parallel()

	t.Run(`mobileWidth != "mobileWidth"`, func(t *testing.T) {
		n := &node.Node{
			Attributes: []xml.Attr{
				{Name: xml.Name{Local: "width"}, Value: "200px"},
			},
			Parent: &node.Node{
				Children: []*node.Node{
					{Type: ColumnTagName},
					{Type: ColumnTagName},
				},
			},
		}
		ctx := &RenderContext{
			ContainerWidth: "450px",
		}

		v, err := MJMLColumn{}.mobileWidth(ctx, n)
		require.NoError(t, err)
		require.Equal(t, "100%", v)
	})

	t.Run(`!hasWidth"`, func(t *testing.T) {
		n := &node.Node{
			Attributes: []xml.Attr{
				{Name: xml.Name{Local: "mobileWidth"}, Value: "mobileWidth"},
			},
			Parent: &node.Node{
				Children: []*node.Node{
					{Type: ColumnTagName},
					{Type: ColumnTagName},
				},
			},
		}
		ctx := &RenderContext{
			ContainerWidth: "450px",
		}

		v, err := MJMLColumn{}.mobileWidth(ctx, n)
		require.NoError(t, err)
		require.Equal(t, "50%", v)
	})

	t.Run(`percent-width"`, func(t *testing.T) {
		n := &node.Node{
			Attributes: []xml.Attr{
				{Name: xml.Name{Local: "width"}, Value: "20%"},
				{Name: xml.Name{Local: "mobileWidth"}, Value: "mobileWidth"},
			},
			Parent: &node.Node{
				Children: []*node.Node{
					{Type: ColumnTagName},
					{Type: ColumnTagName},
				},
			},
		}
		ctx := &RenderContext{
			ContainerWidth: "450px",
		}

		v, err := MJMLColumn{}.mobileWidth(ctx, n)
		require.NoError(t, err)
		require.Equal(t, "20%", v)
	})

	t.Run(`px-width"`, func(t *testing.T) {
		n := &node.Node{
			Attributes: []xml.Attr{
				{Name: xml.Name{Local: "width"}, Value: "130px"},
				{Name: xml.Name{Local: "mobileWidth"}, Value: "mobileWidth"},
			},
			Parent: &node.Node{
				Children: []*node.Node{
					{Type: ColumnTagName},
					{Type: ColumnTagName},
				},
			},
		}
		ctx := &RenderContext{
			ContainerWidth: "450px",
		}

		v, err := MJMLColumn{}.mobileWidth(ctx, n)
		require.NoError(t, err)
		require.Equal(t, "28.888889%", v, "MJML 5 keeps six decimals instead of two")
	})
}

// columnSiblings builds a parent holding n children of the given tags and
// returns them, so a test can ask for any column's position in the row.
func columnSiblings(tags ...string) []*node.Node {
	parent := &node.Node{Type: SectionTagName}

	children := make([]*node.Node, len(tags))
	for i, tag := range tags {
		children[i] = &node.Node{Type: tag, Parent: parent}
		parent.Children = append(parent.Children, children[i])
	}

	return children
}

// TestColumnDesktopWidth covers the width a column keeps next to its siblings
// once the section's gutter has been taken out of it.
func TestColumnDesktopWidth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		gutter     string
		widths     []string
		index      int
		wantUnit   string
		wantParsed float64
	}{
		{
			name:       "no gutter leaves a percentage width alone",
			widths:     []string{"", "", ""},
			index:      1,
			wantUnit:   "%",
			wantParsed: 33.333333333333336,
		},
		{
			name:       "no gutter rounds a pixel width",
			widths:     []string{"200.7px", "200.7px"},
			index:      1,
			wantUnit:   "px",
			wantParsed: 200,
		},
		{
			name:       "three equal columns share a pixel gutter as percentages",
			gutter:     "20px",
			widths:     []string{"", "", ""},
			index:      0,
			wantUnit:   "%",
			wantParsed: 31.111111,
		},
		{
			name:       "a pixel gutter becomes pixels for pixel columns",
			gutter:     "3%",
			widths:     []string{"200px", "200px"},
			index:      0,
			wantUnit:   "px",
			wantParsed: 191,
		},
		{
			name:       "leftover pixels go to the leading columns",
			gutter:     "20px",
			widths:     []string{"200px", "200px", "200px"},
			index:      1,
			wantUnit:   "px",
			wantParsed: 187,
		},
		{
			name:       "the last column keeps the floor",
			gutter:     "20px",
			widths:     []string{"200px", "200px", "200px"},
			index:      2,
			wantUnit:   "px",
			wantParsed: 186,
		},
		{
			name:       "a gutter wider than the columns cannot go negative",
			gutter:     "900px",
			widths:     []string{"200px", "200px"},
			index:      0,
			wantUnit:   "px",
			wantParsed: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			columns := columnSiblings(make([]string, len(tt.widths))...)
			for i, width := range tt.widths {
				if width != "" {
					columns[i].SetAttribute("width", width)
				}
			}

			ctx := &RenderContext{ContainerWidth: "600px", Gutter: tt.gutter}

			unit, parsed, err := MJMLColumn{}.desktopWidth(ctx, columns[tt.index])
			require.NoError(t, err)
			require.Equal(t, tt.wantUnit, unit)

			if tt.wantUnit == "px" {
				require.InDelta(t, tt.wantParsed, parsed, 0)
			} else {
				require.InDelta(t, tt.wantParsed, parsed, 1e-9)
			}
		})
	}
}

// TestColumnNormalizeGutterValue covers converting the section's gutter into the
// unit the column itself is measured in.
func TestColumnNormalizeGutterValue(t *testing.T) {
	t.Parallel()

	ctx := &RenderContext{ContainerWidth: "600px", Gutter: "18px"}

	percent, err := MJMLColumn{}.normalizeGutterValue(ctx, "%")
	require.NoError(t, err)
	require.InDelta(t, 3, percent, 1e-9, "18px of 600px is 3%")

	pixels, err := MJMLColumn{}.normalizeGutterValue(ctx, "px")
	require.NoError(t, err)
	require.InDelta(t, 18, pixels, 0)

	ctx.Gutter = "3%"

	pixels, err = MJMLColumn{}.normalizeGutterValue(ctx, "px")
	require.NoError(t, err)
	require.InDelta(t, 18, pixels, 0, "3% of 600px is 18px")

	ctx.Gutter = ""

	zero, err := MJMLColumn{}.normalizeGutterValue(ctx, "px")
	require.NoError(t, err)
	require.Zero(t, zero, "no gutter is no space")
}

// TestColumnGutterPadding covers where a column's share of the gutter lands: on
// the inner edges only, mirrored for rtl, in halves that add up.
func TestColumnGutterPadding(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		gutter    string
		direction string
		want      paddingValues
	}{
		{
			name:   "first column pads its trailing edge",
			gutter: "25px",
			want:   paddingValues{right: 13},
		},
		{
			name:   "last column pads its leading edge",
			gutter: "25px",
			want:   paddingValues{left: 12},
		},
		{
			name:      "rtl swaps the edges",
			gutter:    "25px",
			direction: "rtl",
			want:      paddingValues{left: 13},
		},
		{
			name:      "rtl last column",
			gutter:    "25px",
			direction: "rtl",
			want:      paddingValues{right: 12},
		},
		{
			name:   "a percentage gutter is halved in pixels",
			gutter: "4%",
			want:   paddingValues{right: 12},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			columns := columnSiblings("mj-column", "mj-column")
			column := columns[0]
			if tt.name == "last column pads its leading edge" || tt.name == "rtl last column" {
				column = columns[1]
			}

			ctx := &RenderContext{
				ContainerWidth:     "600px",
				Gutter:             tt.gutter,
				ContainerDirection: tt.direction,
			}

			got, err := MJMLColumn{}.desktopPaddingValues(ctx, column, "px")
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

// TestColumnGutterPaddingPercent asserts the padding is rendered in the column's
// own unit, with six decimals at most.
func TestColumnGutterPaddingPercent(t *testing.T) {
	t.Parallel()

	columns := columnSiblings("mj-column", "mj-column", "mj-column")
	ctx := &RenderContext{ContainerWidth: "600px", Gutter: "20px"}

	first, err := MJMLColumn{}.desktopPadding(ctx, columns[0])
	require.NoError(t, err)
	require.Equal(t, "0% 1.666667% 0% 0%", first)

	middle, err := MJMLColumn{}.desktopPadding(ctx, columns[1])
	require.NoError(t, err)
	require.Equal(t, "0% 1.666667% 0% 1.666667%", middle)

	last, err := MJMLColumn{}.desktopPadding(ctx, columns[2])
	require.NoError(t, err)
	require.Equal(t, "0% 0% 0% 1.666667%", last)
}

// TestColumnMobileGutterPadding asserts stacked columns take the gutter as
// vertical space between them, never on the outer edges.
func TestColumnMobileGutterPadding(t *testing.T) {
	t.Parallel()

	columns := columnSiblings("mj-column", "mj-column")
	ctx := &RenderContext{ContainerWidth: "600px", Gutter: "4%"}

	first, err := MJMLColumn{}.mobilePadding(ctx, columns[0])
	require.NoError(t, err)
	require.Equal(t, "0% 0% 2% 0%", first)

	last, err := MJMLColumn{}.mobilePadding(ctx, columns[1])
	require.NoError(t, err)
	require.Equal(t, "2% 0% 0% 0%", last)
}

// TestColumnGutterClassName asserts the generated class names the rule it
// carries, so two rows that need the same padding can share it.
func TestColumnGutterClassName(t *testing.T) {
	t.Parallel()

	t.Run("percentage gutter of a percentage column", func(t *testing.T) {
		t.Parallel()

		columns := columnSiblings("mj-column", "mj-column", "mj-column")
		ctx := &RenderContext{ContainerWidth: "600px", Gutter: "20px"}

		name, err := MJMLColumn{}.desktopGutterClassName(ctx, columns[2])
		require.NoError(t, err)
		require.Equal(t, "mj-column-gutter-3-3-per-3-333333", name)
	})

	t.Run("pixel gutter of a pixel column", func(t *testing.T) {
		t.Parallel()

		columns := columnSiblings("mj-column", "mj-column")
		columns[0].SetAttribute("width", "200px")

		ctx := &RenderContext{ContainerWidth: "600px", Gutter: "3%"}

		name, err := MJMLColumn{}.desktopGutterClassName(ctx, columns[0])
		require.NoError(t, err)
		require.Equal(t, "mj-column-gutter-2-1-px-18", name)
	})

	t.Run("rtl sections say so", func(t *testing.T) {
		t.Parallel()

		columns := columnSiblings("mj-column", "mj-column")
		ctx := &RenderContext{ContainerWidth: "600px", Gutter: "4%", ContainerDirection: "rtl"}

		name, err := MJMLColumn{}.desktopGutterClassName(ctx, columns[1])
		require.NoError(t, err)
		require.Equal(t, "mj-column-gutter-2-2-per-4-rtl", name)
	})
}

// TestColumnGutterStyles asserts a column carries its desktop share inline, and
// keeps that share inside a group, which does not stack.
func TestColumnGutterStyles(t *testing.T) {
	t.Parallel()

	columns := columnSiblings("mj-column", "mj-column")
	ctx := &RenderContext{ContainerWidth: "600px", Gutter: "4%"}

	mobile, err := MJMLColumn{}.mobileGutterStyle(ctx, columns[0])
	require.NoError(t, err)
	require.Equal(t, inlineStyle{{Property: "padding", Value: "0% 0% 2% 0%"}}, mobile)

	ctx.InGroup = true

	inGroup, err := MJMLColumn{}.mobileGutterStyle(ctx, columns[0])
	require.NoError(t, err)
	require.Equal(t, inlineStyle{{Property: "padding", Value: "0% 2% 0% 0%"}}, inGroup, "a group column keeps desktop padding")

	outlook, err := MJMLColumn{}.outlookGutterStyle(ctx, columns[0])
	require.NoError(t, err)
	require.Equal(t, inlineStyle{{Property: "padding", Value: "0px 12px 0px 0px"}}, outlook, "12px of a 24px gutter")
}

// TestColumnWithoutGutterHasNoPadding asserts a column under a plain section
// gains no padding and no second class.
func TestColumnWithoutGutterHasNoPadding(t *testing.T) {
	t.Parallel()

	columns := columnSiblings("mj-column", "mj-column")
	ctx := &RenderContext{ContainerWidth: "600px"}

	style, err := MJMLColumn{}.mobileGutterStyle(ctx, columns[0])
	require.NoError(t, err)
	require.Empty(t, style)

	outlook, err := MJMLColumn{}.outlookGutterStyle(ctx, columns[0])
	require.NoError(t, err)
	require.Empty(t, outlook)

	name, err := MJMLColumn{}.columnClass(ctx, columns[0])
	require.NoError(t, err)
	require.Equal(t, "mj-column-per-50", name)
}

// TestColumnGutterAddsAClass asserts the gutter class precedes the width class,
// and that the width rule keeps its decimal point while the class name does not.
func TestColumnGutterAddsAClass(t *testing.T) {
	t.Parallel()

	columns := columnSiblings("mj-column", "mj-column", "mj-column")
	ctx := &RenderContext{
		ContainerWidth: "600px",
		Gutter:         "20px",
		MJMLStylesheet: make(map[string][]string),
	}

	name, err := MJMLColumn{}.columnClass(ctx, columns[0])
	require.NoError(t, err)
	require.Equal(t, "mj-column-gutter-3-1-per-3-333333 mj-column-per-31-111111", name)

	require.Equal(t,
		[]string{"width: 31.111111% !important", "max-width: 31.111111%"},
		ctx.MJMLStylesheet["mj-column-per-31-111111"],
		"the rule spells the width with a point")

	require.Equal(t,
		[]string{"padding: 0% 1.666667% 0% 0% !important"},
		ctx.MJMLStylesheet["mj-column-gutter-3-1-per-3-333333"])

	// Group columns carry that padding inline, so they get no second rule.
	ctx.InGroup = true
	ctx.MJMLStylesheet = make(map[string][]string)

	grouped, err := MJMLColumn{}.columnClass(ctx, columns[0])
	require.NoError(t, err)
	require.Equal(t, "mj-column-gutter-3-1-per-3-333333 mj-column-per-31-111111", grouped)
	require.Contains(t, ctx.MJMLStylesheet, "mj-column-per-31-111111")
	require.NotContains(t, ctx.MJMLStylesheet, "mj-column-gutter-3-1-per-3-333333")
}

// TestFormatPaddingValues asserts pixels are rounded and units are trimmed.
func TestFormatPaddingValues(t *testing.T) {
	t.Parallel()

	require.Equal(t, "0px 13px 0px 13px", formatPaddingValues(paddingValues{right: 12.5, left: 12.5}, "px"))
	require.Equal(t, "0% 2% 0% 2%", formatPaddingValues(paddingValues{right: 2, left: 2}, "%"))
	require.Equal(t, "1.5% 0% 0% 0%", formatPaddingValues(paddingValues{top: 1.5}, "%"))
}
