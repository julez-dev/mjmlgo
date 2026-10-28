package node

import (
	"encoding/xml"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAttributePresenceIsDistinguishableFromAbsence pins the difference between
// "the attribute is missing" and "the attribute is present with an empty value": a
// caller that only looks at the returned string cannot tell them apart, and
// MJML treats an explicitly empty attribute differently from an absent one.
func TestAttributePresenceIsDistinguishableFromAbsence(t *testing.T) {
	t.Parallel()

	n := &Node{Attributes: []xml.Attr{
		{Name: xml.Name{Local: "width"}, Value: "600px"},
		{Name: xml.Name{Local: "background-color"}, Value: ""},
	}}

	value, ok := n.GetAttributeValue("width")
	require.True(t, ok)
	assert.Equal(t, "600px", value)

	value, ok = n.GetAttributeValue("background-color")
	assert.True(t, ok, "an attribute written with an empty value is present")
	assert.Empty(t, value)

	value, ok = n.GetAttributeValue("height")
	assert.False(t, ok)
	assert.Empty(t, value)

	assert.Empty(t, n.GetAttributeValueDefault("height"))
	assert.Empty(t, n.GetAttributeValueDefault("background-color"))
}

// TestSettingAnAttributeKeepsItsDocumentPosition asserts that setting an
// attribute the node already carries keeps its document position. Rendering
// appends defaults to the author's attributes, so a default must never move an
// author attribute or add a second copy of it.
func TestSettingAnAttributeKeepsItsDocumentPosition(t *testing.T) {
	t.Parallel()

	n := &Node{Attributes: []xml.Attr{
		{Name: xml.Name{Local: "align"}, Value: "left"},
		{Name: xml.Name{Local: "width"}, Value: "300px"},
	}}

	n.SetAttribute("width", "200px")

	require.Len(t, n.Attributes, 2, "replacing must not append a duplicate")
	assert.Equal(t, "align", n.Attributes[0].Name.Local)
	assert.Equal(t, "width", n.Attributes[1].Name.Local)
	assert.Equal(t, "200px", n.Attributes[1].Value)
}

// TestNewAttributesAppendAfterAuthorAttributes asserts that a new attribute -
// the shape a component's default takes - goes after the attributes already on
// the node, and that setting the same name again updates it in place.
func TestNewAttributesAppendAfterAuthorAttributes(t *testing.T) {
	t.Parallel()

	n := &Node{Attributes: []xml.Attr{
		{Name: xml.Name{Local: "background-color"}, Value: "#ff0000"},
	}}

	n.SetAttribute("width", "600px")
	n.SetAttribute("width", "480px")

	require.Len(t, n.Attributes, 2)
	assert.Equal(t, "background-color", n.Attributes[0].Name.Local)
	assert.Equal(t, "width", n.Attributes[1].Name.Local)

	width, ok := n.GetAttributeValue("width")
	require.True(t, ok)
	assert.Equal(t, "480px", width)
}
