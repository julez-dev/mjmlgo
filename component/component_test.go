package component

import (
	"testing"

	"github.com/julez-dev/mjmlgo/v2/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// documentWithColumnLeaf builds mjml > mj-body > mj-section > mj-column > leaf.
func documentWithColumnLeaf(leaf *node.Node) *node.Node {
	root := &node.Node{Type: MJMLTagName}
	body := &node.Node{Type: BodyTagName}
	section := &node.Node{Type: SectionTagName}
	column := &node.Node{Type: ColumnTagName}

	link := func(parent, child *node.Node) {
		child.Parent = parent
		parent.Children = append(parent.Children, child)
	}

	link(root, body)
	link(body, section)
	link(section, column)
	link(column, leaf)

	return root
}

func TestAllowsChild(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		parent string
		child  string
		want   bool
	}{
		{name: "mjml accepts body", parent: MJMLTagName, child: BodyTagName, want: true},
		{name: "body accepts section", parent: BodyTagName, child: SectionTagName, want: true},
		{name: "wrapper accepts section", parent: WrapperTagName, child: SectionTagName, want: true},
		{name: "wrapper accepts hero", parent: WrapperTagName, child: HeroTagName, want: true},
		{name: "section accepts group", parent: SectionTagName, child: GroupTagName, want: true},
		{name: "column accepts carousel", parent: ColumnTagName, child: CarouselTagName, want: true},
		{name: "hero accepts navbar", parent: HeroTagName, child: NavbarTagName, want: true},
		{name: "section rejects text", parent: SectionTagName, child: TextTagName, want: false},
		{name: "body rejects text", parent: BodyTagName, child: TextTagName, want: false},
		{name: "column rejects section", parent: ColumnTagName, child: SectionTagName, want: false},
		{name: "attributes accepts any component tag", parent: AttributesTagName, child: TextTagName, want: true},
		{name: "attributes accepts mj-class", parent: AttributesTagName, child: ClassTagName, want: true},
		{name: "class accepts a nested per-tag default", parent: ClassTagName, child: TextTagName, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, allowsChild(&RenderContext{}, tt.parent, tt.child))
		})
	}
}

func TestValidateDocumentRejectsUnknownElementWithPath(t *testing.T) {
	t.Parallel()

	unknown := &node.Node{Type: "mj-foo"}
	err := ValidateDocument(&RenderContext{}, documentWithColumnLeaf(unknown))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "mj-body > mj-section > mj-column > mj-foo: unknown element <mj-foo>")
}

func TestValidateDocumentRejectsUnknownHeadElement(t *testing.T) {
	t.Parallel()

	root := &node.Node{Type: MJMLTagName}
	head := &node.Node{Type: HeadTagName, Parent: root}
	unknown := &node.Node{Type: "mj-unknown", Parent: head}
	head.Children = []*node.Node{unknown}
	root.Children = []*node.Node{head}

	err := ValidateDocument(&RenderContext{}, root)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mj-head > mj-unknown: unknown element <mj-unknown>")
}

func TestValidateDocumentRejectsDisallowedChild(t *testing.T) {
	t.Parallel()

	root := &node.Node{Type: MJMLTagName}
	body := &node.Node{Type: BodyTagName, Parent: root}
	text := &node.Node{Type: TextTagName, Parent: body}
	body.Children = []*node.Node{text}
	root.Children = []*node.Node{body}

	err := ValidateDocument(&RenderContext{}, root)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mj-body > mj-text: <mj-text> is not allowed inside <mj-body>")
	assert.Contains(t, err.Error(), "mj-section")
}

func TestValidateDocumentRejectsUnknownAttribute(t *testing.T) {
	t.Parallel()

	text := &node.Node{Type: TextTagName, Attributes: attributes("made-up", "value")}
	err := ValidateDocument(&RenderContext{}, documentWithColumnLeaf(text))

	require.Error(t, err)
	assert.Contains(t, err.Error(), `mj-body > mj-section > mj-column > mj-text: attribute "made-up" is not allowed on <mj-text>`)
}

func TestValidateDocumentAcceptsUniversalAttributes(t *testing.T) {
	t.Parallel()

	ctx := &RenderContext{Classes: map[string]AttributeMap{"declared": {}}}
	text := &node.Node{
		Type:       TextTagName,
		Attributes: attributes("css-class", "custom", "mj-class", "declared"),
	}

	require.NoError(t, ValidateDocument(ctx, documentWithColumnLeaf(text)))
}

