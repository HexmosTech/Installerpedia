package main

import (
	"encoding/json"
	"fmt"
	"ipm/tracker"
	"ipm/version"
	"net/http"
	"time"

	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func updateCommand() {
	// ---------------------------
	// DETECT PLATFORM / ARCH
	// ---------------------------
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	archMap := map[string]string{
		"amd64": "amd64",
		"arm64": "arm64",
		"386":   "386",
	}
	normArch, ok := archMap[goarch]
	if !ok {
		fmt.Println("❌ Unsupported architecture:", goarch)
		return
	}

	platformName := goos
	ext := ""
	if goos == "windows" {
		platformName = "windows"
		ext = ".exe"
	}

	assetFilename := fmt.Sprintf("ipm-%s-%s%s", platformName, normArch, ext)

	// ---------------------------
	// FETCH LATEST VERSION
	// ---------------------------
	client := &http.Client{Timeout: 10 * time.Second}
	req, _ := http.NewRequest("GET", "https://api.github.com/repos/HexmosTech/Installerpedia/releases/latest", nil)
	req.Header.Set("User-Agent", "ipm-updater") // GitHub requires this!

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("❌ ERROR: Failed to contact GitHub API:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("❌ ERROR: GitHub API returned status %d\n", resp.StatusCode)
		return
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		fmt.Println("❌ ERROR: Failed to parse GitHub response:", err)
		return
	}
	latest := release.TagName

	if latest == "" {
		fmt.Println("❌ ERROR: GitHub returned empty version. Something is wrong.")
		return
	}

	// ---------------------------
	// COMPARE VERSIONS
	// ---------------------------
	local := version.GetVersion()
	cmp := version.CompareVersions(local, latest)

	if cmp == 0 {
		fmt.Println("ipm version:", local)
		return
	}
	if cmp == 1 {
		fmt.Printf("⚠ Local version (%s) is newer than GitHub (%s). Skipping update.\n", local, latest)
		return
	}

	fmt.Printf("⬆ Update available (%s → %s)\n", local, latest)

	// ---------------------------
	// DOWNLOAD UPDATE
	// ---------------------------
	downloadURL := fmt.Sprintf(
		"https://github.com/HexmosTech/Installerpedia/releases/latest/download/%s",
		assetFilename,
	)

	var installPath string
	if goos == "windows" {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println("❌ ERROR: Cannot get home directory:", err)
			return
		}

		installDir := filepath.Join(home, ".local", "bin")
		installPath = filepath.Join(installDir, "ipm.exe")

		if err := os.MkdirAll(installDir, 0755); err != nil {
			fmt.Println("❌ ERROR: Failed to create install directory:", err)
			return
		}
	} else {
		installPath = "/usr/local/bin/ipm" // global binary path
	}

	tmpPath := installPath + ".new"

	fmt.Println("⬇ Downloading update from:", downloadURL)

	// ---------------------------
	// Download using curl with sudo if required
	// ---------------------------
	var curlCmd *exec.Cmd
	if goos == "windows" {
		curlCmd = exec.Command("curl", "-L", "--fail", downloadURL, "-o", tmpPath)
	} else {
		// Ensure /usr/local/bin exists
		exec.Command("sudo", "mkdir", "-p", "/usr/local/bin").Run()

		// Download using sudo
		curlCmd = exec.Command("sudo", "curl", "-L", "--fail", downloadURL, "-o", tmpPath)
	}
	curlCmd.Stdout = os.Stdout
	curlCmd.Stderr = os.Stderr

	if err := curlCmd.Run(); err != nil {
		fmt.Println("❌ ERROR: Download failed:", err)
		return
	}

	// ---------------------------
	// Apply chmod
	// ---------------------------
	if goos != "windows" {
		if err := exec.Command("sudo", "chmod", "755", tmpPath).Run(); err != nil {
			fmt.Println("❌ ERROR: Failed to chmod new binary:", err)
			return
		}
	} else {
		oldPath := installPath + ".old"
		_ = os.Remove(oldPath)

		// 1. Rename CURRENTLY RUNNING to .old
		if err := os.Rename(installPath, oldPath); err != nil {
			fmt.Println("❌ ERROR: Failed to prepare update (rename current):", err)
			return
		}

		// 2. Move NEW to original path
		if err := os.Rename(tmpPath, installPath); err != nil {
			fmt.Println("❌ ERROR: Failed to apply update (move new):", err)
			os.Rename(oldPath, installPath) // rollback
			return
		}

		_ = os.Remove(oldPath)
	}

	// ---------------------------
	// APPLY UPDATE
	// ---------------------------
	fmt.Println("🔧 Applying update...")

	if goos != "windows" {
		// Use sudo mv directly without wrapping in bash -c
		if err := exec.Command("sudo", "mv", tmpPath, installPath).Run(); err != nil {
			fmt.Println("❌ ERROR: Failed to apply update:", err)
			return
		}
	}

	fmt.Println("✔ Update installed successfully!")
	fmt.Println("👉 Restart your terminal or run `ipm` again.")
	tracker.TrackUpdateSuccess(local, latest)

}
