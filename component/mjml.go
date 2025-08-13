package component

import (
	"errors"
	"fmt"
	"io"
	"slices"

	"github.com/julez-dev/mjmlgo/node"
)

var ErrMJMLBadlyFormatted = errors.New("MJML badly formatted")

type MJML struct{}

func (m MJML) Render(ctx *RenderContext, w io.Writer, n *node.Node) error {

	if len(n.Children) < 1 {
		return fmt.Errorf("%w: no children in <mjml> tag", ErrMJMLBadlyFormatted)
	}

	hasBody := slices.ContainsFunc(n.Children, func(node *node.Node) bool {
		return node.Type == BodyTagName
	})

	if !hasBody {
		return fmt.Errorf("%w: no <mj-body> in <mjml> tag", ErrMJMLBadlyFormatted)
	}

	_, _ = io.WriteString(w, `<!doctype html>`)
	_, _ = io.WriteString(w, `<html xmlns="http://www.w3.org/1999/xhtml" xmlns:v="urn:schemas-microsoft-com:vml" xmlns:o="urn:schemas-microsoft-com:office:office">`)

	for _, child := range n.Children {
		switch child.Type {
		case HeadTagName:
			head := Head{}
			if err := head.Render(ctx, w, child); err != nil {
				return err
			}
		}
	}

	_, _ = io.WriteString(w, "</html>")

	return nil
}
