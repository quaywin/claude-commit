# cc (Claude Commit)

**Current Version:** v1.0.6

A CLI tool that uses Claude Code (Haiku) to review your code changes, generate commit messages, and push to your repository.

## Features
- **Automated Review**: Uses Claude Haiku to find bugs and security risks before you commit.
- **Auto-Commit Messages**: Generates professional commit messages based on your diff.
- **Untracked File Support**: Automatically detects and includes new, untracked files in the review and commit.
- **Clean History**: Automatically forbids Claude from adding "Co-Authored-By" or other attribution trailers to your commits.
- **One-Step Workflow**: Handles `git add`, `git commit`, and `git push` in one go.
- **Plan Mode**: Optional confirmation mode to review before committing.

## Installation

### Via Curl (Recommended)
```bash
curl -fsSL https://raw.githubusercontent.com/quaywin/claude-commit/main/install.sh | bash
```

### From Source
```bash
git clone https://github.com/quaywin/claude-commit.git
cd claude-commit
./install.sh
```

## Usage

### Basic Usage

**Quick mode (auto-commit):**
```bash
cc
```
Automatically reviews, generates commit message, and pushes without confirmation.

**Plan mode (with confirmation):**
```bash
cc plan
```
Shows the commit message and asks for your confirmation before committing and pushing.

**Force commit (bypass warnings):**
```bash
cc --force
# or
cc -f
```
Proceeds with the commit and push even if Claude identifies potential issues in your code.

#### Quick Mode Example:
```
🔍 Checking for changes...
🤖 Claude is reviewing your changes ⠋ [main.go] (1/3 files)
```
(Animated spinner cycling through each changed file with progress counter)
```
🤖 Claude is reviewing your changes... ✅

📝 Commit message: feat: add user authentication with JWT tokens

🚀 Staging all changes...
💾 Committing...
📤 Pushing...

✨ Done! Your changes have been reviewed, committed, and pushed.
```

#### Plan Mode Example:
```
🔍 Checking for changes...
🤖 Claude is reviewing your changes ⠋ [main.go] (1/3 files)
```
(Animated spinner cycling through each changed file with progress counter)
```
🤖 Claude is reviewing your changes... ✅

📝 Commit message: feat: add user authentication with JWT tokens

❓ Do you want to commit and push these changes? (y/n): y
🚀 Staging all changes...
💾 Committing...
📤 Pushing...

✨ Done! Your changes have been reviewed, committed, and pushed.
```

### Model Selection
Check or change the Claude model used for reviews:
```bash
cc models
```
This will show the current model and allow you to select from available options (Haiku, Sonnet, Opus, etc.).

### Version Management
Check your current version:
```bash
cc version
# or
cc --version
```

Update to the latest version:
```bash
cc update
```

## Requirements
- [Claude Code CLI](https://github.com/anthropics/claude-code) installed and authenticated
- (Optional) [Go](https://go.dev/) (only if building from source)

## Development & Releasing

To create a new release (build binaries, generate changelog, and upload to GitHub):

1. Ensure you have the GitHub CLI installed: `brew install gh`
2. Run the release script:
```bash
./release.sh v1.0.1
```
Or with custom notes:
```bash
./release.sh v1.0.1 "Major performance improvements"
```
