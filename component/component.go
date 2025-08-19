package component

import (
	"fmt"
	"io"

	"github.com/julez-dev/mjmlgo/node"
)

type Component interface {
	Name() string
	AllowedAttributes() map[string]validateAttributeFunc
	DefaultAttributes(ctx *RenderContext) map[string]string
	Render(ctx *RenderContext, w io.Writer, n *node.Node) error
}

func InitComponent(ctx *RenderContext, comp Component, n *node.Node) error {
	for key, value := range comp.DefaultAttributes(ctx) {
		if _, has := n.GetAttributeValue(key); !has {
			n.SetAttribute(key, value)
		}
	}

	for field, validator := range comp.AllowedAttributes() {
		val := n.GetAttributeValueDefault(field)

		if err := validator(val); err != nil {
			return fmt.Errorf("failed to validate field %s in <%s>: %w", field, comp.Name(), err)
		}
	}

	return nil
}
