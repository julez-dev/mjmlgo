package mjmlgo

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/julez-dev/mjmlgo/component"
)

var ErrUnknownStartingTag = errors.New("mjml: unknown starting tag")

func RenderMJML(input io.Reader) (string, error) {
	node, err := parse(input)
	if err != nil {
		return "", nil
	}

	if node.Type != "mjml" {
		return "", fmt.Errorf("%w: %s", ErrUnknownStartingTag, node.Type)
	}

	var buff strings.Builder
	spew.Dump(node)
	mjml := component.MJML{}
	ctx := &component.RenderContext{}
	if err := mjml.Render(ctx, &buff, node); err != nil {
		return "", err
	}

	spew.Dump(ctx)

	return buff.String(), nil
}
