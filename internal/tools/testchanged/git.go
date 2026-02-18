package testchanged

import (
	"fmt"
	"os/exec"
	"strings"
)

// detectDefaultBranch returns the default branch name (e.g. "main" or "master")
// by checking which common default branches exist locally as remote-tracking refs.
func detectDefaultBranch() (string, error) {
	for _, branch := range []string{"main", "master"} {
		if exec.Command("git", "rev-parse", "--verify", "origin/"+branch).Run() == nil {
			return branch, nil
		}
	}
	return "", fmt.Errorf("could not find origin/main or origin/master")
}

// mergeBase returns the best common ancestor between HEAD and the given branch.
func mergeBase(branch string) (string, error) {
	out, err := exec.Command("git", "merge-base", "HEAD", "origin/"+branch).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// changedFiles returns all files changed compared to the merge base,
// combining committed, unstaged, and staged changes with deduplication.
func changedFiles(base string) ([]string, error) {
	seen := make(map[string]struct{})
	var result []string

	add := func(files []string) {
		for _, f := range files {
			f = strings.TrimSpace(f)
			if f == "" {
				continue
			}
			if _, ok := seen[f]; !ok {
				seen[f] = struct{}{}
				result = append(result, f)
			}
		}
	}

	// Committed changes since merge base.
	out, err := exec.Command("git", "diff", "--name-only", base).Output()
	if err != nil {
		return nil, err
	}
	add(strings.Split(string(out), "\n"))

	// Unstaged changes.
	out, err = exec.Command("git", "diff", "--name-only").Output()
	if err != nil {
		return nil, err
	}
	add(strings.Split(string(out), "\n"))

	// Staged changes.
	out, err = exec.Command("git", "diff", "--name-only", "--cached").Output()
	if err != nil {
		return nil, err
	}
	add(strings.Split(string(out), "\n"))

	return result, nil
}
