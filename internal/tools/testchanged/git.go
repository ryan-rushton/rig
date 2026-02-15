package testchanged

import (
	"os/exec"
	"strings"
)

// detectDefaultBranch returns the default branch name (e.g. "main" or "master")
// by inspecting the remote HEAD symbolic ref.
func detectDefaultBranch() (string, error) {
	out, err := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD").Output()
	if err != nil {
		return "", err
	}
	// "refs/remotes/origin/main" → "main"
	ref := strings.TrimSpace(string(out))
	parts := strings.SplitN(ref, "refs/remotes/origin/", 2)
	if len(parts) == 2 {
		return parts[1], nil
	}
	return ref, nil
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
