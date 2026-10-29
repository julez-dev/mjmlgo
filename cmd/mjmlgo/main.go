// Command mjmlgo compiles MJML documents to HTML.
//
// The command surface follows the reference mjml CLI: files (or glob patterns)
// are read with -r or as positional arguments, -i compiles standard input, HTML
// goes to standard output until -o names a file or directory, -v validates
// without writing, and -c passes renderer configuration. Two differences are
// deliberate. Output is always compact, because that is the only form this
// implementation emits; the reference CLI beautifies standard output by default.
// Validation failures abort a render instead of only warning, so a document the
// reference CLI renders with warnings on stderr fails here.
//
// Exit codes match the reference CLI: 0 for success, 1 for a command-line or
// validation error, 2 when some input failed to render.
//
// Install it with
//
//	go install github.com/julez-dev/mjmlgo/cmd/mjmlgo@latest
package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/julez-dev/mjmlgo"
)

// Exit codes, matching the reference CLI's contract.
const (
	exitSuccess = 0
	exitUsage   = 1
	exitRender  = 2
)

// stdinName is the synthetic file name a document read from standard input is
// known by, so mj-include can resolve its relative paths.
const stdinName = "stdin.mjml"

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// run is the whole command line: it parses args, renders what they name and
// returns the process exit code. Every write goes through the given streams so
// that tests can drive it like the process would.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fl, err := parseFlags(args)
	if err != nil {
		return commandLineError(stderr, err)
	}

	fl.input = stdin

	if fl.help {
		writeUsage(stdout)

		return exitSuccess
	}

	if fl.version {
		fmt.Fprintf(stdout, "mjmlgo %s\n", buildVersion())

		return exitSuccess
	}

	patterns, fromStdin, validate, err := selectInputs(fl)
	if err != nil {
		return commandLineError(stderr, err)
	}

	toStdout, output, err := selectOutputs(fl)
	if err != nil {
		return commandLineError(stderr, err)
	}

	cfg, err := parseConfig(fl.config)
	if err != nil {
		return commandLineError(stderr, err)
	}

	ctx := context.Background()
	renderer := mjmlgo.NewRenderer(cfg.options...)

	if validate {
		// Validation writes no HTML, so a JSON report can own stdout; the text
		// report goes to stderr, where the reference CLI's validator puts it.
		report := stderr
		if fl.json {
			report = stdout
		}

		return validateInputs(ctx, renderer, cfg, patterns, reporter{json: fl.json, w: report}, stderr)
	}

	return compileInputs(ctx, renderer, cfg, patterns, fromStdin, fl, toStdout, output, stdout, stderr)
}

// compileInputs renders every input and writes the HTML where the output option
// asked for it.
func compileInputs(
	ctx context.Context,
	renderer *mjmlgo.Renderer,
	cfg renderConfig,
	patterns []string,
	fromStdin bool,
	fl *flags,
	toStdout bool,
	output string,
	stdout, stderr io.Writer,
) int {
	inputs, err := readInputs(patterns, fromStdin, fl)
	if err != nil {
		return commandLineError(stderr, err)
	}

	var targets []string

	if !toStdout {
		targets, err = resolveOutputs(output, inputs)
		if err != nil {
			return commandLineError(stderr, err)
		}
	}

	// The report is a side channel: HTML owns stdout, so results go to stderr.
	report := reporter{json: fl.json, w: stderr}

	failed := 0

	for i, in := range inputs {
		target := ""

		if !toStdout {
			target = targets[i]
		}

		var html bytes.Buffer

		if err := renderInput(ctx, renderer, cfg, in, &html); err != nil {
			failed++

			report.failed(in.file, err)

			continue
		}

		if toStdout {
			if err := writeStdout(stdout, in.file, html.Bytes(), fl.noFileComment); err != nil {
				failed++

				report.failed(in.file, err)

				continue
			}

			report.succeeded(in.file, "")

			continue
		}

		if err := os.WriteFile(target, html.Bytes(), 0o644); err != nil {
			failed++

			report.failed(in.file, usageError(fmt.Sprintf("Error writing file - %s : %v", target, err)))

			continue
		}

		report.succeeded(in.file, target)
	}

	switch {
	case failed == len(inputs):
		// The reference CLI reports a run where nothing rendered as a
		// command-line error rather than a render failure.
		return commandLineError(stderr, msgRenderFailed)
	case failed > 0:
		return exitRender
	default:
		return exitSuccess
	}
}

