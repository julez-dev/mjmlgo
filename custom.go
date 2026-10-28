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
// A custom component is accepted where content goes - as a direct child of
// mj-body, mj-column or mj-hero - and is a leaf unless it implements
// component.ChildComponent. A leaf rejects every child under its own tag. A
// container declares the children it accepts and renders them with
// component.RenderChildren, so it controls the markup around them; its declared
// children must be built-in content components or other components registered on
// the same renderer.
//
// A component's tag must not be empty, must not be registered twice and must not
// shadow a built-in element; such a renderer is rejected with a *component.Error
// of kind component.KindInvalidComponent, whose Path names the offending tag (and
// whose Value names the declared child, for a child nothing can render), when it
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
			return nil, registrationError("", "", errors.New("nil component"))
		}

		tag := comp.Name()
		if tag == "" {
			return nil, registrationError("", "", errors.New("component tag is empty"))
		}

		if component.IsBuiltinTag(tag) {
			return nil, registrationError(tag, tag, fmt.Errorf("tag <%s> is a built-in MJML element", tag))
		}

		if _, duplicate := table[tag]; duplicate {
			return nil, registrationError(tag, tag, fmt.Errorf("tag <%s> is already registered", tag))
		}

		table[tag] = comp
	}

	// A container's declared children are validated against the finished table,
	// so a container may declare a custom component registered after it.
	for _, comp := range components {
		if err := validateDeclaredChildren(comp, table); err != nil {
			return nil, err
		}
	}

	return table, nil
}

// validateDeclaredChildren rejects a container that declares a child nothing can
// render. The children of a custom container are rendered by
// component.RenderChildren, which dispatches the built-in content components and
// the custom components registered on the same renderer. A declaration outside
// those two sets would pass document validation and then fail the render, so it
// is a renderer configuration error instead - reported for every document, like
// the other registration failures.
func validateDeclaredChildren(comp component.Component, table map[string]component.Component) error {
	container, isContainer := comp.(component.ChildComponent)
	if !isContainer {
		return nil
	}

	for _, child := range container.AllowedChildren() {
		if component.IsContentTag(child) {
			continue
		}

		if _, registered := table[child]; registered {
			continue
		}

		return registrationError(comp.Name(), child, fmt.Errorf(
			"container <%s> declares child <%s>, which is not a content component or a registered custom component",
			comp.Name(), child,
		))
	}

	return nil
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
// an empty tag) uses the document element instead. Value carries what exactly is
// wrong: the tag itself, or the declared child a container cannot render.
func registrationError(tag, value string, err error) error {
	path := []string{component.MJMLTagName}
	if tag != "" {
		path = []string{tag}
	}

	return component.NewError(
		component.KindInvalidComponent,
		path,
		"",
		value,
		err,
	)
}
