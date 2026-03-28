package utils

import (
	"fmt"
	"ipm/types"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	"github.com/fatih/color"
)

var gitCloneRegex = regexp.MustCompile(`\bgit\s+clone\b`)

func IsExternallyManagedError(output string) bool {
	return strings.Contains(output, "externally-managed-environment") ||
		strings.Contains(output, "PEP 668")
}

func IsPermissionError(output string) bool {
	lower := strings.ToLower(output)
	return strings.Contains(lower, "permission denied") ||
		strings.Contains(lower, "operation not permitted") ||
		strings.Contains(lower, "eacces") ||
		strings.Contains(lower, "access to the path") && strings.Contains(lower, "denied") ||
		strings.Contains(lower, "unauthorizedaccessexception")
}

func IsGitCloneDestExistsError(output string) (bool, string) {
	lower := strings.ToLower(output)

	if strings.Contains(lower, "fatal: destination path") &&
		strings.Contains(lower, "already exists") &&
		strings.Contains(lower, "not an empty directory") {

		re := regexp.MustCompile(`destination path '([^']+)'`)
		m := re.FindStringSubmatch(output)
		if len(m) == 2 {
			return true, m[1]
		}
		return true, ""
	}

	return false, ""
}

func RemoveGitCloneCommands(cmds []types.Instruction) []types.Instruction {
	out := make([]types.Instruction, 0, len(cmds))
	for _, c := range cmds {
		if !gitCloneRegex.MatchString(c.Command) {
			out = append(out, c)
		}
	}
	return out
}


// CommandExists checks if a binary is available in the PATH
func CommandExists(cmd string) bool {
    _, err := exec.LookPath(cmd)
    return err == nil
}

// InstallPipx handles platform-specific installation using existing family logic
func InstallPipx() error {
    pipxDep := types.Dependency{
        Install: map[string][]string{
            "darwin":  {"brew install pipx"},
            "windows": {"python -m pip install --user pipx", "python -m pipx ensurepath"},
            "debian":  {"sudo apt-get update", "sudo apt-get install -y pipx"},
            "rpm":     {"sudo dnf install -y pipx"},
            "arch":    {"sudo pacman -Sy --needed --noconfirm python-pipx"},
        },
    }

    cmds := GetInstallCommands(pipxDep)

    if len(cmds) == 0 {
        family := GetLinuxFamily(false)
        return fmt.Errorf("no install commands found for platform: %s (OS: %s)", family, runtime.GOOS)
    }

    // 1. Install pipx using your existing logic
    for _, c := range cmds {
        if err := ExecuteInstallCommand(c); err != nil {
            return fmt.Errorf("failed to install pipx: %w", err)
        }
    }

    // 2. Ensure path (persistence)
    if err := ExecuteInstallCommand("pipx ensurepath"); err != nil {
        // don't hard fail, it's not critical
        fmt.Println("⚠️ failed to run pipx ensurepath:", err)
    }

    // 3. Immediate PATH fix 
    home := os.Getenv("HOME")
    if home == "" {
        return nil
    }

    var pipxPath string
    if runtime.GOOS == "windows" {
        pipxPath = fmt.Sprintf("%s\\AppData\\Roaming\\Python\\Python3\\Scripts", home)
    } else {
        pipxPath = fmt.Sprintf("%s/.local/bin", home)
    }

    currentPath := os.Getenv("PATH")

    // case-insensitive check for safety (esp. Windows)
    if !strings.Contains(strings.ToLower(currentPath), strings.ToLower(pipxPath)) {
        pathSep := ":"
        if runtime.GOOS == "windows" {
            pathSep = ";"
        }

        newPath := pipxPath + pathSep + currentPath
        os.Setenv("PATH", newPath)
    }

    return nil
}


func GetInstallCommands(dep types.Dependency) []string {
	family := GetLinuxFamily(false)
	if runtime.GOOS == "linux" {
		if cmds, ok := dep.Install[family]; ok {
			return cmds
		}
	}
	return dep.Install[runtime.GOOS]
}



func ExecuteInstallCommand(installCmd string) error {
	indent := "    "
	_, err := exec.LookPath("sudo")
	if err != nil && strings.Contains(installCmd, "sudo ") {
		fmt.Println(indent + color.HiBlackString("ℹ 'sudo' not found, stripping from command..."))
		installCmd = strings.ReplaceAll(installCmd, "sudo ", "")
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("powershell", "-NoProfile", "-Command", installCmd)
	} else {
		cmd = exec.Command("bash", "-c", installCmd)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
