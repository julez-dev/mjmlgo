package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/julez-dev/mjmlgo"
	"github.com/stretchr/testify/require"
)

func TestReporterText(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer

	report := reporter{w: &out}
	report.failed("a.mjml", errors.New("boom"))
	report.succeeded("b.mjml", "b.html")

	require.Equal(t, "File: a.mjml\nboom\n", out.String())
}

func TestReporterJSON(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer

	report := reporter{json: true, w: &out}
	report.failed("a.mjml", &mjmlgo.Error{
		Kind:      mjmlgo.KindUnknownElement,
		Path:      []string{"mj-body", "mj-foo"},
		Attribute: "",
		Value:     "mj-foo",
	})
	report.succeeded("b.mjml", "b.html")
	report.failed("", errors.New("plain failure"))

	lines := bytes.Split(bytes.TrimSuffix(out.Bytes(), []byte("\n")), []byte("\n"))
	require.Len(t, lines, 3)

	var structured jsonRecord
	require.NoError(t, json.Unmarshal(lines[0], &structured))
	require.Equal(t, "a.mjml", structured.File)
	require.False(t, structured.OK)
	require.Equal(t, "unknown-element", structured.Error.Kind)
	require.Equal(t, []string{"mj-body", "mj-foo"}, structured.Error.Path)
	require.Equal(t, "mj-foo", structured.Error.Value)

	var ok jsonRecord
	require.NoError(t, json.Unmarshal(lines[1], &ok))
	require.True(t, ok.OK)
	require.Equal(t, "b.html", ok.Output)
	require.Nil(t, ok.Error)

	var plain jsonRecord
	require.NoError(t, json.Unmarshal(lines[2], &plain))
	require.Equal(t, "error", plain.Error.Kind, "an unstructured failure still has a kind")
	require.Equal(t, "<stdin>", plain.File)
}

func TestReporterJSONKeepsTextReadable(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer

	reporter{json: true, w: &out}.failed("a.mjml", errors.New("unknown element <mj-foo>"))

	require.Contains(t, out.String(), "<mj-foo>")
	require.NotContains(t, out.String(), `\u003c`)
}

func TestKindSlug(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		kind mjmlgo.ErrorKind
		want string
	}{
		{mjmlgo.KindUnknownElement, "unknown-element"},
		{mjmlgo.KindUnknownAttribute, "unknown-attribute"},
		{mjmlgo.KindInvalidValue, "invalid-value"},
		{mjmlgo.KindDisallowedChild, "disallowed-child"},
		{mjmlgo.KindUnknownIncludeType, "unknown-include-type"},
		{mjmlgo.KindUndefinedMJClass, "undefined-mj-class"},
		{mjmlgo.KindParseFailure, "parse-failure"},
		{"something new", "error"},
	} {
		require.Equal(t, tc.want, kindSlug(tc.kind))
	}
}

func TestDescribeErrorWrapped(t *testing.T) {
	t.Parallel()

	// A structured error stays reachable through wrapping, so its fields survive
	// the trip to JSON.
	wrapped := errors.Join(errors.New("extra detail"), &mjmlgo.Error{
		Kind:      mjmlgo.KindInvalidValue,
		Attribute: "width",
		Value:     "wide",
	})

	described := describeError(wrapped)
	require.Equal(t, "invalid-value", described.Kind)
	require.Equal(t, "width", described.Attribute)
	require.Equal(t, "wide", described.Value)
}
