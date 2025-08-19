package component

import (
	"encoding/xml"
	"fmt"
	"testing"

	"github.com/julez-dev/mjmlgo/node"
	"github.com/stretchr/testify/require"
)

func TestInlineStyle(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		s := inlineStyle{}
		require.Equal(t, "", s.InlineString())
	})

	t.Run("important-mix", func(t *testing.T) {
		s := inlineStyle{
			{Property: "color", Value: "red", Important: true},
			{Property: "background-color", Value: "blue"},
		}
		require.Equal(t, "color:red!important;background-color:blue;", s.InlineString())
	})
}

func TestInlineAttribute(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		s := inlineAttributes{}
		require.Equal(t, "", s.InlineString())
	})

	t.Run("multiple", func(t *testing.T) {
		s := inlineAttributes{
			"class":       "my-class",
			"cellpadding": "10",
		}
		require.Equal(t, `class="my-class" cellpadding="10"`, s.InlineString())
	})
}

func TestIsPercentage(t *testing.T) {
	t.Parallel()

	t.Run("string", func(t *testing.T) {
		require.False(t, isPercentage("string"))
	})

	t.Run("number", func(t *testing.T) {
		require.False(t, isPercentage("10"))
	})

	t.Run("px", func(t *testing.T) {
		require.False(t, isPercentage("10"))
	})

	t.Run("percentage", func(t *testing.T) {
		require.True(t, isPercentage("100%"))
	})
}

func TestParseBackgroundPosition(t *testing.T) {
	t.Parallel()

	t.Run("empty-center", func(t *testing.T) {
		v1, v2 := parseBackgroundPosition("")
		require.Equal(t, "center", v1)
		require.Equal(t, "center", v2)
	})

	t.Run("one-value", func(t *testing.T) {
		v1, v2 := parseBackgroundPosition("bottom")
		require.Equal(t, "center", v1)
		require.Equal(t, "bottom", v2)
	})

	t.Run("two-values", func(t *testing.T) {
		v1, v2 := parseBackgroundPosition("center right")
		require.Equal(t, "right", v1)
		require.Equal(t, "center", v2)
	})

	t.Run("too-many-values", func(t *testing.T) {
		v1, v2 := parseBackgroundPosition("bottom top center")
		require.Equal(t, "center", v1)
		require.Equal(t, "top", v2)
	})
}

func TestGetBackgroundPosition(t *testing.T) {
	t.Parallel()

	t.Run("only-background-position", func(t *testing.T) {
		n := &node.Node{
			Attributes: []xml.Attr{
				{Name: xml.Name{Local: "background-position"}, Value: "center top"},
			},
		}

		v1, v2 := getBackgroundPosition(n)
		require.Equal(t, "center", v1)
		require.Equal(t, "top", v2)
	})

	t.Run("background-position-x", func(t *testing.T) {
		n := &node.Node{
			Attributes: []xml.Attr{
				{Name: xml.Name{Local: "background-position"}, Value: "center top"},
				{Name: xml.Name{Local: "background-position-x"}, Value: "left"},
			},
		}

		v1, v2 := getBackgroundPosition(n)
		require.Equal(t, "left", v1)
		require.Equal(t, "top", v2)
	})

	t.Run("background-position-y", func(t *testing.T) {
		n := &node.Node{
			Attributes: []xml.Attr{
				{Name: xml.Name{Local: "background-position"}, Value: "left center"},
				{Name: xml.Name{Local: "background-position-y"}, Value: "top"},
			},
		}

		v1, v2 := getBackgroundPosition(n)
		require.Equal(t, "left", v1)
		require.Equal(t, "top", v2)
	})
}

func TestGetBackgroundString(t *testing.T) {
	t.Parallel()

	t.Run("only-background-position", func(t *testing.T) {
		n := &node.Node{
			Attributes: []xml.Attr{
				{Name: xml.Name{Local: "background-position"}, Value: "center top"},
			},
		}

		v1 := getBackgroundString(n)
		require.Equal(t, "center top", v1)
	})
}

func TestAddSuffixToClasses(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		class := addSuffixToClasses("", "outlook")
		require.Equal(t, "", class)
	})

	t.Run("filled", func(t *testing.T) {
		class := addSuffixToClasses("bold column", "outlook")
		require.Equal(t, "bold-outlook column-outlook", class)
	})
}

