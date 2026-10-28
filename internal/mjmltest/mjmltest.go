// Package mjmltest contains the test harness that renders MJML input with the
// reference JavaScript implementation (the `mjml` npm package) so that the Go
// implementation in this repository can be diffed against it.
//
// The reference MJML version is configured project wide in package.json
// (devDependencies.mjml). It can be overridden for a single run with the
// MJML_VERSION environment variable, or bypassed entirely with MJML_BIN.
package mjmltest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	// VersionEnvVar overrides the MJML version pinned in package.json.
	// Example: MJML_VERSION=4.15.3 go test ./...
	VersionEnvVar = "MJML_VERSION"

	// BinEnvVar points at an existing MJML CLI executable. When set, no
	// version resolution happens and no package is downloaded.
	// Example: MJML_BIN=./node_modules/.bin/mjml go test ./...
	BinEnvVar = "MJML_BIN"

	// SkipEnvVar disables the reference comparison entirely (the tests are
	// skipped instead of failed). Useful when Node.js is unavailable.
	SkipEnvVar = "MJML_SKIP"
)

// renderTimeout bounds a single reference render.
const renderTimeout = 5 * time.Minute

// CLI is a resolved reference MJML implementation.
type CLI struct {
	// argv is the command prefix, e.g. ["./node_modules/.bin/mjml"] or
	// ["npx", "--yes", "mjml@4.18.0"].
	argv []string

	// Version is the MJML version reported by the resolved CLI.
	Version string

	// Source describes how the CLI was resolved, for logging purposes.
	Source string
}

var (
	rootOnce sync.Once
	rootDir  string
	rootErr  error
)

// RepoRoot returns the module root directory (the one holding package.json).
func RepoRoot(t testing.TB) string {
	t.Helper()

	rootOnce.Do(func() {
		dir, err := os.Getwd()
		if err != nil {
			rootErr = err
			return
		}

		for {
			if _, err := os.Stat(filepath.Join(dir, "package.json")); err == nil {
				rootDir = dir
				return
			}

			parent := filepath.Dir(dir)
			if parent == dir {
				rootErr = fmt.Errorf("no package.json found above %q", dir)
				return
			}

			dir = parent
		}
	})

	if rootErr != nil {
		t.Fatalf("locate repository root: %v", rootErr)
	}

	return rootDir
}

// PinnedVersion returns the MJML version declared in package.json.
func PinnedVersion(t testing.TB) string {
	t.Helper()

	path := filepath.Join(RepoRoot(t), "package.json")

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	var pkg struct {
		DevDependencies map[string]string `json:"devDependencies"`
		Dependencies    map[string]string `json:"dependencies"`
	}

	if err := json.Unmarshal(raw, &pkg); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	version := pkg.DevDependencies["mjml"]
	if version == "" {
		version = pkg.Dependencies["mjml"]
	}

	if version == "" {
		t.Fatalf("%s does not pin an mjml version (devDependencies.mjml)", path)
	}

	return cleanVersion(version)
}

// cleanVersion turns a semver range into a bare version string.
func cleanVersion(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "^")
	v = strings.TrimPrefix(v, "~")
	v = strings.TrimPrefix(v, ">=")
	v = strings.TrimPrefix(v, "=")
	v = strings.TrimPrefix(v, "v")

	return v
}

// NewCLI resolves the reference MJML CLI to use. It fails the test when the
// configuration is unusable and skips it when Node.js tooling is unavailable.
func NewCLI(t testing.TB) *CLI {
	t.Helper()

	if os.Getenv(SkipEnvVar) != "" {
		t.Skipf("%s is set, skipping comparison against the reference MJML implementation", SkipEnvVar)
	}

	root := RepoRoot(t)
	pinned := PinnedVersion(t)

	desired := pinned
	if v := cleanVersion(os.Getenv(VersionEnvVar)); v != "" {
		desired = v
	}

	var cli *CLI

	local := localCLI(root)
	npxPath, npxErr := exec.LookPath("npx")

	switch {
	case os.Getenv(BinEnvVar) != "":
		bin := os.Getenv(BinEnvVar)
		if _, err := exec.LookPath(bin); err != nil {
			t.Fatalf("%s=%q is not executable: %v", BinEnvVar, bin, err)
		}
		cli = &CLI{argv: []string{bin}, Source: BinEnvVar}
	case local != "" && localVersion(root) == desired:
		cli = &CLI{
			argv:   []string{local},
			Source: fmt.Sprintf("node_modules (pinned in package.json: %s)", desired),
		}
	case npxErr == nil:
		cli = &CLI{
			argv:   []string{npxPath, "--yes", "mjml@" + desired},
			Source: fmt.Sprintf("npx mjml@%s", desired),
		}
	default:
		t.Skipf(
			"reference MJML implementation unavailable: run `npm install` in %s (pinned version %s), "+
				"or install Node.js/npm, or set %s=1 to skip these comparisons",
			root, desired, SkipEnvVar,
		)
	}

	cli.Version = cli.detectVersion(t)

	if cli.Version != "" && desired != "" && cli.Version != desired {
		t.Fatalf(
			"reference MJML version mismatch: resolved %s (%s) but %s/%s asks for %s",
			cli.Version, cli.Source, VersionEnvVar, "package.json", desired,
		)
	}

	return cli
}

