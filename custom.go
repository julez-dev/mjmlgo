package mjmlgo

import (
	"errors"
	"fmt"

	"github.com/julez-dev/mjmlgo/component"
)

// WithComponents registers custom components on the Renderer. A registered tag
// becomes a first-class MJML element: it is validated like a built-in one,
// receives the defaults declared in <mj-attributes> and <mj-class>, and renders
// at its document position. Registration is per renderer - there is no global
// registry - so two renderers in one process may know different tags.
//
// A custom component is a leaf: the validator accepts it where content goes (as
// a direct child of mj-body, mj-column or mj-hero) and rejects children under
// its own tag unless it implements component.ChildComponent. Its tag must not be
// empty, must not be registered twice and must not shadow a built-in element;
// such a renderer is rejected with a *component.Error of kind
// component.KindInvalidComponent when it renders.
func WithComponents(components ...component.Component) Option {
	registered := make([]component.Component, len(components))
	copy(registered, components)

	return func(config *rendererConfig) {
		config.components = registered
	}
}

// componentTable indexes the custom components a Renderer was configured with
// by tag name. Registration is checked once, when the renderer is built, so
// every render reports the same configuration failure instead of a
// document-dependent one. The returned map is never modified after this point,
// which is what lets concurrent renders share it.
func componentTable(components []component.Component) (map[string]component.Component, error) {
	if len(components) == 0 {
		return nil, nil
	}

	table := make(map[string]component.Component, len(components))

	for _, comp := range components {
		if comp == nil {
			return nil, registrationError("", errors.New("nil component"))
		}

		tag := comp.Name()
		if tag == "" {
			return nil, registrationError("", errors.New("component tag is empty"))
		}

		if component.IsBuiltinTag(tag) {
			return nil, registrationError(tag, fmt.Errorf("tag <%s> is a built-in MJML element", tag))
		}

		if _, duplicate := table[tag]; duplicate {
			return nil, registrationError(tag, fmt.Errorf("tag <%s> is already registered", tag))
		}

		table[tag] = comp
	}

	return table, nil
}

// registrationError builds the structured failure for a component the renderer
// cannot register. There is no offending document element, so the path is the
// document element the renderer would fail to build; value is the tag, when the
// component has one.
func registrationError(tag string, err error) error {
	return component.NewError(
		component.KindInvalidComponent,
		[]string{component.MJMLTagName},
		"",
		tag,
		err,
	)
}