func TestParseWidth(t *testing.T) {
	t.Parallel()

	t.Run("default-unit-pixel", func(t *testing.T) {
		v, u, err := parseWidth("100")
		require.NoError(t, err)
		require.Equal(t, float64(100), v)
		require.Equal(t, "px", u)
	})

	t.Run("explicit-px", func(t *testing.T) {
		v, u, err := parseWidth("100px")
		require.NoError(t, err)
		require.Equal(t, float64(100), v)
		require.Equal(t, "px", u)
	})

	t.Run("float-truncate", func(t *testing.T) {
		v, u, err := parseWidth("10.9px")
		require.NoError(t, err)
		require.Equal(t, float64(10), v)
		require.Equal(t, "px", u)
	})

	t.Run("explicit-percentage", func(t *testing.T) {
		v, u, err := parseWidth("10.9%")
		require.NoError(t, err)
		require.Equal(t, float64(10), v)
		require.Equal(t, "%", u)
	})
}

func TestRemoveNonNumeric(t *testing.T) {
	t.Parallel()

	v := removeNonNumeric("100px")
	require.Equal(t, "100", v)
}

func TestShorthandParser(t *testing.T) {
	t.Parallel()

	t.Run("example-left", func(t *testing.T) {
		v, err := shorthandParser("10", "left")
		require.NoError(t, err)
		require.Equal(t, 10, v)
	})

	t.Run("example-empty", func(t *testing.T) {
		v, err := shorthandParser("10 0", "right")
		require.NoError(t, err)
		require.Equal(t, 0, v)
	})

	t.Run("with-unit", func(t *testing.T) {
		v, err := shorthandParser("10px", "right")
		require.NoError(t, err)
		require.Equal(t, 10, v)
	})
}

func TestGetShorthandAttrValue(t *testing.T) {
	t.Parallel()

	t.Run("padding-left", func(t *testing.T) {
		n := &node.Node{
			Attributes: []xml.Attr{
				{Name: xml.Name{Local: "padding-left"}, Value: "10px"},
			},
		}
		v, err := getShorthandAttrValue(n, "padding", "left")
		require.NoError(t, err)
		require.Equal(t, 10, v)
	})

	t.Run("padding", func(t *testing.T) {
		n := &node.Node{
			Attributes: []xml.Attr{
				{Name: xml.Name{Local: "padding"}, Value: "10px"},
			},
		}
		v, err := getShorthandAttrValue(n, "padding", "top")
		require.NoError(t, err)
		require.Equal(t, 10, v)
	})

	t.Run("empty", func(t *testing.T) {
		n := &node.Node{
			Attributes: []xml.Attr{
				{Name: xml.Name{Local: "padding"}, Value: ""},
			},
		}
		v, err := getShorthandAttrValue(n, "padding", "top")
		require.NoError(t, err)
		require.Equal(t, 0, v)
	})
}

func TestBorderParser(t *testing.T) {
	t.Parallel()

	t.Run("1px solid black", func(t *testing.T) {
		v := borderParser("1px solid black")
		require.Equal(t, 1, v)
	})

	t.Run("10px", func(t *testing.T) {
		v := borderParser("10px")
		require.Equal(t, 10, v)
	})

	t.Run("22", func(t *testing.T) {
		v := borderParser("22")
		require.Equal(t, 22, v)
	})

	t.Run("0", func(t *testing.T) {
		v := borderParser("0")
		require.Equal(t, 0, v)
	})
}

func TestGetShorthandBorderValue(t *testing.T) {
	t.Parallel()

	t.Run("border-right", func(t *testing.T) {
		n := &node.Node{
			Attributes: []xml.Attr{
				{Name: xml.Name{Local: "border-right"}, Value: "10px"},
			},
		}
		v := getShorthandBorderValue(n, "right", "border")
		require.Equal(t, 10, v)
	})

	t.Run("empty-attribute", func(t *testing.T) {
		n := &node.Node{
			Attributes: []xml.Attr{
				{Name: xml.Name{Local: "border-right"}, Value: "10px"},
			},
		}
		v := getShorthandBorderValue(n, "right", "")
		require.Equal(t, 10, v)
	})

	t.Run("border-attribute", func(t *testing.T) {
		n := &node.Node{
			Attributes: []xml.Attr{
				{Name: xml.Name{Local: "border"}, Value: "10px"},
			},
		}
		v := getShorthandBorderValue(n, "", "")
		require.Equal(t, 10, v)
	})
}