func localCLI(root string) string {
	path := filepath.Join(root, "node_modules", ".bin", "mjml")
	if runtime.GOOS == "windows" {
		path += ".cmd"
	}

	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return ""
	}

	return path
}

func localVersion(root string) string {
	raw, err := os.ReadFile(filepath.Join(root, "node_modules", "mjml", "package.json"))
	if err != nil {
		return ""
	}

	var pkg struct {
		Version string `json:"version"`
	}

	if err := json.Unmarshal(raw, &pkg); err != nil {
		return ""
	}

	return pkg.Version
}

var cliVersionRegexp = regexp.MustCompile(`mjml-core:\s*(\S+)`)

// detectVersion asks the CLI for its version. An empty string means the version
// could not be determined.
func (c *CLI) detectVersion(t testing.TB) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), renderTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.argv[0], append(c.argv[1:], "--version")...)
	cmd.Dir = RepoRoot(t)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Logf("could not determine reference MJML version (%v): %s", err, strings.TrimSpace(stderr.String()))
		return ""
	}

	out := stdout.String() + stderr.String()

	if m := cliVersionRegexp.FindStringSubmatch(out); m != nil {
		return m[1]
	}

	return ""
}

// RenderAll renders each of the given .mjml files with the reference MJML
// implementation and returns the resulting HTML keyed by file path.
//
// All files are rendered with a single CLI invocation where possible, because
// starting Node.js once per fixture dominates the runtime.
func (c *CLI) RenderAll(t testing.TB, files []string) map[string]string {
	t.Helper()

	if len(files) == 0 {
		return map[string]string{}
	}

	// Group by output basename: the CLI writes <basename>.html into the output
	// directory, so identically named fixtures must be rendered separately.
	groups := make(map[string][]string)
	for _, file := range files {
		base := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		groups[base] = append(groups[base], file)
	}

	unique := make([]string, 0, len(files))
	var duplicates []string

	for _, group := range groups {
		if len(group) == 1 {
			unique = append(unique, group[0])
			continue
		}

		sort.Strings(group)
		duplicates = append(duplicates, group...)
	}

	sort.Strings(unique)

	out := make(map[string]string, len(files))

	if len(unique) > 0 {
		for file, html := range c.renderBatch(t, unique) {
			out[file] = html
		}
	}

	for _, file := range duplicates {
		for f, html := range c.renderBatch(t, []string{file}) {
			out[f] = html
		}
	}

	return out
}

func (c *CLI) renderBatch(t testing.TB, files []string) map[string]string {
	t.Helper()

	outDir := t.TempDir()

	args := append([]string{}, c.argv[1:]...)
	absFiles := make([]string, len(files))

	for i, file := range files {
		abs, err := filepath.Abs(file)
		if err != nil {
			t.Fatalf("resolve fixture path %s: %v", file, err)
		}

		absFiles[i] = abs
	}

	args = append(args, absFiles...)
	args = append(args, "-o", outDir)

	ctx, cancel := context.WithTimeout(context.Background(), renderTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, c.argv[0], args...)
	cmd.Dir = RepoRoot(t)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			t.Fatalf("reference MJML render timed out after %s for %v", renderTimeout, files)
		}

		t.Fatalf(
			"reference MJML render failed for %v: %v\nstdout:\n%s\nstderr:\n%s",
			files, err, stdout.String(), stderr.String(),
		)
	}

	result := make(map[string]string, len(files))

	for _, file := range files {
		base := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		htmlPath := filepath.Join(outDir, base+".html")

		raw, err := os.ReadFile(htmlPath)
		if err != nil {
			t.Fatalf("reference MJML produced no output for %s: %v", file, err)
		}

		result[file] = string(raw)
	}

	return result
}
