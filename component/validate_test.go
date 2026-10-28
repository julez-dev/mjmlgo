package component

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateEnum(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		valid   []string
		value   string
		wantErr bool
	}{
		{name: "member", valid: []string{"ltr", "rtl"}, value: "ltr"},
		{name: "not a member", valid: []string{"ltr", "rtl"}, value: "what", wantErr: true},
		{name: "empty is not a member", valid: []string{"ltr", "rtl"}, value: "", wantErr: true},
		{name: "empty is a member when listed", valid: []string{"full-width", "false", ""}, value: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := validateEnum(tt.valid)
			if tt.wantErr {
				require.Error(t, f(tt.value))
				return
			}

			require.NoError(t, f(tt.value))
		})
	}
}

func TestValidateColor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{name: "less than three digits", value: "#FF", wantErr: true},
		{name: "more than six digits", value: "#FFFFFFF", wantErr: true},
		{name: "six digits", value: "#FFFFFF"},
		{name: "three digits", value: "#abc"},
		{name: "rgb", value: "rgb(1, 2, 3)"},
		{name: "rgba", value: "rgba(1,2,3,0.5)"},
		{name: "named", value: "Chartreuse"},
		{name: "empty", value: "", wantErr: true},
		{name: "no hash prefix", value: "FFFFF", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateColor()(tt.value)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestValidateUnit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		validator AttributeValidator
		value     string
		wantErr   bool
		want      string
	}{
		{name: "single px", validator: validateUnit([]string{"px"}, false), value: "10px"},
		{name: "zero without unit", validator: validateUnit([]string{"px"}, false), value: "0"},
		{name: "wrong unit", validator: validateUnit([]string{"px"}, false), value: "10em", wantErr: true, want: `unit "em" from value "10em" is not one of [px]`},
		{name: "negative is rejected", validator: validateUnit([]string{"px"}, false), value: "-10px", wantErr: true, want: `negative value "-10px" is not allowed`},
		{name: "negative is allowed by unitWithNegative", validator: validateUnitWithNegative([]string{"px"}, false), value: "-10px"},
		{name: "negative em is allowed by unitWithNegative", validator: validateUnitWithNegative([]string{"px", "em"}, false), value: "-1.5em"},
		{name: "empty", validator: validateUnit([]string{"px"}, false), value: "", wantErr: true, want: "empty value is not a valid unit"},
		{name: "auto without auto unit", validator: validateUnit([]string{"px"}, false), value: "auto", wantErr: true},
		{name: "auto with auto unit", validator: validateUnit([]string{"px", "auto"}, false), value: "auto"},
		{name: "shorthand one value", validator: validateUnit([]string{"px", "%"}, true), value: "10px"},
		{name: "shorthand four values", validator: validateUnit([]string{"px", "%"}, true), value: "10px 20% 30px 40%"},
		{name: "shorthand five values", validator: validateUnit([]string{"px", "%"}, true), value: "1px 2px 3px 4px 5px", wantErr: true, want: `"1px 2px 3px 4px 5px" has 5 value(s), want 1 to 4`},
		{name: "shorthand rejects auto", validator: validateUnit([]string{"px", "auto"}, true), value: "auto auto", wantErr: true},
		{name: "line-height without unit", validator: validateUnit([]string{"px", "%", ""}, false), value: "1.5"},
		{name: "line-height with em", validator: validateUnit([]string{"px", "%", ""}, false), value: "1em", wantErr: true},
		// The value scanner reads the sign, number and unit in one pass, so the
		// edge cases of MJML's permissive number part stay pinned here.
		{name: "leading space is an empty value", validator: validateUnit([]string{"px"}, true), value: " 10px", wantErr: true, want: `empty value in " 10px"`},
		{name: "trailing space is an empty value", validator: validateUnit([]string{"px"}, false), value: "10px ", wantErr: true},
		{name: "double space is an empty value", validator: validateUnit([]string{"px"}, true), value: "10px  20px", wantErr: true},
		{name: "unit without number", validator: validateUnit([]string{"px"}, false), value: "px", wantErr: true, want: "unable to parse CSS unit from: px"},
		{name: "sign without number", validator: validateUnitWithNegative([]string{"px"}, false), value: "-px", wantErr: true},
		{name: "character after unit", validator: validateUnit([]string{"px"}, false), value: "10px2", wantErr: true},
		{name: "dots and commas form the number", validator: validateUnit([]string{"px"}, false), value: "1.2.3px"},
		{name: "unit matching is case sensitive", validator: validateUnit([]string{"px"}, false), value: "10PX", wantErr: true},
		{name: "negative shorthand value", validator: validateUnitWithNegative([]string{"px"}, true), value: "-1px -2px"},
		{name: "negative shorthand value is rejected by unit", validator: validateUnit([]string{"px"}, true), value: "10px -2px", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.validator(tt.value)
			if tt.want != "" {
				require.EqualError(t, err, tt.want)
				return
			}

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestValidateInteger(t *testing.T) {
	t.Parallel()

	tests := []struct {
		value   string
		wantErr bool
	}{
		{value: "0"},
		{value: "12"},
		// MJML's integer matcher is an unanchored /\d+/, so any value with a
		// digit passes; we keep that permissiveness rather than being stricter.
		{value: "abc12"},
		{value: "abc", wantErr: true},
		{value: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			t.Parallel()

			err := integerValue(tt.value)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestValidateType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		expectedType string
		value        string
		wantErr      bool
	}{
		{name: "string accepts empty", expectedType: "string", value: ""},
		{name: "boolean true", expectedType: "boolean", value: "true"},
		{name: "boolean is case insensitive", expectedType: "boolean", value: "TRUE"},
		{name: "boolean false", expectedType: "boolean", value: "False"},
		{name: "boolean empty", expectedType: "boolean", value: "", wantErr: true},
		{name: "boolean invalid", expectedType: "boolean", value: "yes", wantErr: true},
		{name: "number", expectedType: "number", value: "1.5"},
		{name: "number empty", expectedType: "number", value: "", wantErr: true},
		{name: "number invalid", expectedType: "number", value: "1px", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateType(tt.expectedType)(tt.value)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestValidateColorAcceptsCSSWideKeywords(t *testing.T) {
	t.Parallel()

	validate := validateColor()

	for _, value := range []string{"transparent", "inherit", "TRANSPARENT", "#abc", "#AABBCC", "rgb(1,2,3)", "rgba(1,2,3,0.5)", "rebeccapurple"} {
		if err := validate(value); err != nil {
			t.Errorf("validateColor(%q) = %v, want nil", value, err)
		}
	}

	for _, value := range []string{"notacolor", "#ab", "rgb(1,2)"} {
		if err := validate(value); err == nil {
			t.Errorf("validateColor(%q) = nil, want error", value)
		}
	}
}
