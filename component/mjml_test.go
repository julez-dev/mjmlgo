package component

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/julez-dev/mjmlgo/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func attribute(name, value string) xml.Attr {
	return xml.Attr{Name: xml.Name{Local: name}, Value: value}
}

func attributes(pairs ...string) []xml.Attr {
	attrs := make([]xml.Attr, 0, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		attrs = append(attrs, attribute(pairs[i], pairs[i+1]))
	}

	return attrs
}

// testContainerWidth is the document width most component tests measure
// against.
const testContainerWidth = "600px"

// renderComponent initialises n with comp and renders it, so tests read the
// markup and the error from one call.
func renderComponent(t *testing.T, ctx *RenderContext, comp Component, n *node.Node) (string, error) {
	t.Helper()

	if err := InitComponent(ctx, comp, n); err != nil {
		return "", err
	}

	var out strings.Builder
	err := comp.Render(ctx, &out, n)

	return out.String(), err
}

// requireInvalidValue asserts err is a structured invalid-value failure for the
// given attribute and value, so tests pin the classification a caller branches
// on instead of the human-readable message.
func requireInvalidValue(t *testing.T, err error, attribute, value string) {
	t.Helper()

	var structured *Error
	require.ErrorAs(t, err, &structured)
	require.Equal(t, KindInvalidValue, structured.Kind)
	require.Equal(t, attribute, structured.Attribute)
	require.Equal(t, value, structured.Value)
}

func headWithAttributes(children ...*node.Node) *node.Node {
	return &node.Node{
		Type: HeadTagName,
		Children: []*node.Node{{
			Type:     AttributesTagName,
			Children: children,
		}},
	}
}

func applyHeadAttributes(t *testing.T, head, body *node.Node) *RenderContext {
	t.Helper()

	ctx := &RenderContext{}
	var mjml MJML
	require.NoError(t, mjml.preparseHeadMetaValues(ctx, head))

	mjml.setAttributeDefaults(ctx, body, "")

	return ctx
}

func requireAttribute(t *testing.T, n *node.Node, name, want string) {
	t.Helper()

	got, has := n.GetAttributeValue(name)
	require.True(t, has, "attribute %q is missing", name)
	require.Equal(t, want, got, "attribute %q", name)
}

