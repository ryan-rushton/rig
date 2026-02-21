package testchanged

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// TestRunner abstracts test discovery and execution for a build system.
type TestRunner interface {
	Name() string
	Detect() bool
	FindTargets(files []string) []string
	RunTests(targets []string) *exec.Cmd
}

// GoRunner discovers and runs Go tests.
type GoRunner struct{}

func (GoRunner) Name() string { return "go" }

func (GoRunner) Detect() bool {
	_, err := os.Stat("go.mod")
	return err == nil
}

// FindTargets maps changed .go files to unique Go package paths (./pkg/...).
func (GoRunner) FindTargets(files []string) []string {
	seen := make(map[string]struct{})
	for _, f := range files {
		if !strings.HasSuffix(f, ".go") {
			continue
		}
		dir := filepath.Dir(f)
		if dir == "." {
			dir = "./..."
		} else {
			dir = "./" + dir + "/..."
		}
		seen[dir] = struct{}{}
	}

	targets := make([]string, 0, len(seen))
	for t := range seen {
		targets = append(targets, t)
	}
	sort.Strings(targets)
	return targets
}

func (GoRunner) RunTests(targets []string) *exec.Cmd {
	args := append([]string{"test", "-v"}, targets...)
	return exec.Command("go", args...)
}

// BazelRunner discovers and runs Bazel tests.
type BazelRunner struct{}

func (BazelRunner) Name() string { return "bazel" }

func (BazelRunner) Detect() bool {
	for _, f := range []string{"BUILD.bazel", "WORKSPACE", "WORKSPACE.bazel", "MODULE.bazel"} {
		if _, err := os.Stat(f); err == nil {
			return true
		}
	}
	return false
}

// FindTargets uses bazel query to find test targets affected by changed files.
func (BazelRunner) FindTargets(files []string) []string {
	if len(files) == 0 {
		return nil
	}

	fileSet := strings.Join(files, " ")
	query := "kind('.*_test', rdeps(//..., set(" + fileSet + ")))"

	out, err := exec.Command("bazel", "query", query, "--output=label").Output()
	if err != nil {
		return nil
	}

	var targets []string
	for line := range strings.SplitSeq(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			targets = append(targets, line)
		}
	}
	sort.Strings(targets)
	return targets
}

func (BazelRunner) RunTests(targets []string) *exec.Cmd {
	args := append([]string{"test"}, targets...)
	return exec.Command("bazel", args...)
}

// allRunners returns all registered runners.
func allRunners() []TestRunner {
	return []TestRunner{GoRunner{}, BazelRunner{}}
}