// validateInputs renders every input for its errors only and reports whether all
// of them passed.
func validateInputs(
	ctx context.Context,
	renderer *mjmlgo.Renderer,
	cfg renderConfig,
	patterns []string,
	report reporter,
	stderr io.Writer,
) int {
	files, err := expandInputs(patterns)
	if err != nil {
		return commandLineError(stderr, err)
	}

	if len(files) == 0 {
		return commandLineError(stderr, msgNoInputFiles)
	}

	invalid := 0

	for _, file := range files {
		if err := renderInput(ctx, renderer, cfg, input{file: file}, io.Discard); err != nil {
			invalid++

			report.failed(file, err)

			continue
		}

		report.succeeded(file, "")
	}

	if invalid > 0 {
		return commandLineError(stderr, msgValidationFailed)
	}

	return exitSuccess
}

// input is one document to render: a file path, or the bytes of a document read
// from standard input.
type input struct {
	file string
	src  []byte
}

// readInputs turns the selected input option into the documents to render. The
// file patterns are expanded here so that output paths are resolved against the
// same list.
func readInputs(patterns []string, fromStdin bool, fl *flags) ([]input, error) {
	if fromStdin {
		src, err := io.ReadAll(fl.input)
		if err != nil {
			return nil, fmt.Errorf("cannot read standard input: %w", err)
		}

		return []input{{src: src}}, nil
	}

	files, err := expandInputs(patterns)
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, msgNoInputFiles
	}

	inputs := make([]input, len(files))
	for i, file := range files {
		inputs[i] = input{file: file}
	}

	return inputs, nil
}

// renderInput renders one input into w. The document is read where it lives;
// mj-include paths resolve against the configured root, or against the
// document's own directory when none was given.
func renderInput(ctx context.Context, renderer *mjmlgo.Renderer, cfg renderConfig, in input, w io.Writer) error {
	file, src := in.file, in.src

	if file == "" {
		// A document from standard input answers to a synthetic name, so that
		// its includes have somewhere to resolve from.
		file = stdinName
	} else {
		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("cannot read file: %w", err)
		}

		src = content
	}

	root := cfg.root
	if root == "" {
		root = filepath.Dir(file)
	}

	name := filepath.Base(file)

	return renderer.RenderFS(ctx, w, documentFS{name: name, src: src, dir: root}, name)
}

// writeStdout writes the compiled document with the file header the reference
// CLI prints unless --noStdoutFileComment was given.
func writeStdout(w io.Writer, file string, html []byte, noFileComment bool) error {
	var b bytes.Buffer

	// A document from standard input has no file name to announce.
	if !noFileComment && file != "" {
		fmt.Fprintf(&b, "<!-- FILE: %s -->\n", file)
	}

	b.Write(html)
	b.WriteByte('\n')

	_, err := w.Write(b.Bytes())

	return err
}

// usageError is a message for the user worded the way the reference CLI words
// it, capitalization included. It is an error so that the command line has one
// failure shape, but it is not a diagnostic: it is output, and scripts match on
// it.
type usageError string

func (e usageError) Error() string { return string(e) }

// The reference CLI's own wording.
const (
	msgNoInputArgument  = usageError("No input argument received")
	msgTooManyInputs    = usageError("Too many input arguments received")
	msgTooManyOutputs   = usageError("Too many output arguments received")
	msgNoInputFiles     = usageError("No input files found")
	msgRenderFailed     = usageError("Input file(s) failed to render")
	msgValidationFailed = usageError("Validation failed")
	msgStdinNeedsFile   = usageError("Output option must name a file when reading from standard input")
)

// commandLineError reports a problem with the invocation the way the reference
// CLI does and returns the exit code that goes with it.
func commandLineError(w io.Writer, err error) int {
	fmt.Fprintf(w, "\nCommand line error:\n%s\n", err)

	return exitUsage
}

// buildVersion is the module version the binary was built from, or "devel" when
// it was built without one.
func buildVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "devel"
	}

	// A build from a checkout carries a pseudo-version ("v0.0.0-<time>-<hash>",
	// possibly with "+dirty"), which says more about the working tree than
	// about the release.
	version := info.Main.Version
	if !strings.HasPrefix(version, "v") || strings.ContainsAny(version, "-+") {
		return "devel"
	}

	return version
}

// writeUsage prints the option summary.
func writeUsage(w io.Writer) {
	fmt.Fprint(w, `Usage: mjmlgo [options] [files...]

Compiles MJML documents to compact HTML.

Input:
  -r, --read <file...>       Compile MJML file(s); glob patterns are expanded
  -i, --stdin                Compile MJML from standard input
  -v, --validate <file...>   Validate file(s) without writing output

Output:
  -s, --stdout               Write HTML to standard output (default)
  -o, --output <path>        Output file name, or an existing directory
      --noStdoutFileComment  Omit the "<!-- FILE: ... -->" header on stdout

Configuration:
  -c, --config <key=value>   Renderer option; also --config.<key>=<value>
                             keys: fonts, keepComments, filePath

Other:
      --json                 Report per-file results as newline-delimited JSON
  -h, --help                 Show this help
  -V, --version              Show the version

Exit codes: 0 success, 1 command-line or validation error, 2 render failure.
`)
}
