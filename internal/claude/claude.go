package claude

import (
	"bytes"
	"fmt"
	"os/exec"
)

// ReviewAndCommitMessage takes a git diff and returns a suggested commit message or an error if issues are found.
func ReviewAndCommitMessage(diff string) (string, error) {
	if diff == "" {
		return "", fmt.Errorf("no changes detected")
	}

	prompt := fmt.Sprintf(`Review the following git diff for any issues (bugs, security risks, style).
If there are critical issues, start your response with "ISSUE: " followed by the description.
If the code looks good, provide ONLY a concise, professional commit message.

Diff:
%s`, diff)

	// We use the 'haiku' model as requested, and '-p' for non-interactive output.
	cmd := exec.Command("claude", "--model", "haiku", "-p", prompt)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("claude command failed: %w, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}