func TestSetAttributeDefaultsPrecedence(t *testing.T) {
	t.Parallel()

	t.Run("mj-all then per-tag then element", func(t *testing.T) {
		t.Parallel()

		head := headWithAttributes(
			&node.Node{
				Type:       AllTagName,
				Attributes: attributes("font-family", "Arial", "color", "#111111"),
			},
			&node.Node{
				Type:       TextTagName,
				Attributes: attributes("font-size", "15px", "color", "#222222"),
			},
		)

		text := &node.Node{
			Type:       TextTagName,
			Attributes: attributes("color", "#ff0000"),
		}
		section := &node.Node{
			Type: SectionTagName,
			Children: []*node.Node{{
				Type:     ColumnTagName,
				Children: []*node.Node{text},
			}},
		}
		body := &node.Node{Type: BodyTagName, Children: []*node.Node{section}}

		applyHeadAttributes(t, head, body)

		// mj-all applies to every tag; the per-tag default wins over it; the
		// attribute written on the element wins over both.
		requireAttribute(t, section, "font-family", "Arial")
		requireAttribute(t, section, "color", "#111111")
		requireAttribute(t, text, "font-family", "Arial")
		requireAttribute(t, text, "font-size", "15px")
		requireAttribute(t, text, "color", "#ff0000")
	})

	t.Run("mj-class wins over per-tag defaults", func(t *testing.T) {
		t.Parallel()

		head := headWithAttributes(
			&node.Node{
				Type:       TextTagName,
				Attributes: attributes("color", "#222222", "font-size", "12px"),
			},
			&node.Node{
				Type:       ClassTagName,
				Attributes: attributes("name", "primary", "color", "#ff0000"),
			},
		)

		text := &node.Node{Type: TextTagName, Attributes: attributes("mj-class", "primary")}
		body := &node.Node{Type: BodyTagName, Children: []*node.Node{text}}

		applyHeadAttributes(t, head, body)

		requireAttribute(t, text, "color", "#ff0000")
		requireAttribute(t, text, "font-size", "12px")
	})

	t.Run("chained classes merge css-class and later classes win", func(t *testing.T) {
		t.Parallel()

		head := headWithAttributes(
			&node.Node{
				Type:       ClassTagName,
				Attributes: attributes("name", "primary", "font-size", "10px", "css-class", "primary-class"),
			},
			&node.Node{
				Type:       ClassTagName,
				Attributes: attributes("name", "spaced", "font-size", "20px", "css-class", "spaced-class"),
			},
		)

		text := &node.Node{Type: TextTagName, Attributes: attributes("mj-class", "primary spaced")}
		body := &node.Node{Type: BodyTagName, Children: []*node.Node{text}}

		applyHeadAttributes(t, head, body)

		requireAttribute(t, text, "font-size", "20px")
		requireAttribute(t, text, "css-class", "primary-class spaced-class")
	})

	t.Run("element css-class overrides chained class css-class", func(t *testing.T) {
		t.Parallel()

		head := headWithAttributes(
			&node.Node{
				Type:       ClassTagName,
				Attributes: attributes("name", "primary", "css-class", "primary-class"),
			},
			&node.Node{
				Type:       ClassTagName,
				Attributes: attributes("name", "spaced", "css-class", "spaced-class"),
			},
		)

		text := &node.Node{
			Type:       TextTagName,
			Attributes: attributes("mj-class", "primary spaced", "css-class", "explicit-class"),
		}
		body := &node.Node{Type: BodyTagName, Children: []*node.Node{text}}

		applyHeadAttributes(t, head, body)

		requireAttribute(t, text, "css-class", "explicit-class")
	})

	t.Run("class defaults apply to descendants of a classed element", func(t *testing.T) {
		t.Parallel()

		head := headWithAttributes(
			&node.Node{
				Type:       ClassTagName,
				Attributes: attributes("name", "layout"),
				Children: []*node.Node{
					{
						Type:       TextTagName,
						Attributes: attributes("line-height", "30px", "color", "#123456"),
					},
					{
						Type:       SectionTagName,
						Attributes: attributes("padding", "99px"),
					},
				},
			},
		)

		text := &node.Node{Type: TextTagName}
		section := &node.Node{
			Type:       SectionTagName,
			Attributes: attributes("mj-class", "layout"),
			Children: []*node.Node{{
				Type:     ColumnTagName,
				Children: []*node.Node{text},
			}},
		}
		body := &node.Node{Type: BodyTagName, Children: []*node.Node{section}}

		applyHeadAttributes(t, head, body)

		// The class defaults are for descendants, not for the classed element.
		_, has := section.GetAttributeValue("padding")
		require.False(t, has, "section should not receive its own class's nested defaults")

		requireAttribute(t, text, "line-height", "30px")
		requireAttribute(t, text, "color", "#123456")
	})

	t.Run("nested class defaults win over the class's own attributes", func(t *testing.T) {
		t.Parallel()

		head := headWithAttributes(
			&node.Node{
				Type:       ClassTagName,
				Attributes: attributes("name", "layout", "color", "#aaaaaa"),
				Children: []*node.Node{{
					Type:       TextTagName,
					Attributes: attributes("color", "#123456"),
				}},
			},
		)

		text := &node.Node{Type: TextTagName}
		section := &node.Node{
			Type:       SectionTagName,
			Attributes: attributes("mj-class", "layout"),
			Children:   []*node.Node{text},
		}
		body := &node.Node{Type: BodyTagName, Children: []*node.Node{section}}

		applyHeadAttributes(t, head, body)

		requireAttribute(t, section, "color", "#aaaaaa")
		requireAttribute(t, text, "color", "#123456")
	})
}

func TestPreparseHTMLAttributes(t *testing.T) {
	t.Parallel()

	head := &node.Node{
		Type: HeadTagName,
		Children: []*node.Node{{
			Type: HTMLAttributesTagName,
			Children: []*node.Node{
				{
					Type:       SelectorTagName,
					Attributes: attributes("path", ".text div"),
					Children: []*node.Node{{
						Type:       HTMLAttributeTagName,
						Attributes: attributes("name", "data-id"),
						Content:    "42",
					}},
				},
				{
					Type:       SelectorTagName,
					Attributes: attributes("path", ".image td"),
					Children: []*node.Node{
						{
							Type:       HTMLAttributeTagName,
							Attributes: attributes("name", "data-name"),
							Content:    "43",
						},
						{
							Type:       HTMLAttributeTagName,
							Attributes: attributes("name", "title"),
							Content:    "image-cell",
						},
					},
				},
				{
					Type:       SelectorTagName,
					Attributes: attributes("path", ".text div"),
					Children: []*node.Node{{
						Type:       HTMLAttributeTagName,
						Attributes: attributes("name", "data-extra"),
						Content:    "second",
					}},
				},
			},
		}},
	}

	ctx := &RenderContext{}
	require.NoError(t, (MJML{}).preparseHeadMetaValues(ctx, head))

	require.Equal(t, []HTMLAttributeSelector{
		{
			Path: ".text div",
			Attributes: []HTMLAttribute{
				{Name: "data-id", Value: "42"},
				{Name: "data-extra", Value: "second"},
			},
		},
		{
			Path: ".image td",
			Attributes: []HTMLAttribute{
				{Name: "data-name", Value: "43"},
				{Name: "title", Value: "image-cell"},
			},
		},
	}, ctx.HTMLAttributes)
}

