package component

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  *Error
		want string
	}{
		{
			name: "unknown element",
			err:  NewError(KindUnknownElement, []string{"mj-body", "mj-foo"}, "", "mj-foo", nil),
			want: "mj-body > mj-foo: unknown element <mj-foo>",
		},
		{
			name: "unknown attribute",
			err:  NewError(KindUnknownAttribute, []string{"mj-text"}, "made-up", "value", nil),
			want: `mj-text: attribute "made-up" is not allowed on <mj-text>`,
		},
		{
			name: "invalid value",
			err:  NewError(KindInvalidValue, []string{"mj-text"}, "color", "chartreuseish", errors.New("chartreuseish not a valid color string")),
			want: `mj-text: attribute "color" in <mj-text>: chartreuseish not a valid color string`,
		},
		{
			name: "disallowed child",
			err:  NewError(KindDisallowedChild, []string{"mj-body", "mj-text"}, "mj-text", "mj-text", errors.New("allowed children: mj-section")),
			want: "mj-body > mj-text: <mj-text> is not allowed inside <mj-body> (allowed children: mj-section)",
		},
		{
			name: "disallowed child of the document root",
			err:  NewError(KindDisallowedChild, []string{"mj-text"}, "", "mj-text", errors.New("allowed children: mj-body, mj-head, mj-raw")),
			want: "mj-text: <mj-text> is not allowed inside <mjml> (allowed children: mj-body, mj-head, mj-raw)",
		},
		{
			name: "undefined mj-class",
			err:  NewError(KindUndefinedMJClass, []string{"mj-text"}, "mj-class", "missing", nil),
			want: `mj-text: mj-class "missing" is not declared in <mj-attributes>`,
		},
		{
			name: "unknown include type",
			err:  NewError(KindUnknownIncludeType, []string{"mj-body", "mj-include"}, "", "js", errors.New("mjml: unknown mj-include type")),
			want: `mj-body > mj-include: unknown mj-include type "js" (want mjml, html or css)`,
		},
		{
			name: "parse failure",
			err:  NewError(KindParseFailure, []string{"mjml"}, "", "", errors.New("parsing MJML structure failed: no root element specified")),
			want: "mjml: parse failure: parsing MJML structure failed: no root element specified",
		},
		{
			name: "no path",
			err:  NewError(KindParseFailure, nil, "", "", errors.New("boom")),
			want: "parse failure: boom",
		},
		{
			name: "invalid component",
			err:  NewError(KindInvalidComponent, []string{MJMLTagName}, "", "mj-text", errors.New("tag <mj-text> is a built-in MJML element")),
			want: `mjml: invalid component "mj-text": tag <mj-text> is a built-in MJML element`,
		},
		{
			name: "invalid component without a tag",
			err:  NewError(KindInvalidComponent, []string{MJMLTagName}, "", "", errors.New("nil component")),
			want: "mjml: invalid component: nil component",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.err.Error())
		})
	}
}

func TestErrorUnwrap(t *testing.T) {
	t.Parallel()

	cause := errors.New("boom")
	err := NewError(KindInvalidValue, []string{"mj-text"}, "color", "x", cause)

	assert.ErrorIs(t, err, cause)
	assert.Same(t, cause, errors.Unwrap(err))
}

func TestErrorIsValidationSentinel(t *testing.T) {
	t.Parallel()

	wantValidationKinds := []ErrorKind{
		KindUnknownElement,
		KindUnknownAttribute,
		KindInvalidValue,
		KindDisallowedChild,
		KindUndefinedMJClass,
	}

	for _, kind := range wantValidationKinds {
		t.Run(string(kind), func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, NewError(kind, []string{"mj-text"}, "", "", nil), ErrValidation)
		})
	}

	for _, kind := range []ErrorKind{KindUnknownIncludeType, KindParseFailure, KindInvalidComponent} {
		t.Run(string(kind), func(t *testing.T) {
			t.Parallel()
			assert.NotErrorIs(t, NewError(kind, []string{"mjml"}, "", "", nil), ErrValidation)
		})
	}
}

func TestErrorAsThroughWrapping(t *testing.T) {
	t.Parallel()

	err := fmt.Errorf("outer: %w", NewError(KindUndefinedMJClass, []string{"mj-text"}, "mj-class", "missing", nil))

	var structured *Error
	require.ErrorAs(t, err, &structured)

	assert.Equal(t, KindUndefinedMJClass, structured.Kind)
	assert.Equal(t, []string{"mj-text"}, structured.Path)
	assert.Equal(t, "mj-class", structured.Attribute)
	assert.Equal(t, "missing", structured.Value)
}