func TestGetBoxWidths(t *testing.T) {
	t.Parallel()

	n := &node.Node{
		Attributes: []xml.Attr{
			{Name: xml.Name{Local: "border-left"}, Value: "10px solid black"},
			{Name: xml.Name{Local: "padding"}, Value: "20px"},
		},
	}
	ctx := &RenderContext{ContainerWidth: "600px"}

	v, err := getBoxWidths(ctx, n)
	require.NoError(t, err)

	expected := map[string]int{
		"totalWidth": 600,
		"borders":    10,
		"paddings":   40,
		"box":        550,
	}

	require.Equal(t, fmt.Sprint(expected), fmt.Sprint(v))
}

func TestContentWidth(t *testing.T) {
	t.Parallel()

	t.Run("width-not-set", func(t *testing.T) {
		n := &node.Node{
			Attributes: []xml.Attr{
				{Name: xml.Name{Local: "border-left"}, Value: "10px solid black"},
				{Name: xml.Name{Local: "padding"}, Value: "20px"},
			},
		}
		ctx := &RenderContext{ContainerWidth: "600px"}

		v, err := getContentWidth(ctx, n)
		require.NoError(t, err)
		require.Equal(t, 550, v)
	})

	t.Run("width-set-smaller", func(t *testing.T) {
		n := &node.Node{
			Attributes: []xml.Attr{
				{Name: xml.Name{Local: "width"}, Value: "300px"},
				{Name: xml.Name{Local: "border-left"}, Value: "10px solid black"},
				{Name: xml.Name{Local: "padding"}, Value: "20px"},
			},
		}
		ctx := &RenderContext{ContainerWidth: "600px"}

		v, err := getContentWidth(ctx, n)
		require.NoError(t, err)
		require.Equal(t, 300, v)
	})
}

func TestNonRawSiblings(t *testing.T) {
	t.Parallel()

	n := &node.Node{
		Parent: &node.Node{
			Children: []*node.Node{
				{Type: RawTagName},
				{Type: ColumnTagName},
				{Type: ColumnTagName},
			},
		},
	}

	require.Equal(t, 2, len(nonRawSiblings(n)))
}

func TestGetParsedWidth(t *testing.T) {
	t.Parallel()

	t.Run("width-set", func(t *testing.T) {
		n := &node.Node{
			Attributes: []xml.Attr{
				{Name: xml.Name{Local: "width"}, Value: "200px"},
			},
			Parent: &node.Node{
				Children: []*node.Node{
					{Type: RawTagName},
					{Type: ColumnTagName},
					{Type: ColumnTagName},
				},
			},
		}

		u, v, err := getParsedWidth(n)
		require.NoError(t, err)
		require.Equal(t, "px", u)
		require.Equal(t, float64(200), v)
	})

	t.Run("width-not-set", func(t *testing.T) {
		n := &node.Node{
			Attributes: []xml.Attr{},
			Parent: &node.Node{
				Children: []*node.Node{
					{Type: RawTagName},
					{Type: ColumnTagName},
					{Type: ColumnTagName},
				},
			},
		}

		u, v, err := getParsedWidth(n)
		require.NoError(t, err)
		require.Equal(t, "%", u)
		require.Equal(t, float64(50), v)
	})
}

func TestGetColumnClass(t *testing.T) {
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

		v, err := getColumnClass(ctx, n)
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

		v, err := getColumnClass(ctx, n)
		require.NoError(t, err)
		require.Equal(t, "mj-column-px-200", v)

		m, exists := ctx.MJMLStylesheet["mj-column-px-200"]
		require.True(t, exists, "key mj-column-px-200 should be in map")
		require.Len(t, m, 2)
		require.Equal(t, "width: 200px !important", m[0])
		require.Equal(t, "max-width: 200px", m[1])
	})
}

func TestGetMobileWidth(t *testing.T) {
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

		v, err := getMobileWidth(ctx, n)
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

		v, err := getMobileWidth(ctx, n)
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

		v, err := getMobileWidth(ctx, n)
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

		v, err := getMobileWidth(ctx, n)
		require.NoError(t, err)
		require.Equal(t, "28.89%", v)
	})
}

func TestMakeBackgroundString(t *testing.T) {
	t.Parallel()
	v := makeBackgroundString([]string{"content-box", "", "radial-gradient(crimson, skyblue)"})
	require.Equal(t, "content-box radial-gradient(crimson, skyblue)", v)
}
