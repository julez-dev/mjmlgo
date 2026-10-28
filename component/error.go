package component

import (
	"fmt"
	"strings"
)

// ErrorKind classifies why a document was rejected. The set is closed: every
// validation and parse failure this package reports is one of these kinds, so a
// caller can branch on it instead of matching error text.
type ErrorKind string

const (
	// KindUnknownElement reports an element that is not part of MJML.
	KindUnknownElement ErrorKind = "unknown element"
	// KindUnknownAttribute reports an attribute that is not allowed on its
	// element.
	KindUnknownAttribute ErrorKind = "unknown attribute"
	// KindInvalidValue reports an attribute value that does not match MJML's
	// type for it, or a required attribute that is missing.
	KindInvalidValue ErrorKind = "invalid value"
	// KindDisallowedChild reports an element nested where MJML's dependency
	// table does not allow it.
	KindDisallowedChild ErrorKind = "disallowed child"
	// KindUnknownIncludeType reports an mj-include whose type is not mjml, html
	// or css.
	KindUnknownIncludeType ErrorKind = "unknown include type"
	// KindUndefinedMJClass reports an mj-class reference to a class the document
	// does not declare in mj-attributes.
	KindUndefinedMJClass ErrorKind = "undefined mj-class"
	// KindParseFailure reports a document that could not be read into a
	// renderable MJML tree: malformed XML, no root element, no mj-body, or an
	// mj-include that cannot be resolved. Include I/O failures are in this
	// kind because they are failures to assemble the document, not
	// attribute validation.
	KindParseFailure ErrorKind = "parse failure"
	// KindInvalidComponent reports a custom component the renderer refuses to
	// register: a nil component, an empty tag, a tag registered twice, or a tag
	// that shadows a built-in element. It is a renderer-configuration failure
	// rather than a document failure, so it does not match ErrValidation. There
	// is no offending document element: Path names the tag that cannot be
	// registered, or the document element when the component has no tag, and
	// Value is the tag when there is one.
	KindInvalidComponent ErrorKind = "invalid component"
)

// validationKinds are the kinds errors.Is reports as ErrValidation. The include
// type has its own sentinel (ErrIncludeType), and a parse failure never counts
// as validation.
var validationKinds = map[ErrorKind]struct{}{
	KindUnknownElement:   {},
	KindUnknownAttribute: {},
	KindInvalidValue:     {},
	KindDisallowedChild:  {},
	KindUndefinedMJClass: {},
}

// Error is the structured failure returned by every validation and parse site.
// It carries the offending element's path as data, the attribute and value when
// they are what failed, and the underlying cause.
//
// Error text is for humans and may change; callers should inspect Kind, Path,
// Attribute and Value, or use errors.Is/errors.As. Unwrap exposes Err, so
// errors.Is works for the sentinels (ErrValidation, ErrIncludeCycle, ...) that
// used to be returned on their own.
type Error struct {
	// Kind is the classification of the failure.
	Kind ErrorKind
	// Path is the element path from the document root to the offending element,
	// e.g. []string{"mj-body", "mj-section", "mj-text"}. The root <mjml>
	// element is omitted, like the paths in error messages always have been. A
	// document-level failure that happens before an element exists uses the
	// document element it failed to build. Every site in this package supplies
	// a non-empty Path; the constructor does not enforce it for hand-built
	// errors.
	Path []string
	// Attribute is the attribute that failed, when the failure is about one.
	Attribute string
	// Value is the offending attribute value, or the offending element name for
	// element-level kinds.
	Value string
	// Err is the underlying cause. It is nil for failures that need no detail.
	Err error
}

// NewError is the single constructor for a structured failure. Every validation
// and parse site reports through it, so the kind set and the shape of the error
// cannot drift between sites.
func NewError(kind ErrorKind, path []string, attribute, value string, err error) *Error {
	return &Error{Kind: kind, Path: path, Attribute: attribute, Value: value, Err: err}
}

// Error renders the failure for people. The format follows the messages this
// implementation has always emitted, so logs keep their meaning; callers that
// need to branch on a failure use Kind or errors.Is instead.
func (e *Error) Error() string {
	var b strings.Builder

	if len(e.Path) > 0 {
		b.WriteString(strings.Join(e.Path, " > "))
		b.WriteString(": ")
	}

	switch e.Kind {
	case KindUnknownElement:
		fmt.Fprintf(&b, "unknown element <%s>", e.Value)
	case KindUnknownAttribute:
		fmt.Fprintf(&b, "attribute %q is not allowed on <%s>", e.Attribute, e.element())
	case KindInvalidValue:
		fmt.Fprintf(&b, "attribute %q in <%s>", e.Attribute, e.element())
		e.writeCause(&b)
	case KindDisallowedChild:
		fmt.Fprintf(&b, "<%s> is not allowed inside <%s>", e.Value, e.parent())
		if e.Err != nil {
			fmt.Fprintf(&b, " (%s)", e.Err)
		}
	case KindUndefinedMJClass:
		fmt.Fprintf(&b, "mj-class %q is not declared in <mj-attributes>", e.Value)
	case KindUnknownIncludeType:
		fmt.Fprintf(&b, "unknown mj-include type %q (want mjml, html or css)", e.Value)
	case KindParseFailure:
		b.WriteString("parse failure")
		e.writeCause(&b)
	case KindInvalidComponent:
		b.WriteString("invalid component")
		e.writeCause(&b)
	default:
		fmt.Fprintf(&b, "validation failed (%s)", e.Kind)
		e.writeCause(&b)
	}

	return b.String()
}

func (e *Error) writeCause(b *strings.Builder) {
	if e.Err == nil {
		return
	}

	b.WriteString(": ")
	b.WriteString(e.Err.Error())
}

// Unwrap returns the underlying cause, so errors.Is and errors.As keep working
// through a structured error.
func (e *Error) Unwrap() error {
	return e.Err
}

// Is reports the sentinels a structured error stands for. Validation kinds are
// ErrValidation even though their cause is a detailed message rather than the
// sentinel; every other sentinel is reached through Err.
func (e *Error) Is(target error) bool {
	if target != ErrValidation {
		return false
	}

	_, isValidation := validationKinds[e.Kind]

	return isValidation
}

// element returns the element the failure is about: the last path segment.
func (e *Error) element() string {
	if len(e.Path) == 0 {
		return ""
	}

	return e.Path[len(e.Path)-1]
}

// parent returns the element the offending child sits in: the second-to-last
// path segment. A one-segment path is a direct child of the document's root
// element, which element paths omit, so the root is the parent.
func (e *Error) parent() string {
	if len(e.Path) < 2 {
		if len(e.Path) == 1 && e.Path[0] != MJMLTagName {
			return MJMLTagName
		}

		return ""
	}

	return e.Path[len(e.Path)-2]
}
