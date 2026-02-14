package gitbranch

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Branch represents a local git branch with optional remote tracking info.
type Branch struct {
	Name      string
	Upstream  string
	IsCurrent bool
	HasRemote bool
}

func getBranches() ([]Branch, error) {
	cmd := exec.Command("git", "for-each-ref",
		"--format=%(refname:short)|%(upstream:short)|%(HEAD)",
		"refs/heads/")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("not a git repository or git not found")
	}

	var branches []Branch
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		if len(parts) != 3 {
			continue
		}
		b := Branch{
			Name:      parts[0],
			Upstream:  parts[1],
			IsCurrent: parts[2] == "*",
			HasRemote: parts[1] != "",
		}
		branches = append(branches, b)
	}
	return branches, nil
}

func renameBranch(oldName, newName string) error {
	var buf bytes.Buffer
	cmd := exec.Command("git", "branch", "-m", oldName, newName)
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("rename branch: %s", strings.TrimSpace(buf.String()))
	}
	return nil
}

func renameRemoteBranch(remoteName, oldBranch, newBranch string) error {
	var buf bytes.Buffer

	delCmd := exec.Command("git", "push", remoteName, "--delete", oldBranch)
	delCmd.Stderr = &buf
	if err := delCmd.Run(); err != nil {
		return fmt.Errorf("delete remote branch: %s", strings.TrimSpace(buf.String()))
	}

	buf.Reset()
	pushCmd := exec.Command("git", "push", "--set-upstream", remoteName, newBranch)
	pushCmd.Stderr = &buf
	if err := pushCmd.Run(); err != nil {
		return fmt.Errorf("push new branch: %s", strings.TrimSpace(buf.String()))
	}

	return nil
}
