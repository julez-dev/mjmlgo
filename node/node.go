package node

import "encoding/xml"

type Node struct {
	Type       string
	Attributes []xml.Attr `json:"-"`
	Content    string
	Children   []*Node
	Parent     *Node `json:"-"`
}

func (n *Node) SetAttribute(name, value string) {
	for i, attr := range n.Attributes {
		if attr.Name.Local == name {
			n.Attributes[i].Value = value
			return
		}
	}
	n.Attributes = append(n.Attributes, xml.Attr{Name: xml.Name{Local: name}, Value: value})
}

func (n *Node) GetAttributeValue(name string) (string, bool) {
	for _, attr := range n.Attributes {
		if attr.Name.Local == name {
			return attr.Value, true
		}
	}
	return "", false
}

func (n *Node) GetAttributeValueDefault(name string) string {
	for _, attr := range n.Attributes {
		if attr.Name.Local == name {
			return attr.Value
		}
	}
	return ""
}
