# cc (Claude Commit)

**Current Version:** v1.0.0

A CLI tool that uses Claude Code (Haiku) to review your code changes, generate commit messages, and push to your repository.

## Features
- **Automated Review**: Uses Claude Haiku to find bugs and security risks before you commit.
- **Auto-Commit Messages**: Generates professional commit messages based on your diff.
- **One-Step Workflow**: Handles `git add`, `git commit`, and `git push` in one go.

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
Run the following command in any git repository:
```bash
cc
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
