package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Ensure PATH contains ~/.local/bin (fallback for non-root installs)
func ensurePath() {
	home, _ := os.UserHomeDir()
	line := fmt.Sprintf(`export PATH="%s/.local/bin:$PATH"`, home)
	file := home + "/.bashrc"

	content, _ := os.ReadFile(file)
	if !strings.Contains(string(content), line) {
		f, _ := os.OpenFile(file, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		defer f.Close()
		f.WriteString("\n" + line + "\n")
		fmt.Println("Added ~/.local/bin to PATH in ~/.bashrc")
	}
}

// Multi-platform setup (Linux/mac/Windows)
func setupCommand() {
	fmt.Println("🔧 Setting up permanent ipm installation...")

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

	ext := ""
	var installPath string
	var tmpPath string
	var downloadDir string

	if goos == "windows" {
		ext = ".exe"
		home, _ := os.UserHomeDir()
		downloadDir = home + "/.local/bin"
		installPath = downloadDir + "/ipm" + ext
		tmpPath = installPath + ".new"
	} else {
		downloadDir = "/usr/local/bin"
		installPath = downloadDir + "/ipm"
		tmpPath = installPath + ".new"
	}

	// Ensure install directory exists
	if goos != "windows" {
		cmd := exec.Command("sudo", "mkdir", "-p", downloadDir)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	} else {
		os.MkdirAll(downloadDir, 0755)
	}

	assetFilename := fmt.Sprintf("ipm-%s-%s%s", goos, normArch, ext)
	downloadURL := fmt.Sprintf(
		"https://github.com/HexmosTech/Installerpedia/releases/latest/download/%s",
		assetFilename,
	)

	fmt.Println("⬇ Downloading latest binary:", assetFilename)
	cmd := exec.Command("curl", "-L", downloadURL, "-o", tmpPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("❌ Download failed:", err)
		return
	}

	if goos != "windows" {
		// Overwrite existing binary with sudo
		fmt.Println("🔧 Installing binary to /usr/local/bin with sudo...")
		cmd = exec.Command("sudo", "mv", "-f", tmpPath, installPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()

		cmd = exec.Command("sudo", "chmod", "+x", installPath)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	} else {
		os.Chmod(tmpPath, 0755)
		os.Rename(tmpPath, installPath)
	}

	fmt.Println("📦 Installed to:", installPath)

	// Add PATH entry for fallback (~/.local/bin)
	if goos != "windows" {
		ensurePath()
	}

	fmt.Println("🎉 Setup complete! Run: ipm")
}
