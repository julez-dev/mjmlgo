package mjmlgo

import "github.com/julez-dev/mjmlgo/v2/component"

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
// package. Each kind's meaning is documented on the component constant of the
// same name; the block is the whole set: KindUnknownElement,
// KindUnknownAttribute, KindInvalidValue, KindDisallowedChild,
// KindUnknownIncludeType, KindUndefinedMJClass and KindParseFailure.
const (
	KindUnknownElement     = component.KindUnknownElement
	KindUnknownAttribute   = component.KindUnknownAttribute
	KindInvalidValue       = component.KindInvalidValue
	KindDisallowedChild    = component.KindDisallowedChild
	KindUnknownIncludeType = component.KindUnknownIncludeType
	KindUndefinedMJClass   = component.KindUndefinedMJClass
	KindParseFailure       = component.KindParseFailure
)

// ErrValidation reports a document that failed validation. A structured error
// of a validation kind matches it with errors.Is.
var ErrValidation = component.ErrValidation
