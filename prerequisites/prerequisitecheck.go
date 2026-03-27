package prerequisites

import (
	"fmt"
	"ipm/types"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	"github.com/blang/semver/v4"
	"github.com/fatih/color"
)

var binaryAliases = map[string]string{
	"Node.js": "node",
	"npm":     "npm",
	"Go":      "go",
	"make":    "make",
	"Python":  "python3",
	"pip":     "pip3",
}

func CheckCommandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
func GetCmdVersion(cmd string, args ...string) (string, error) {
	out, err := exec.Command(cmd, args...).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
func VersionSatisfies(current string, constraint string) bool {
	// Extract first semver-looking substring (e.g., "1.2.3")
	re := regexp.MustCompile(`\d+\.\d+\.\d+`)
	verStr := re.FindString(current)
	if verStr == "" {
		return false // no valid semver found
	}

	v, err := semver.ParseTolerant(verStr)
	if err != nil {
		return false
	}

	c, err := semver.ParseRange(constraint)
	if err != nil {
		return false
	}

	return c(v)
}

func isApplicable(p types.Prerequisite, methodType string) bool {
	// If explicitly marked global, always applicable
	for _, a := range p.AppliesTo {
		if a == "global" {
			return true
		}
	}

	// Check if method type matches
	for _, a := range p.AppliesTo {
		if a == methodType {
			return true
		}
	}

	// Check OS
	osName := runtime.GOOS
	var normalizedOS string

	switch osName {
	case "linux":
		normalizedOS = "linux"
	case "darwin":
		normalizedOS = "macos"
	case "windows":
		normalizedOS = "windows"
	default:
		normalizedOS = osName
	}

	for _, a := range p.AppliesTo {
		if a == normalizedOS {
			return true
		}
	}

	// If no match → not applicable
	return false
}

func CheckPrerequisites(prereqs []types.Prerequisite, methodType string) error {
	green := color.New(color.FgGreen, color.Bold).SprintFunc()
	yellow := color.New(color.FgYellow, color.Bold).SprintFunc()
	red := color.New(color.FgRed, color.Bold).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	headingPrefix := "⚙"
	prefix := "•"
	indent := "     " // 5 spaces

	// Heading
	fmt.Println(bold("\n" + headingPrefix + " Checking prerequisites...\n"))

	// compute max length of tool names for alignment
	maxLen := 0
	for _, p := range prereqs {
		if !isApplicable(p, methodType) {
			continue
		}
		if len(p.Name) > maxLen {
			maxLen = len(p.Name)
		}
	}

	failed := false // <— TRACK FAILURES

	for _, p := range prereqs {
		if !isApplicable(p, methodType) {
			continue
		}

		cmdName := p.Name
		if alias, ok := binaryAliases[p.Name]; ok {
			cmdName = alias
		}

		// Get full path of binary
		path, err := exec.LookPath(cmdName)
		if err != nil {
			if p.Optional {
				fmt.Printf("%s%s %-*s (checking path: %s) : %s\n", indent, prefix, maxLen, p.Name, cmdName, yellow("Missing (optional)"))
				continue
			}

			fmt.Printf("%s%s %-*s (checking path: %s) : %s\n", indent, prefix, maxLen, p.Name, cmdName, red("❌ Missing"))
			failed = true
			continue

		}

		// Version check if applicable
		if p.Version != "" {
			ver, err := GetCmdVersion(cmdName, "--version")
			if err != nil {
				fmt.Printf("%s%s %-*s (checking path: %s) : %s (failed to read version)\n",
					indent, prefix, maxLen, p.Name, path, yellow("⚠"))
				failed = true
				continue
			}
			if !VersionSatisfies(ver, p.Version) {
				fmt.Printf("%s%s %-*s (checking path: %s) : %s (need %s)\n",
					indent, prefix, maxLen, p.Name, path, red("❌ Version mismatch"), p.Version)
				failed = true
				continue

			}
		}

		fmt.Printf("%s%s %-*s (checking path: %s) : %s\n", indent, prefix, maxLen, p.Name, path, green("✔"))
	}

	// Final summary
	if failed {
		fmt.Println("\n" + indent + yellow("⚠ Some prerequisites need attention. Please review and fix them if needed before installing.\n"))
	} else {
		fmt.Println(green("✔ All prerequisites satisfied.\n"))
	}

	return nil
}