func TestValidateDocumentRejectsUndeclaredMjClass(t *testing.T) {
	t.Parallel()

	text := &node.Node{Type: TextTagName, Attributes: attributes("mj-class", "missing")}
	err := ValidateDocument(&RenderContext{}, documentWithColumnLeaf(text))

	require.Error(t, err)
	assert.Contains(t, err.Error(), `mj-body > mj-section > mj-column > mj-text: mj-class "missing" is not declared in <mj-attributes>`)
}

func TestValidateDocumentAttributes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		tag     string
		attr    string
		value   string
		wantErr bool
	}{
		// unitWithNegative is used exactly where MJML 5.4 uses it.
		{name: "text negative letter-spacing", tag: TextTagName, attr: "letter-spacing", value: "-1px"},
		{name: "button negative letter-spacing", tag: ButtonTagName, attr: "letter-spacing", value: "-0.5em"},
		{name: "navbar-link negative letter-spacing", tag: NavbarLinkTagName, attr: "letter-spacing", value: "-1px"},
		{name: "accordion-text negative letter-spacing", tag: AccordionTextTagName, attr: "letter-spacing", value: "-1px"},
		{name: "text negative font-size", tag: TextTagName, attr: "font-size", value: "-12px", wantErr: true},
		{name: "divider negative width", tag: DividerTagName, attr: "width", value: "-10px", wantErr: true},
		// auto is allowed on image height and table width only.
		{name: "image auto height", tag: ImageTagName, attr: "height", value: "auto"},
		{name: "image auto width", tag: ImageTagName, attr: "width", value: "auto", wantErr: true},
		{name: "table auto width", tag: TableTagName, attr: "width", value: "auto"},
		{name: "spacer auto height", tag: SpacerTagName, attr: "height", value: "auto", wantErr: true},
		// wrapper has gap, section does not.
		{name: "wrapper gap", tag: WrapperTagName, attr: "gap", value: "20px"},
		{name: "section gap is unknown", tag: SectionTagName, attr: "gap", value: "20px", wantErr: true},
		// hero gained width in MJML 5, but still has no container background.
		{name: "hero width", tag: HeroTagName, attr: "width", value: "10px"},
		{name: "hero container background is unknown", tag: HeroTagName, attr: "container-background-color", value: "#fff", wantErr: true},
		{name: "hero mode is a string", tag: HeroTagName, attr: "mode", value: "anything"},
		// group direction only allows ltr/rtl.
		{name: "group direction auto", tag: GroupTagName, attr: "direction", value: "auto", wantErr: true},
		// social-element has no inner-padding attribute of its own.
		{name: "social-element inner-padding is unknown", tag: SocialElementTagName, attr: "inner-padding", value: "1px", wantErr: true},
		// section full-width enum includes the empty value.
		{name: "section empty full-width", tag: SectionTagName, attr: "full-width", value: ""},
		// table cellpadding is MJML's unanchored integer matcher.
		{name: "table cellpadding integer", tag: TableTagName, attr: "cellpadding", value: "12px"},
		{name: "table cellpadding without digit", tag: TableTagName, attr: "cellpadding", value: "none", wantErr: true},
		// head components.
		{name: "breakpoint width", tag: BreakpointTagName, attr: "width", value: "480px"},
		{name: "breakpoint wrong unit", tag: BreakpointTagName, attr: "width", value: "30em", wantErr: true},
		{name: "font name", tag: FontTagName, attr: "name", value: "Roboto"},
		{name: "style inline", tag: StyleTagName, attr: "inline", value: "inline"},
		{name: "preview has no attributes", tag: PreviewTagName, attr: "foo", value: "bar", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			n := &node.Node{Type: tt.tag, Attributes: attributes(tt.attr, tt.value)}
			err := validateDocumentAttributes(&RenderContext{}, n)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestInitComponentValidatesBeforeDefaults(t *testing.T) {
	t.Parallel()

	t.Run("invalid author value is rejected", func(t *testing.T) {
		t.Parallel()

		text := &node.Node{Type: TextTagName, Attributes: attributes("color", "chartreuseish")}
		err := InitComponent(&RenderContext{}, MJMLText{}, text)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "color")
		assert.Contains(t, err.Error(), "chartreuseish not a valid color string")
	})

	t.Run("defaults are applied after author attributes are checked", func(t *testing.T) {
		t.Parallel()

		text := &node.Node{Type: TextTagName, Attributes: attributes("color", "#ff0000")}
		require.NoError(t, InitComponent(&RenderContext{}, MJMLText{}, text))
		assert.Equal(t, "#ff0000", text.GetAttributeValueDefault("color"))
		assert.Equal(t, "13px", text.GetAttributeValueDefault("font-size"))
	})
}
