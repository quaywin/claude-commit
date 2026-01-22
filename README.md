# Claude Commit CLI

A CLI tool that uses Claude Code (Haiku) to review your code changes, generate commit messages, and push to your repository.

## Features
- **Automated Review**: Uses Claude Haiku to find bugs and security risks before you commit.
- **Auto-Commit Messages**: Generates professional commit messages based on your diff.
- **One-Step Workflow**: Handles `git add`, `git commit`, and `git push` in one go.

## Installation
```bash
./install.sh
```

## Usage
Run the following command in any git repository:
```bash
claude-commit
```

## Requirements
- Go installed
- [Claude Code CLI](https://github.com/anthropics/claude-code) installed and authenticated
