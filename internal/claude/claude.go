package claude

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
)

// ReviewAndCommitMessage takes a git diff and returns a suggested commit message or an error if issues are found.
// progressWriter can be provided to show real-time output from Claude.
func ReviewAndCommitMessage(diff string, model string, useSummaryMode bool, progressWriter io.Writer) (string, error) {
	if diff == "" {
		return "", fmt.Errorf("no changes detected")
	}

	var prompt string
	if useSummaryMode {
		prompt = fmt.Sprintf(`Review the following git diff summary showing changed files and line counts.
Since this is a large changeset (10+ files), you're seeing a summary rather than full diffs.

Focus on:
- Overall scope and impact of changes
- File naming and organizational patterns
- Scale of changes (large refactors vs small fixes)

If you notice concerning patterns (e.g., many files with massive changes suggesting risky refactoring),
start your response with "ISSUE: " followed by the concern.

Otherwise, provide a concise commit message following Conventional Commits specification.
Focus on the "why" and overall scope, not individual file details.

Diff Summary:
%s`, diff)
	} else {
		prompt = fmt.Sprintf(`Review the following git diff for any issues (bugs, security risks, style).
If there are critical issues, you MUST start your response with "ISSUE: " followed by the description.

If the code looks good, provide a concise, professional commit message.
Follow the Conventional Commits specification (e.g., feat: ..., fix: ..., chore: ...).
Focus on "why" the change was made, not just "what" changed.
Provide ONLY the commit message in one line. Do NOT include any "Co-Authored-By" trailers or attribution.

Diff:
%s`, diff)
	}

	// We use the specified model, and '-p' for non-interactive output.
	// We pass the prompt via stdin to avoid "argument list too long" errors for large diffs.
	cmd := exec.Command("claude", "--model", model, "-p")
	cmd.Stdin = bytes.NewReader([]byte(prompt))
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// If progressWriter is provided, also write stderr to it for progress updates
	if progressWriter != nil {
		cmd.Stderr = io.MultiWriter(&stderr, progressWriter)
	}

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("claude command failed: %w, stderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}
