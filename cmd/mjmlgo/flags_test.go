package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseFlagsReadsFilesInOrder(t *testing.T) {
	t.Parallel()

	// Positional arguments are read inputs, the same as -r, and keep the order
	// they were written in.
	fl, err := parseFlags([]string{"a.mjml", "-r", "b.mjml", "c.mjml", "--read=d.mjml", "e.mjml"})
	require.NoError(t, err)
	require.Equal(t, []string{"a.mjml", "b.mjml", "c.mjml", "d.mjml", "e.mjml"}, fl.read)
}

func TestParseFlagsAliases(t *testing.T) {
	t.Parallel()

	fl, err := parseFlags([]string{"-i"})
	require.NoError(t, err)
	require.True(t, fl.stdin)

	fl, err = parseFlags([]string{"--stdin"})
	require.NoError(t, err)
	require.True(t, fl.stdin)

	fl, err = parseFlags([]string{"-s"})
	require.NoError(t, err)
	require.True(t, fl.stdout)

	fl, err = parseFlags([]string{"-o", "out.html"})
	require.NoError(t, err)
	require.True(t, fl.outputSet)
	require.Equal(t, "out.html", fl.output)

	fl, err = parseFlags([]string{"--output=out.html"})
	require.NoError(t, err)
	require.Equal(t, "out.html", fl.output)

	fl, err = parseFlags([]string{"--noStdoutFileComment", "--json"})
	require.NoError(t, err)
	require.True(t, fl.noFileComment)
	require.True(t, fl.json)

	// -v validates, -V prints the version: the case is the whole difference.
	fl, err = parseFlags([]string{"-v", "a.mjml"})
	require.NoError(t, err)
	require.Equal(t, []string{"a.mjml"}, fl.validate)

	fl, err = parseFlags([]string{"-V"})
	require.NoError(t, err)
	require.True(t, fl.version)
}

func TestParseFlagsArrayOptions(t *testing.T) {
	t.Parallel()

	fl, err := parseFlags([]string{"-r", "a.mjml", "b.mjml", "-i"})
	require.NoError(t, err)
	require.Equal(t, []string{"a.mjml", "b.mjml"}, fl.read)
	require.True(t, fl.stdin)

	fl, err = parseFlags([]string{"--validate", "a.mjml", "b.mjml", "c.mjml"})
	require.NoError(t, err)
	require.Equal(t, []string{"a.mjml", "b.mjml", "c.mjml"}, fl.validate)

	_, err = parseFlags([]string{"-r"})
	require.Error(t, err)
}

func TestParseFlagsConfigSpellings(t *testing.T) {
	t.Parallel()

	want := []configPair{
		{key: "keepComments", value: "false"},
		{key: "fonts", value: `{"a":"b"}`},
	}

	fl, err := parseFlags([]string{"--config.keepComments=false", "--config", `fonts={"a":"b"}`})
	require.NoError(t, err)
	require.Equal(t, want, fl.config)

	fl, err = parseFlags([]string{"-c", "keepComments=false", "-c", `fonts={"a":"b"}`})
	require.NoError(t, err)
	require.Equal(t, want, fl.config)

	_, err = parseFlags([]string{"--config.keepComments"})
	require.Error(t, err)

	_, err = parseFlags([]string{"-c", "keepComments"})
	require.Error(t, err)
}

func TestParseFlagsTerminator(t *testing.T) {
	t.Parallel()

	// After -- nothing is an option any more, so an option-looking file name can
	// still be read.
	fl, err := parseFlags([]string{"-s", "--", "-weird.mjml", "--also.mjml"})
	require.NoError(t, err)
	require.True(t, fl.stdout)
	require.Equal(t, []string{"-weird.mjml", "--also.mjml"}, fl.read)
}

func TestParseFlagsBoolValues(t *testing.T) {
	t.Parallel()

	fl, err := parseFlags([]string{"--json=false", "--stdout=true"})
	require.NoError(t, err)
	require.False(t, fl.json)
	require.True(t, fl.stdout)

	_, err = parseFlags([]string{"--json=maybe"})
	require.Error(t, err)
}

func TestParseFlagsRejectsBadOptions(t *testing.T) {
	t.Parallel()

	for _, args := range [][]string{
		{"--frobnicate"},
		{"-x"},
		{"-o"},
		{"--config"},
		{"-r", "--json"},
	} {
		_, err := parseFlags(args)
		require.Error(t, err, "args: %v", args)
	}
}

func TestSelectInputs(t *testing.T) {
	t.Parallel()

	patterns, fromStdin, validate, err := selectInputs(&flags{read: []string{"a.mjml"}})
	require.NoError(t, err)
	require.Equal(t, []string{"a.mjml"}, patterns)
	require.False(t, fromStdin)
	require.False(t, validate)

	_, fromStdin, _, err = selectInputs(&flags{stdin: true})
	require.NoError(t, err)
	require.True(t, fromStdin)

	patterns, _, validate, err = selectInputs(&flags{validate: []string{"a.mjml"}})
	require.NoError(t, err)
	require.Equal(t, []string{"a.mjml"}, patterns)
	require.True(t, validate)

	_, _, _, err = selectInputs(&flags{})
	require.ErrorContains(t, err, "No input argument received")

	_, _, _, err = selectInputs(&flags{stdin: true, validate: []string{"a.mjml"}})
	require.ErrorContains(t, err, "Too many input arguments received")

	_, _, _, err = selectInputs(&flags{read: []string{"a.mjml"}, stdin: true})
	require.ErrorContains(t, err, "Too many input arguments received")
}

func TestSelectOutputs(t *testing.T) {
	t.Parallel()

	toStdout, output, err := selectOutputs(&flags{})
	require.NoError(t, err)
	require.True(t, toStdout)
	require.Empty(t, output, "standard output has no path")

	toStdout, output, err = selectOutputs(&flags{outputSet: true, output: "out.html"}) //nolint:staticcheck // output is asserted below
	require.NoError(t, err)
	require.False(t, toStdout)
	require.Equal(t, "out.html", output)

	_, _, err = selectOutputs(&flags{stdout: true, outputSet: true})
	require.ErrorContains(t, err, "Too many output arguments received")
}
