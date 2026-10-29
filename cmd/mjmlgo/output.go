package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// expandInputs expands the file patterns of the command line, keeping the order
// the user wrote them in. A pattern that matches nothing contributes nothing;
// whether the document exists is settled when it is read.
func expandInputs(patterns []string) ([]string, error) {
	var files []string

	for _, pattern := range patterns {
		if !strings.ContainsAny(pattern, "*?[") {
			files = append(files, pattern)

			continue
		}

		matches, err := glob(pattern)
		if err != nil {
			return nil, fmt.Errorf("cannot expand %s: %w", pattern, err)
		}

		files = append(files, matches...)
	}

	return files, nil
}

// glob expands one pattern. filepath.Glob matches a single level, so the "**"
// that the reference CLI's glob understands (any number of directories) is
// matched by walking instead.
func glob(pattern string) ([]string, error) {
	if !strings.Contains(pattern, "**") {
		return filepath.Glob(pattern)
	}

	return globRecursive(pattern)
}

// globRecursive matches a pattern containing "**" against every file below the
// pattern's literal prefix.
func globRecursive(pattern string) ([]string, error) {
	segments := strings.Split(filepath.ToSlash(pattern), "/")

	root, rest := ".", segments

	for i, segment := range segments {
		if strings.ContainsAny(segment, "*?[") {
			// Join by hand: an absolute pattern starts with an empty segment,
			// which filepath.Join would reduce to the working directory.
			root = strings.Join(segments[:i], "/")
			if root == "" {
				root = "."
			}

			rest = segments[i:]

			break
		}

		if i == len(segments)-1 {
			// No metacharacter at all: nothing to walk.
			return []string{pattern}, nil
		}
	}

	var matches []string

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}

		if matchSegments(rest, strings.Split(filepath.ToSlash(rel), "/")) {
			matches = append(matches, path)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(matches)

	return matches, nil
}

// matchSegments matches path segments against pattern segments, where "**"
// stands for any number of them.
func matchSegments(pattern, path []string) bool {
	if len(pattern) == 0 {
		return len(path) == 0
	}

	if pattern[0] == "**" {
		for skip := 0; skip <= len(path); skip++ {
			if matchSegments(pattern[1:], path[skip:]) {
				return true
			}
		}

		return false
	}

	if len(path) == 0 {
		return false
	}

	ok, err := filepath.Match(pattern[0], path[0])
	if err != nil || !ok {
		return false
	}

	return matchSegments(pattern[1:], path[1:])
}

// resolveOutputs maps every input to the file it should be written to, applying
// the reference CLI's rules: an existing directory collects one .html file per
// input, an empty path puts them in the working directory, and otherwise the
// single input takes the path as given.
func resolveOutputs(output string, inputs []input) ([]string, error) {
	if len(inputs) == 1 && inputs[0].file == "" {
		// Standard input has no name to derive one from.
		if output == "" || isDir(output) || strings.HasSuffix(output, string(filepath.Separator)) {
			return nil, msgStdinNeedsFile
		}

		return []string{output}, nil
	}

	switch {
	case len(inputs) > 1 && output != "" && !isDir(output):
		return nil, usageError(fmt.Sprintf(
			"Multiple input files, but output option should be either an existing directory or an empty string: %s given",
			output,
		))
	case len(inputs) == 1 && output != "" && !isDir(output):
		// A single input may name the file it is written to, but the CLI does
		// not create the directory that file lives in.
		if strings.HasSuffix(output, string(filepath.Separator)) || !isDir(filepath.Dir(output)) {
			return nil, usageError(fmt.Sprintf("Output directory doesn't exist for path : %s", output))
		}
	}

	targets := make([]string, len(inputs))

	for i, in := range inputs {
		targets[i] = outputPath(output, in.file)
	}

	return targets, nil
}

// outputPath is the file one input is written to.
func outputPath(output, file string) string {
	if output == "" || isDir(output) {
		return filepath.Join(output, htmlName(file))
	}

	return output
}

// htmlName is the output file name for an input path: its base name with the
// .mjml extension replaced by .html.
func htmlName(input string) string {
	base := filepath.Base(input)

	if ext := filepath.Ext(base); strings.EqualFold(ext, ".mjml") {
		base = strings.TrimSuffix(base, ext)
	}

	return base + ".html"
}

// isDir reports whether path names an existing directory.
func isDir(path string) bool {
	info, err := os.Stat(path)

	return err == nil && info.IsDir()
}
