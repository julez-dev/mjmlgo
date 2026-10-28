package mjmlgo

import "github.com/julez-dev/mjmlgo/component"

// Error is the structured failure every validation and parse site returns. It is
// the same type as component.Error, aliased here so callers of this package can
// use errors.As to a *mjmlgo.Error without importing the component package.
//
// Inspect Kind, Path, Attribute and Value instead of matching the message text:
// the text is for people and may change. errors.Is works for the package's
// error sentinels (ErrValidation, ErrParsingFailed, ErrIncludeCycle, ...) and
// errors.As unwraps through wrapping.
type Error = component.Error

// ErrorKind classifies a structured Error. See the component package constants
// for the closed set of kinds.
type ErrorKind = component.ErrorKind

// The closed set of structured-error kinds, re-exported from the component
// package.
const (
	// KindUnknownElement reports an element that is not part of MJML.
	KindUnknownElement = component.KindUnknownElement
	// KindUnknownAttribute reports an attribute that is not allowed on its
	// element.
	KindUnknownAttribute = component.KindUnknownAttribute
	// KindInvalidValue reports an attribute value that does not match MJML's
	// type for it, or a required attribute that is missing.
	KindInvalidValue = component.KindInvalidValue
	// KindDisallowedChild reports an element nested where MJML's dependency
	// table does not allow it.
	KindDisallowedChild = component.KindDisallowedChild
	// KindUnknownIncludeType reports an mj-include whose type is not mjml, html
	// or css.
	KindUnknownIncludeType = component.KindUnknownIncludeType
	// KindUndefinedMJClass reports an mj-class reference to a class the document
	// does not declare in mj-attributes.
	KindUndefinedMJClass = component.KindUndefinedMJClass
	// KindParseFailure reports a document that could not be read into a
	// renderable MJML tree.
	KindParseFailure = component.KindParseFailure
)

// ErrValidation reports a document that failed validation. A structured error
// of a validation kind matches it with errors.Is.
var ErrValidation = component.ErrValidation
