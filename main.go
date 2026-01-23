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
	"strings"
	"sync"
	"time"

	"github.com/quaywin/claude-commit/internal/claude"
	"github.com/quaywin/claude-commit/internal/git"
)

const VERSION = "v1.0.3"

func main() {
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

	// Check if plan mode (with confirmation)
	planMode := len(os.Args) > 1 && os.Args[1] == "plan"

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

	// Get list of changed files
	changedFiles, err := git.GetChangedFiles()
	if err != nil {
		fmt.Printf("❌ Error getting changed files: %v\n", err)
		os.Exit(1)
	}

	// 2. Call Claude for review and commit message
	fmt.Print("🤖 Claude is reviewing your changes")

	// Start spinner animation
	var wg sync.WaitGroup
	stopSpinner := make(chan bool)
	wg.Add(1)
	go func() {
		defer wg.Done()
		spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		i := 0
		fileIndex := 0
		for {
			select {
			case <-stopSpinner:
				fmt.Print("\r🤖 Claude is reviewing your changes... ✅\n")
				return
			default:
				currentFile := ""
				fileCount := ""
				if len(changedFiles) > 0 {
					currentFileIndex := fileIndex % len(changedFiles)
					currentFile = fmt.Sprintf(" [%s]", changedFiles[currentFileIndex])
					fileCount = fmt.Sprintf(" (%d/%d files)", currentFileIndex+1, len(changedFiles))
				}
				fmt.Printf("\r🤖 Claude is reviewing your changes %s%s%s ", spinner[i%len(spinner)], currentFile, fileCount)

				// Clear to end of line to handle varying file name lengths
				fmt.Print("\033[K")

				i++
				if i%3 == 0 { // Change file every 3 spinner frames
					fileIndex++
				}
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	result, err := claude.ReviewAndCommitMessage(diff, nil)

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
		fmt.Println("\nPlease fix these issues before committing.")
		os.Exit(1)
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

	fmt.Println("📤 Pushing...")
	if err := git.Push(); err != nil {
		fmt.Printf("❌ Error pushing: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n✨ Done! Your changes have been reviewed, committed, and pushed.")
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
	if err := os.Rename(tmpPath, exePath); err != nil {
		// If rename fails (cross-device link), try copy
		if err := copyFile(tmpPath, exePath); err != nil {
			fmt.Printf("❌ Error replacing binary: %v\n", err)
			fmt.Println("💡 You may need to run with sudo: sudo cc update")
			os.Exit(1)
		}
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
