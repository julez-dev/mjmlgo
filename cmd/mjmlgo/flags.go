package main

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// configPair is one --config assignment, kept in command-line order.
type configPair struct {
	key   string
	value string
}

// flags is the parsed command line.
type flags struct {
	// input is the stream -i reads from. It is not a flag value: run fills it
	// in from the process's standard input.
	input io.Reader

	read          []string
	validate      []string
	stdin         bool
	stdout        bool
	output        string
	outputSet     bool
	noFileComment bool
	json          bool
	help          bool
	version       bool
	config        []configPair
}

// parseFlags parses args the way the reference CLI's option parser does: long
// and short spellings, "--option=value" as well as "--option value", repeated
// array options, and "--" to end option parsing.
func parseFlags(args []string) (*flags, error) {
	f := &flags{}

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if arg == "--" {
			// Everything after the terminator is an input file, even if it
			// looks like an option.
			f.read = append(f.read, args[i+1:]...)

			return f, nil
		}

		if !isOption(arg) {
			// Positional arguments are read inputs, the same as -r, so the
			// order the user wrote them in is kept.
			f.read = append(f.read, arg)

			continue
		}

		name, value, hasValue := splitOption(arg)

		// --config.fonts=... names the key after the dot.
		if base, key, ok := strings.Cut(name, "."); ok && canonical(base) == "config" {
			pair, err := configPairOf(key, value, hasValue)
			if err != nil {
				return nil, err
			}

			f.config = append(f.config, pair)

			continue
		}

		switch canonical(name) {
		case "read", "validate":
			values := []string{}

			if hasValue {
				values = append(values, value)
			} else {
				for ; i+1 < len(args) && !isOption(args[i+1]); i++ {
					values = append(values, args[i+1])
				}
			}

			if len(values) == 0 {
				return nil, fmt.Errorf("option %s needs at least one value", arg)
			}

			if canonical(name) == "read" {
				f.read = append(f.read, values...)
			} else {
				f.validate = append(f.validate, values...)
			}
		case "output":
			if !hasValue {
				if i+1 >= len(args) || isOption(args[i+1]) {
					return nil, fmt.Errorf("option %s needs a value", arg)
				}

				i++
				value = args[i]
			}

			f.output, f.outputSet = value, true
		case "config":
			if !hasValue {
				if i+1 >= len(args) || isOption(args[i+1]) {
					return nil, fmt.Errorf("option %s needs a key=value", arg)
				}

				i++
				value = args[i]
			}

			pair, err := splitConfigAssignment(value)
			if err != nil {
				return nil, err
			}

			f.config = append(f.config, pair)
		case "stdin", "stdout", "noStdoutFileComment", "json", "help", "version":
			on, err := boolOption(name, value, hasValue)
			if err != nil {
				return nil, err
			}

			switch canonical(name) {
			case "stdin":
				f.stdin = on
			case "stdout":
				f.stdout = on
			case "noStdoutFileComment":
				f.noFileComment = on
			case "json":
				f.json = on
			case "help":
				f.help = on
			case "version":
				f.version = on
			}
		default:
			return nil, fmt.Errorf("unknown option %q", arg)
		}
	}

	return f, nil
}

// selectInputs picks the one input option that was used.
func selectInputs(f *flags) (patterns []string, fromStdin bool, validate bool, err error) {
	groups := 0

	if len(f.read) > 0 {
		groups++
	}

	if len(f.validate) > 0 {
		groups++
	}

	if f.stdin {
		groups++
	}

	switch {
	case groups == 0:
		return nil, false, false, msgNoInputArgument
	case groups > 1:
		return nil, false, false, msgTooManyInputs
	case len(f.validate) > 0:
		return f.validate, false, true, nil
	case f.stdin:
		return nil, true, false, nil
	default:
		return f.read, false, false, nil
	}
}

// selectOutputs picks the one output option that was used. Standard output is
// the default, like it is in the reference CLI.
func selectOutputs(f *flags) (toStdout bool, output string, err error) {
	if f.stdout && f.outputSet {
		return false, "", msgTooManyOutputs
	}

	if f.outputSet {
		return false, f.output, nil
	}

	return true, "", nil
}

// isOption reports whether arg is an option rather than a file name. A lone "-"
// is a file name.
func isOption(arg string) bool {
	return len(arg) > 1 && arg[0] == '-'
}

// splitOption splits an option into its name and its inline value. Both
// "--output=x" and "-o=x" are accepted, as are the space-separated spellings.
func splitOption(arg string) (name, value string, hasValue bool) {
	name = strings.TrimLeft(arg, "-")

	if before, after, ok := strings.Cut(name, "="); ok {
		return before, after, true
	}

	return name, "", false
}

// canonical maps an option's spellings to one name. Case matters: -v validates
// and -V prints the version.
func canonical(name string) string {
	switch name {
	case "r", "read":
		return "read"
	case "v", "validate":
		return "validate"
	case "i", "stdin":
		return "stdin"
	case "s", "stdout":
		return "stdout"
	case "o", "output":
		return "output"
	case "c", "config":
		return "config"
	case "h", "help":
		return "help"
	case "V", "version":
		return "version"
	case "noStdoutFileComment":
		return "noStdoutFileComment"
	case "json":
		return "json"
	}

	return ""
}

// boolOption reads a switch, which may still carry an explicit value
// ("--json=false").
func boolOption(name, value string, hasValue bool) (bool, error) {
	if !hasValue {
		return true, nil
	}

	on, err := strconv.ParseBool(value)
	if err != nil {
		return false, fmt.Errorf("option %s: %q is not a boolean", name, value)
	}

	return on, nil
}

// configPairOf builds one assignment from the value side of --config.<key>.
func configPairOf(key, value string, hasValue bool) (configPair, error) {
	if key == "" {
		return configPair{}, errors.New("config: missing key name after \"--config.\"")
	}

	if !hasValue {
		return configPair{}, fmt.Errorf("config.%s needs a value", key)
	}

	return configPair{key: key, value: value}, nil
}

// splitConfigAssignment splits one "key=value" assignment.
func splitConfigAssignment(assignment string) (configPair, error) {
	key, value, ok := strings.Cut(assignment, "=")
	if !ok || key == "" {
		return configPair{}, fmt.Errorf("config: %q is not a key=value assignment", assignment)
	}

	return configPair{key: key, value: value}, nil
}
