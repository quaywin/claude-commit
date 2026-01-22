package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/quaywin/claude-commit/internal/claude"
	"github.com/quaywin/claude-commit/internal/git"
)

func main() {
	fmt.Println("🔍 Checking for changes...")

	// 1. Get diff
	diff, err := git.GetDiff()
	if err != nil {
		fmt.Printf("❌ Error getting git diff: %v\n", err)
		os.Exit(1)
	}

	if diff == "" {
		fmt.Println("✅ No changes to commit.")
		return
	}

	// 2. Call Claude for review and commit message
	fmt.Println("🤖 Claude is reviewing your changes...")
	result, err := claude.ReviewAndCommitMessage(diff)
	if err != nil {
		fmt.Printf("❌ Error calling Claude: %v\n", err)
		os.Exit(1)
	}

	result = strings.TrimSpace(result)

	// 3. Check for issues
	if strings.HasPrefix(strings.ToUpper(result), "ISSUE:") {
		fmt.Println("\n⚠️  Claude found potential issues in your code:")
		fmt.Println(result)
		fmt.Println("\nPlease fix these issues before committing.")
		os.Exit(1)
	}

	// 4. Show commit message
	fmt.Printf("\n📝 Commit message: %s\n", result)

	// 5. Stage, Commit, and Push
	fmt.Println("🚀 Staging all changes...")
	if err := git.StageAll(); err != nil {
		fmt.Printf("❌ Error staging changes: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("💾 Committing...")
	if err := git.Commit(result); err != nil {
		fmt.Printf("❌ Error committing: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("📤 Pushing...")
	if err := git.Push(); err != nil {
		fmt.Printf("❌ Error pushing: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n✨ Done! Your changes have been reviewed, committed, and pushed.")
}
