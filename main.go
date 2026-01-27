package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/quaywin/claude-commit/internal/claude"
	"github.com/quaywin/claude-commit/internal/config"
	"github.com/quaywin/claude-commit/internal/git"
)

const VERSION = "v1.0.10"

func main() {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("⚠️  Warning: Could not load config: %v\n", err)
		cfg = &config.Config{Model: "haiku"}
	}

	// Handle version command
	if len(os.Args) > 1 && (os.Args[1] == "version" || os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("cc version %s\n", VERSION)
		return
	}

	// Handle update command
	if len(os.Args) > 1 && os.Args[1] == "update" {
		handleUpdate()
		return
	}

	// Handle models command
	if len(os.Args) > 1 && os.Args[1] == "models" {
		handleModels(cfg)
		return
	}

	// Check if plan mode (with confirmation)
	planMode := false
	forceMode := false
	noPush := false
	for _, arg := range os.Args[1:] {
		switch arg {
		case "plan":
			planMode = true
		case "--force", "-f":
			forceMode = true
		case "--no-push":
			noPush = true
		case "version", "--version", "-v", "update", "models":
			// These are handled by early returns at the beginning of main()
			// but we include them here to avoid "Unknown parameter" errors
			continue
		default:
			fmt.Printf("❌ Error: Unknown parameter: %s\n", arg)
			fmt.Println("Usage: cc [plan] [--force|-f] [--no-push] [version|--version|-v] [update] [models]")
			os.Exit(1)
		}
	}

	fmt.Println("🔍 Checking for changes...")

	// 1. Get changed files and determine mode
	changedFiles, err := git.GetChangedFiles()
	if err != nil {
		fmt.Printf("❌ Error getting changed files: %v\n", err)
		os.Exit(1)
	}

	if len(changedFiles) == 0 {
		fmt.Println("✅ No changes to commit.")
		return
	}

	fileCount := len(changedFiles)
	useSummaryMode := fileCount >= git.FileSummaryThreshold

	// 2. Get appropriate diff
	var diff string
	if useSummaryMode {
		diff, err = git.GetDiffSummary()
		if err != nil {
			fmt.Printf("❌ Error getting git diff summary: %v\n", err)
			os.Exit(1)
		}
	} else {
		diff, err = git.GetDiff()
		if err != nil {
			fmt.Printf("❌ Error getting git diff: %v\n", err)
			os.Exit(1)
		}
	}

	if diff == "" {
		fmt.Println("✅ No changes to commit.")
		return
	}

	// 3. Call Claude for review and commit message
	fmt.Print("🤖 Claude is reviewing your changes")

	// Start spinner animation
	var wg sync.WaitGroup
	stopSpinner := make(chan bool)
	wg.Add(1)
	go func() {
		defer wg.Done()
		spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		i := 0

		fileCountText := ""
		if fileCount > 0 {
			modeText := ""
			if useSummaryMode {
				modeText = ", summary mode"
			}
			fileCountText = fmt.Sprintf(" (%d files%s)", fileCount, modeText)
		}

		for {
			select {
			case <-stopSpinner:
				fmt.Print("\r🤖 Claude is reviewing your changes... ✅\n")
				return
			default:
				fmt.Printf("\r🤖 Claude is reviewing your changes%s %s ", fileCountText, spinner[i%len(spinner)])

				// Clear to end of line
				fmt.Print("\033[K")

				i++
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	result, err := claude.ReviewAndCommitMessage(diff, cfg.Model, useSummaryMode, nil)

	// Stop spinner
	stopSpinner <- true
	wg.Wait()

	if err != nil {
		fmt.Printf("❌ Error calling Claude: %v\n", err)
		os.Exit(1)
	}

	result = strings.TrimSpace(result)

	// 3. Check for issues
	if strings.HasPrefix(strings.ToUpper(result), "ISSUE:") {
		fmt.Println("\n⚠️  Claude found potential issues in your code:")
		fmt.Println(result)

		if !forceMode {
			fmt.Println("\nPlease fix these issues before committing. Use --force or -f to commit anyway.")
			os.Exit(1)
		} else {
			fmt.Println("\n⚠️  Force mode enabled. Proceeding with commit despite issues.")
			// Remove the ISSUE: prefix for the commit message if we're forcing
			lines := strings.Split(result, "\n")
			if len(lines) > 0 {
				// Try to find a line that doesn't start with ISSUE: or use a default message
				// Usually, Claude output for ISSUE: looks like:
				// ISSUE: <description>
				// Suggested message: <message>
				foundMessage := false
				for _, line := range lines {
					if strings.HasPrefix(strings.ToLower(line), "suggested message:") || strings.HasPrefix(strings.ToLower(line), "commit message:") {
						result = strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
						foundMessage = true
						break
					}
				}
				if !foundMessage {
					result = "chore: commit despite potential issues"
				}
			}
		}
	}

	// 4. Show commit message
	fmt.Printf("\n📝 Commit message: %s\n", result)

	// 5. Ask for confirmation (only in plan mode)
	if planMode {
		fmt.Print("\n❓ Do you want to commit and push these changes? (y/n): ")
		reader := bufio.NewReader(os.Stdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("❌ Error reading input: %v\n", err)
			os.Exit(1)
		}

		response = strings.ToLower(strings.TrimSpace(response))
		if response != "y" && response != "yes" {
			fmt.Println("❌ Aborted. No changes were committed.")
			os.Exit(0)
		}
	}

	// 6. Stage, Commit, and Push
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

	if !noPush {
		fmt.Println("📤 Pushing...")
		if err := git.Push(); err != nil {
			fmt.Printf("❌ Error pushing: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("\n✨ Done! Your changes have been reviewed, committed, and pushed.")
	} else {
		fmt.Println("\n✨ Done! Your changes have been reviewed and committed (not pushed).")
	}
}

func handleModels(cfg *config.Config) {
	models := []string{
		"haiku",
		"sonnet",
		"opus",
	}

	fmt.Printf("Current model: %s\n", cfg.Model)
	fmt.Println("\nSelect a model:")

	// Check if current model is in the list
	found := false
	for i, m := range models {
		prefix := "  "
		if m == cfg.Model {
			prefix = "* "
			found = true
		}
		fmt.Printf("%s%d. %s\n", prefix, i+1, m)
	}

	// Add custom option
	customIdx := len(models) + 1
	prefix := "  "
	if !found {
		prefix = "* "
	}
	fmt.Printf("%s%d. Custom...\n", prefix, customIdx)

	fmt.Print("\nEnter number to select (or press Enter to keep current): ")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return
	}

	idx, err := strconv.Atoi(input)
	if err != nil || idx < 1 || idx > customIdx {
		fmt.Println("❌ Invalid selection")
		return
	}

	if idx == customIdx {
		fmt.Print("Enter custom model name: ")
		customInput, _ := reader.ReadString('\n')
		customInput = strings.TrimSpace(customInput)
		if customInput == "" {
			fmt.Println("❌ Model name cannot be empty")
			return
		}
		cfg.Model = customInput
	} else {
		cfg.Model = models[idx-1]
	}

	if err := config.Save(cfg); err != nil {
		fmt.Printf("❌ Error saving config: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Model set to: %s\n", cfg.Model)
}

type GithubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func handleUpdate() {
	fmt.Println("🔍 Checking for updates...")

	// Fetch latest release from GitHub
	req, err := http.NewRequest("GET", "https://api.github.com/repos/quaywin/claude-commit/releases/latest", nil)
	if err != nil {
		fmt.Printf("❌ Error creating request: %v\n", err)
		os.Exit(1)
	}
	req.Header.Set("User-Agent", "cc-cli/"+VERSION)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ Error checking for updates: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("❌ Error fetching release info: HTTP %d\n", resp.StatusCode)
		os.Exit(1)
	}

	var release GithubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		fmt.Printf("❌ Error parsing release info: %v\n", err)
		os.Exit(1)
	}

	latestVersion := release.TagName
	if latestVersion == VERSION {
		fmt.Printf("✅ You're already on the latest version (%s)\n", VERSION)
		return
	}

	fmt.Printf("📦 New version available: %s (you have %s)\n", latestVersion, VERSION)

	// Determine OS and architecture
	osName := runtime.GOOS
	arch := runtime.GOARCH

	// Find matching binary
	binaryName := fmt.Sprintf("cc-%s-%s", osName, arch)
	if osName == "windows" {
		binaryName += ".exe"
	}
	var downloadURL string
	var checksumURL string
	for _, asset := range release.Assets {
		if asset.Name == binaryName {
			downloadURL = asset.BrowserDownloadURL
		}
		if asset.Name == "checksums.txt" {
			checksumURL = asset.BrowserDownloadURL
		}
	}

	if downloadURL == "" {
		fmt.Printf("❌ No binary found for %s/%s\n", osName, arch)
		os.Exit(1)
	}

	if checksumURL == "" {
		fmt.Println("⚠️  Warning: No checksums file found in release")
		fmt.Println("❌ Cannot verify download integrity. Aborting for security.")
		os.Exit(1)
	}

	// Download and parse checksums
	fmt.Println("🔐 Downloading checksums...")
	checksumResp, err := http.Get(checksumURL)
	if err != nil {
		fmt.Printf("❌ Error downloading checksums: %v\n", err)
		os.Exit(1)
	}
	defer checksumResp.Body.Close()

	checksumData, err := io.ReadAll(checksumResp.Body)
	if err != nil {
		fmt.Printf("❌ Error reading checksums: %v\n", err)
		os.Exit(1)
	}

	// Parse expected checksum
	var expectedChecksum string
	for _, line := range strings.Split(string(checksumData), "\n") {
		parts := strings.Fields(line)
		if len(parts) == 2 && parts[1] == binaryName {
			expectedChecksum = parts[0]
			break
		}
	}

	if expectedChecksum == "" {
		fmt.Printf("❌ No checksum found for %s\n", binaryName)
		os.Exit(1)
	}

	// Download new binary
	fmt.Printf("📥 Downloading %s...\n", binaryName)
	resp, err = http.Get(downloadURL)
	if err != nil {
		fmt.Printf("❌ Error downloading binary: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("❌ Error downloading binary: HTTP %d\n", resp.StatusCode)
		os.Exit(1)
	}

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "cc-update-*")
	if err != nil {
		fmt.Printf("❌ Error creating temporary file: %v\n", err)
		os.Exit(1)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	// Write downloaded binary to temp file
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		fmt.Printf("❌ Error saving binary: %v\n", err)
		os.Exit(1)
	}
	tmpFile.Close()

	// Verify checksum
	fmt.Println("🔐 Verifying checksum...")
	actualChecksum, err := calculateSHA256(tmpPath)
	if err != nil {
		fmt.Printf("❌ Error calculating checksum: %v\n", err)
		os.Exit(1)
	}

	if actualChecksum != expectedChecksum {
		fmt.Printf("❌ Checksum mismatch!\n")
		fmt.Printf("   Expected: %s\n", expectedChecksum)
		fmt.Printf("   Got:      %s\n", actualChecksum)
		fmt.Println("   The download may have been corrupted or tampered with.")
		os.Exit(1)
	}
	fmt.Println("✅ Checksum verified")

	// Make it executable
	if err := os.Chmod(tmpPath, 0755); err != nil {
		fmt.Printf("❌ Error setting permissions: %v\n", err)
		os.Exit(1)
	}

	// Get current executable path
	exePath, err := os.Executable()
	if err != nil {
		fmt.Printf("❌ Error finding current executable: %v\n", err)
		os.Exit(1)
	}

	// On Windows, we can't replace a running executable
	// Instead, we rename the old one and place the new one
	if runtime.GOOS == "windows" {
		backupPath := exePath + ".old"
		newPath := exePath + ".new"

		// Copy new binary to .new file
		if err := copyFile(tmpPath, newPath); err != nil {
			fmt.Printf("❌ Error copying new binary: %v\n", err)
			os.Exit(1)
		}

		// Create a batch script to complete the update after we exit
		batchScript := filepath.Join(filepath.Dir(exePath), "update.bat")
		batchContent := fmt.Sprintf(`@echo off
timeout /t 1 /nobreak >nul
move /y "%s" "%s" >nul 2>&1
move /y "%s" "%s" >nul 2>&1
del "%s" >nul 2>&1
del "%%~f0"
`, exePath, backupPath, newPath, exePath, backupPath)

		if err := os.WriteFile(batchScript, []byte(batchContent), 0755); err != nil {
			fmt.Printf("❌ Error creating update script: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Update to %s ready!\n", latestVersion)
		fmt.Println("🔄 Completing update... (this will restart cc)")

		// Execute the batch script and exit
		cmd := exec.Command("cmd", "/c", "start", "/b", batchScript)
		cmd.Start()
		os.Exit(0)
	}

	// Replace current binary (Unix-like systems)
	fmt.Println("🚚 Installing update...")

	// On Unix, we can't always rename/overwrite a running binary
	// The safest way is to rename the OLD binary and then put the NEW one in its place
	oldPath := exePath + ".old"
	if err := os.Rename(exePath, oldPath); err != nil {
		// If rename fails (might not have permission), try copy then rename
		if err := copyFile(tmpPath, exePath); err != nil {
			fmt.Printf("❌ Error replacing binary: %v\n", err)
			fmt.Println("💡 You may need to run with sudo: sudo cc update")
			os.Exit(1)
		}
	} else {
		// Moved old binary to .old, now move new binary to original path
		if err := os.Rename(tmpPath, exePath); err != nil {
			// If rename fails (might be cross-device link), try copy
			if err := copyFile(tmpPath, exePath); err != nil {
				// If copying new binary fails, try to restore old one
				os.Rename(oldPath, exePath)
				fmt.Printf("❌ Error installing new binary: %v\n", err)
				os.Exit(1)
			}
		}
		// Successfully installed new binary, remove the .old one
		os.Remove(oldPath)
	}

	fmt.Printf("✅ Updated to %s successfully!\n", latestVersion)
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	return os.Chmod(dst, 0755)
}

func calculateSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
