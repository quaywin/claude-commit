package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

const FileSummaryThreshold = 10

// GetDiff returns the combined diff of staged, unstaged, and untracked changes
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

	// Get untracked changes
	untracked, err := runGitCommand("ls-files", "--others", "--exclude-standard")
	if err != nil {
		return "", err
	}

	untrackedDiff := ""
	if untracked != "" {
		for _, file := range strings.Split(untracked, "\n") {
			if file != "" {
				// Use git diff --no-index /dev/null <file> to show new file content
				// Note: git diff --no-index returns exit code 1 if there are differences
				cmd := exec.Command("git", "diff", "--no-index", "/dev/null", file)
				var stdout bytes.Buffer
				cmd.Stdout = &stdout
				_ = cmd.Run() // Ignore error as exit 1 is expected for differences
				diff := strings.TrimSpace(stdout.String())
				if diff != "" {
					untrackedDiff += diff + "\n"
				}
			}
		}
	}

	if unstaged == "" && staged == "" && untrackedDiff == "" {
		return "", nil
	}

	return fmt.Sprintf("--- UNSTAGED CHANGES ---\n%s\n--- STAGED CHANGES ---\n%s\n--- UNTRACKED FILES ---\n%s", unstaged, staged, untrackedDiff), nil
}

// GetDiffSummary returns a summary of changed files with line counts (for large changesets)
func GetDiffSummary() (string, error) {
	// Get unstaged changes summary
	unstaged, err := runGitCommand("diff", "--stat")
	if err != nil {
		return "", err
	}

	// Get staged changes summary
	staged, err := runGitCommand("diff", "--cached", "--stat")
	if err != nil {
		return "", err
	}

	// Get untracked files
	untracked, err := runGitCommand("ls-files", "--others", "--exclude-standard")
	if err != nil {
		return "", err
	}

	untrackedSummary := ""
	if untracked != "" {
		files := strings.Split(untracked, "\n")
		count := 0
		for _, f := range files {
			if f != "" {
				count++
			}
		}
		untrackedSummary = fmt.Sprintf("%d untracked files", count)
	}

	if unstaged == "" && staged == "" && untrackedSummary == "" {
		return "", nil
	}

	return fmt.Sprintf("--- UNSTAGED CHANGES ---\n%s\n--- STAGED CHANGES ---\n%s\n--- UNTRACKED FILES ---\n%s", unstaged, staged, untrackedSummary), nil
}

// GetChangedFiles returns a list of files that have been changed (staged, unstaged, and untracked)
func GetChangedFiles() ([]string, error) {
	// Get unstaged files
	unstaged, err := runGitCommand("diff", "--name-only")
	if err != nil {
		return nil, err
	}

	// Get staged files
	staged, err := runGitCommand("diff", "--cached", "--name-only")
	if err != nil {
		return nil, err
	}

	// Get untracked files
	untracked, err := runGitCommand("ls-files", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}

	// Combine and deduplicate
	filesMap := make(map[string]bool)
	for _, file := range strings.Split(unstaged, "\n") {
		if file != "" {
			filesMap[file] = true
		}
	}
	for _, file := range strings.Split(staged, "\n") {
		if file != "" {
			filesMap[file] = true
		}
	}
	for _, file := range strings.Split(untracked, "\n") {
		if file != "" {
			filesMap[file] = true
		}
	}

	var files []string
	for file := range filesMap {
		files = append(files, file)
	}

	return files, nil
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
