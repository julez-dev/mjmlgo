package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/julez-dev/mjmlgo"
)

// reporter writes one record per input. Text output follows the reference CLI:
// a failure is announced as "File: <path>" followed by the error. JSON output is
// newline-delimited objects for tooling.
type reporter struct {
	json bool
	w    io.Writer
}

// failed reports an input that could not be rendered.
func (r reporter) failed(file string, err error) {
	if !r.json {
		fmt.Fprintf(r.w, "File: %s\n%s\n", displayName(file), err)

		return
	}

	r.write(jsonRecord{File: displayName(file), Error: describeError(err)})
}

// succeeded reports an input that rendered. Text output says nothing about it:
// the HTML itself is the result.
func (r reporter) succeeded(file, output string) {
	if !r.json {
		return
	}

	r.write(jsonRecord{File: displayName(file), Output: output, OK: true})
}

// write emits one JSON record per line. Encoding a record of strings and string
// slices cannot fail, so its error would have nothing to add. HTML escaping is
// off: the messages quote element names, and "\u003cmj-foo\u003e" is harder to
// read than the text it stands for.
func (r reporter) write(record jsonRecord) {
	encoder := json.NewEncoder(r.w)
	encoder.SetEscapeHTML(false)

	_ = encoder.Encode(record)
}

// jsonRecord is one input's result in JSON output.
type jsonRecord struct {
	File   string     `json:"file"`
	Output string     `json:"output,omitempty"`
	OK     bool       `json:"ok"`
	Error  *jsonError `json:"error,omitempty"`
}

// jsonError is the structured form of a failure. Kind is stable and machine
// readable; Message is the same text the command prints.
type jsonError struct {
	Kind      string   `json:"kind"`
	Message   string   `json:"message"`
	Path      []string `json:"path,omitempty"`
	Attribute string   `json:"attribute,omitempty"`
	Value     string   `json:"value,omitempty"`
}

// describeError flattens an error into the JSON shape, keeping the structured
// fields when the renderer reported one.
func describeError(err error) *jsonError {
	described := &jsonError{Kind: "error", Message: err.Error()}

	var structured *mjmlgo.Error
	if errors.As(err, &structured) {
		described.Kind = kindSlug(structured.Kind)
		described.Path = structured.Path
		described.Attribute = structured.Attribute
		described.Value = structured.Value
	}

	return described
}

// kindSlug is the stable name of an error kind, so JSON consumers never depend
// on the human wording of the message.
func kindSlug(kind mjmlgo.ErrorKind) string {
	switch kind {
	case mjmlgo.KindUnknownElement:
		return "unknown-element"
	case mjmlgo.KindUnknownAttribute:
		return "unknown-attribute"
	case mjmlgo.KindInvalidValue:
		return "invalid-value"
	case mjmlgo.KindDisallowedChild:
		return "disallowed-child"
	case mjmlgo.KindUnknownIncludeType:
		return "unknown-include-type"
	case mjmlgo.KindUndefinedMJClass:
		return "undefined-mj-class"
	case mjmlgo.KindParseFailure:
		return "parse-failure"
	default:
		return "error"
	}
}

// displayName is the name a record reports. A document read from standard input
// has no path, and the reference CLI's "undefined" is not worth reproducing.
func displayName(file string) string {
	if file == "" {
		return "<stdin>"
	}

	return file
}
