package component

import "encoding/xml"

type RenderContext struct {
	GlobalTextAttibutes     [][]xml.Attr
	GlobalMJClassAttributes [][]xml.Attr
	GlobalAllAttibutes      [][]xml.Attr

	PreviewText string
}
