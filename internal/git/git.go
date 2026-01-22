package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// GetDiff returns the combined diff of staged and unstaged changes
func GetDiff() (string, error) {
	// Get unstaged changes
	unstaged, err := runGitCommand("diff")
	if err != nil {
		return "", err
	}

	// Get staged changes
	staged, err := runGitCommand("diff", "--cached")
	if err != nil {
		return "", err
	}

	if unstaged == "" && staged == "" {
		return "", nil
	}

	return fmt.Sprintf("--- UNSTAGED CHANGES ---\n%s\n--- STAGED CHANGES ---\n%s", unstaged, staged), nil
}

// StageAll stages all changes in the repository
func StageAll() error {
	_, err := runGitCommand("add", ".")
	return err
}

// Commit creates a commit with the given message
func Commit(message string) error {
	_, err := runGitCommand("commit", "-m", message)
	return err
}

// Push pushes the current branch to the remote
func Push() error {
	_, err := runGitCommand("push")
	return err
}

func runGitCommand(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("git command failed: %w, stderr: %s", err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}