func TestValidateHTMLAttributeTags(t *testing.T) {
	t.Parallel()

	mjmlTree := func(headChild *node.Node, bodyChildren ...*node.Node) *node.Node {
		head := &node.Node{Type: HeadTagName}
		if headChild != nil {
			head.Children = []*node.Node{headChild}
		}

		return &node.Node{
			Type: MJMLTagName,
			Children: []*node.Node{
				head,
				{Type: BodyTagName, Children: bodyChildren},
			},
		}
	}

	selector := func(path string, children ...*node.Node) *node.Node {
		attributes := []xml.Attr{}
		if path != "" {
			attributes = append(attributes, attribute("path", path))
		}

		return &node.Node{Type: SelectorTagName, Attributes: attributes, Children: children}
	}

	htmlAttribute := func(name string) *node.Node {
		attributes := []xml.Attr{}
		if name != "" {
			attributes = append(attributes, attribute("name", name))
		}

		return &node.Node{Type: HTMLAttributeTagName, Attributes: attributes, Content: "value"}
	}

	tests := []struct {
		name     string
		root     *node.Node
		wantErr  string
		wantKind ErrorKind
	}{
		{
			name: "valid",
			root: mjmlTree(&node.Node{
				Type: HTMLAttributesTagName,
				Children: []*node.Node{
					selector(".text", htmlAttribute("data-id")),
				},
			}),
		},
		{
			name:     "mj-html-attributes outside mj-head",
			root:     mjmlTree(nil, &node.Node{Type: HTMLAttributesTagName}),
			wantErr:  "<mj-html-attributes> must be a direct child of <mj-head>",
			wantKind: KindDisallowedChild,
		},
		{
			name:     "mj-selector outside mj-html-attributes",
			root:     mjmlTree(selector(".text", htmlAttribute("data-id"))),
			wantErr:  "<mj-selector> must be a direct child of <mj-html-attributes>",
			wantKind: KindDisallowedChild,
		},
		{
			name: "mj-html-attribute outside mj-selector",
			root: mjmlTree(&node.Node{
				Type:     HTMLAttributesTagName,
				Children: []*node.Node{htmlAttribute("data-id")},
			}),
			wantErr:  "<mj-html-attribute> must be a direct child of <mj-selector>",
			wantKind: KindDisallowedChild,
		},
		{
			name: "missing path",
			root: mjmlTree(&node.Node{
				Type:     HTMLAttributesTagName,
				Children: []*node.Node{selector("", htmlAttribute("data-id"))},
			}),
			wantErr:  "missing path in mj-selector element",
			wantKind: KindInvalidValue,
		},
		{
			name: "missing name",
			root: mjmlTree(&node.Node{
				Type:     HTMLAttributesTagName,
				Children: []*node.Node{selector(".text", htmlAttribute(""))},
			}),
			wantErr:  "missing name in mj-html-attribute element",
			wantKind: KindInvalidValue,
		},
		{
			name:     "mj-selector nested in body",
			root:     mjmlTree(nil, selector(".text", htmlAttribute("data-id"))),
			wantErr:  "<mj-selector> must be a direct child of <mj-html-attributes>",
			wantKind: KindDisallowedChild,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := (MJML{}).validateHTMLAttributeTags(tt.root, nil)

			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}

			require.ErrorContains(t, err, tt.wantErr)

			var structured *Error
			require.ErrorAs(t, err, &structured)
			assert.Equal(t, tt.wantKind, structured.Kind)
			assert.NotEmpty(t, structured.Path)
		})
	}
}
