package component

import (
	"io"
	"testing"

	"github.com/julez-dev/mjmlgo/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testComponent is a minimal custom component for the validation tests here. The
// public contract is proven end to end by the example package and the root
// package's tests; this one only has to be a Component.
type testComponent struct {
	tag      string
	attrs    map[string]AttributeValidator
	defaults map[string]string
}

func (c testComponent) Name() string { return c.tag }

func (c testComponent) AllowedAttributes() map[string]AttributeValidator { return c.attrs }

func (c testComponent) DefaultAttributes(*RenderContext) map[string]string { return c.defaults }

func (c testComponent) Render(*RenderContext, io.Writer, *node.Node) error { return nil }

// testChildComponent is a testComponent that also declares its children.
type testChildComponent struct {
	testComponent
	children []string
}

func (c testChildComponent) AllowedChildren() []string { return c.children }

// contextWith registers components on a fresh render context.
func contextWith(components ...Component) *RenderContext {
	registered := make(map[string]Component, len(components))
	for _, comp := range components {
		registered[comp.Name()] = comp
	}

	return &RenderContext{CustomComponents: registered}
}

func TestAllowsChildWithCustomComponents(t *testing.T) {
	t.Parallel()

	leaf := testComponent{tag: "mj-badge"}
	parent := testChildComponent{
		tag:      "mj-panel",
		children: []string{TextTagName},
	}
	ctx := contextWith(leaf, parent)

	tests := []struct {
		name   string
		ctx    *RenderContext
		parent string
		child  string
		want   bool
	}{
		{name: "body accepts a custom leaf", ctx: ctx, parent: BodyTagName, child: "mj-badge", want: true},
		{name: "column accepts a custom leaf", ctx: ctx, parent: ColumnTagName, child: "mj-badge", want: true},
		{name: "hero accepts a custom leaf", ctx: ctx, parent: HeroTagName, child: "mj-badge", want: true},
		{name: "section rejects a custom leaf", ctx: ctx, parent: SectionTagName, child: "mj-badge"},
		{name: "wrapper rejects a custom leaf", ctx: ctx, parent: WrapperTagName, child: "mj-badge"},
		{name: "group rejects a custom leaf", ctx: ctx, parent: GroupTagName, child: "mj-badge"},
		{name: "a leaf custom component rejects every child", ctx: ctx, parent: "mj-badge", child: TextTagName},
		{name: "a child component accepts a declared child", ctx: ctx, parent: "mj-panel", child: TextTagName, want: true},
		{name: "a child component rejects an undeclared child", ctx: ctx, parent: "mj-panel", child: ImageTagName},
		{name: "mj-attributes accepts a custom tag", ctx: ctx, parent: AttributesTagName, child: "mj-badge", want: true},
		{name: "mj-class accepts a custom tag", ctx: ctx, parent: ClassTagName, child: "mj-badge", want: true},
		{name: "a custom tag is not accepted without a renderer", ctx: &RenderContext{}, parent: ColumnTagName, child: "mj-badge"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, allowsChild(tt.ctx, tt.parent, tt.child))
		})
	}
}

func TestAllowedChildListOfCustomComponents(t *testing.T) {
	t.Parallel()

	leaf := testComponent{tag: "mj-badge"}
	parent := testChildComponent{
		tag:      "mj-panel",
		children: []string{TextTagName, ImageTagName},
	}

	assert.Equal(t, "none", allowedChildList(contextWith(leaf), leaf.Name()))
	assert.Equal(t, "mj-text, mj-image", allowedChildList(contextWith(parent), parent.Name()))
	assert.Equal(t, "mj-raw, mj-section, mj-wrapper, mj-hero", allowedChildList(&RenderContext{}, BodyTagName))
}

func TestValidateDocumentCustomComponent(t *testing.T) {
	t.Parallel()

	leaf := testComponent{
		tag: "mj-badge",
		attrs: map[string]AttributeValidator{
			"background-color": func(value string) error {
				if value != "#1f2937" {
					return assert.AnError
				}

				return nil
			},
		},
	}

	t.Run("accepts a declared attribute", func(t *testing.T) {
		t.Parallel()

		badge := &node.Node{Type: "mj-badge", Attributes: attributes("background-color", "#1f2937")}
		require.NoError(t, ValidateDocument(contextWith(leaf), documentWithColumnLeaf(badge)))
	})

	t.Run("rejects an undeclared attribute", func(t *testing.T) {
		t.Parallel()

		badge := &node.Node{Type: "mj-badge", Attributes: attributes("made-up", "value")}
		err := ValidateDocument(contextWith(leaf), documentWithColumnLeaf(badge))

		var structured *Error
		require.ErrorAs(t, err, &structured)
		assert.Equal(t, KindUnknownAttribute, structured.Kind)
		assert.Equal(t, []string{"mj-body", "mj-section", "mj-column", "mj-badge"}, structured.Path)
		assert.Equal(t, "made-up", structured.Attribute)
	})

	t.Run("rejects an invalid declared value", func(t *testing.T) {
		t.Parallel()

		badge := &node.Node{Type: "mj-badge", Attributes: attributes("background-color", "chartreuseish")}
		err := ValidateDocument(contextWith(leaf), documentWithColumnLeaf(badge))

		var structured *Error
		require.ErrorAs(t, err, &structured)
		assert.Equal(t, KindInvalidValue, structured.Kind)
		assert.Equal(t, "background-color", structured.Attribute)
		assert.Equal(t, "chartreuseish", structured.Value)
	})

	t.Run("rejects a child of a leaf", func(t *testing.T) {
		t.Parallel()

		badge := &node.Node{Type: "mj-badge"}
		text := &node.Node{Type: TextTagName, Parent: badge}
		badge.Children = []*node.Node{text}

		err := ValidateDocument(contextWith(leaf), documentWithColumnLeaf(badge))

		var structured *Error
		require.ErrorAs(t, err, &structured)
		assert.Equal(t, KindDisallowedChild, structured.Kind)
		assert.Equal(t, []string{"mj-body", "mj-section", "mj-column", "mj-badge", "mj-text"}, structured.Path)
		assert.Equal(t, TextTagName, structured.Value)
	})
}

func TestIsBuiltinTag(t *testing.T) {
	t.Parallel()

	for _, tag := range []string{
		MJMLTagName, BodyTagName, TextTagName, AttributesTagName, ClassTagName, AllTagName, IncludeTagName,
	} {
		assert.True(t, IsBuiltinTag(tag), "%s should be a built-in tag", tag)
	}

	for _, tag := range []string{"mj-badge", "mj-foo", ""} {
		assert.False(t, IsBuiltinTag(tag), "%s should not be a built-in tag", tag)
	}
}
