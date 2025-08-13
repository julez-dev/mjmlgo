package node

import "encoding/xml"

type Node struct {
	Type       string
	Attributes []xml.Attr
	Content    string
	Children   []*Node
}
