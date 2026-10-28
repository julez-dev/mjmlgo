package mjmlgo

import (
	"errors"
	"fmt"
	"reflect"

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
// component.KindInvalidComponent, whose Path names the offending tag, when it
// renders. Calling WithComponents more than once registers the union, so a
// duplicate tag across calls is rejected like a duplicate within one call.
func WithComponents(components ...component.Component) Option {
	registered := make([]component.Component, len(components))
	copy(registered, components)

	return func(config *rendererConfig) {
		config.components = append(config.components, registered...)
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
		if isNilComponent(comp) {
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

// isNilComponent reports whether comp is nil, including the typed nil pointer a
// caller can pass without the interface itself being nil. Either would panic when
// Name is called, so registration reports it instead.
func isNilComponent(comp component.Component) bool {
	if comp == nil {
		return true
	}

	value := reflect.ValueOf(comp)

	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

// registrationError builds the structured failure for a component the renderer
// cannot register. There is no offending document element, so the path names the
// tag that cannot be registered; a component without a tag (a nil component or
// an empty tag) uses the document element instead. Value carries the tag when
// there is one.
func registrationError(tag string, err error) error {
	path := []string{component.MJMLTagName}
	if tag != "" {
		path = []string{tag}
	}

	return component.NewError(
		component.KindInvalidComponent,
		path,
		"",
		tag,
		err,
	)
}
