package prerequisites

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	utils "ipm/utils"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
)

type Dependency struct {
	Name        string
	Regexes     map[string]*regexp.Regexp
	Install     map[string][]string
	PathAugment map[string][]string
}

// Helper to get the correct regex for the current platform
func (d Dependency) getRegex() *regexp.Regexp {
	// Try to get OS-specific regex
	if r, ok := d.Regexes[runtime.GOOS]; ok {
		return r
	}
	// Fallback to a default/generic one if available
	return d.Regexes["default"]
}

var Dependencies = []Dependency{
	{
		Name: "Git",
		Regexes: map[string]*regexp.Regexp{
			"windows": regexp.MustCompile(`(?i)\bgit\b.*(?:is not recognized|is not installed)`),
			"default": regexp.MustCompile(`(?i)\bgit\b[:\s]+(?:command\s+)?not\s+found|\bgit\b.*is not installed`),
		},
		Install: map[string][]string{
			"darwin":  {"brew install git"},
			"windows": {"winget install --id Git.Git -e --source winget"},
			"debian":  {"sudo apt-get update", "sudo apt-get install -y git"},
			"rpm":     {"sudo dnf install -y git"},
			"arch":    {"sudo pacman -Sy --needed --noconfirm git"},
		},
	},
	{
		Name: "PNPM",
		Regexes: map[string]*regexp.Regexp{
			"windows": regexp.MustCompile(`(?i)\bpnpm\b.*is not recognized`),
			"default": regexp.MustCompile(`(?i)\bpnpm\b[:\s]+(?:command\s+)?not\s+found`),
		},
		Install: map[string][]string{
			"darwin":  {"brew install pnpm"},
			"windows": {"winget install --id pnpm.pnpm -e"},
			"debian":  {"curl -fsSL https://get.pnpm.io/install.sh | sh -"},
			"rpm":     {"curl -fsSL https://get.pnpm.io/install.sh | sh -"},
			"arch":    {"sudo pacman -Sy --needed --noconfirm pnpm"},
		},
	},
	{
		Name: "Node.js/NPM",
		Regexes: map[string]*regexp.Regexp{
			"windows": regexp.MustCompile(`(?i)\bnpm\b.*is not recognized`),
			"default": regexp.MustCompile(`(?i)\bnpm\b[:\s]+(?:command\s+)?not\s+found`),
		},
		Install: map[string][]string{
			"darwin":  {"brew install node"},
			"windows": {"winget install --id OpenJS.NodeJS -e"},
			"debian":  {"sudo apt-get update", "sudo apt-get install -y nodejs npm"},
			"rpm":     {"sudo dnf install -y nodejs"},
			"arch":    {"sudo pacman -Sy --needed --noconfirm nodejs npm"},
		},
	},
	{
		Name: "Docker",
		Regexes: map[string]*regexp.Regexp{
			"windows": regexp.MustCompile(`(?i)\bdocker\b.*is not recognized`),
			"default": regexp.MustCompile(`(?i)\bdocker\b[:\s]+(?:command\s+)?not\s+found`),
		},
		Install: map[string][]string{
			"darwin":  {"brew install --cask docker"},
			"windows": {"winget install --id Docker.DockerDesktop -e"},
			"debian":  {"curl -fsSL https://get.docker.com | sh"},
			"rpm":     {"curl -fsSL https://get.docker.com | sh"},
			"arch":    {"sudo pacman -Sy --needed --noconfirm docker", "sudo systemctl enable --now docker"},
		},
	},
	{
		Name: "Pip",
		Regexes: map[string]*regexp.Regexp{
			"windows": regexp.MustCompile(`(?i)\bpip3?\b.*is not recognized`),
			"default": regexp.MustCompile(`(?i)\bpip3?\b[:\s]+(?:command\s+)?not\s+found`),
		},
		Install: map[string][]string{
			"darwin":  {"python3 -m ensurepip --upgrade"},
			"windows": {"python -m ensurepip --upgrade"},
			"debian":  {"sudo apt-get update", "sudo apt-get install -y python3-pip"},
			"rpm":     {"sudo dnf install -y python3-pip"},
			"arch":    {"sudo pacman -Sy --needed --noconfirm python-pip"},
		},
	},
	{
		Name: "Python",
		Regexes: map[string]*regexp.Regexp{
			"windows": regexp.MustCompile(`(?i)\bpython3?\b.*is not recognized`),
			"default": regexp.MustCompile(`(?i)\bpython3?\b[:\s]+(?:command\s+)?not\s+found`),
		},
		Install: map[string][]string{
			"darwin":  {"brew install python"},
			"windows": {"winget install --id Python.Python.3 -e"},
			"debian":  {"sudo apt-get update", "sudo apt-get install -y python3 python3-pip"},
			"rpm":     {"sudo dnf install -y python3 python3-pip"},
			"arch":    {"sudo pacman -Sy --needed --noconfirm python python-pip"},
		},
	},
	{
		Name: "Conda",
		Regexes: map[string]*regexp.Regexp{
			"windows": regexp.MustCompile(`(?i)\bconda\b.*is not recognized`),
			"default": regexp.MustCompile(`(?i)\bconda\b[:\s]+(?:command\s+)?not\s+found`),
		},
		Install: map[string][]string{
			"darwin":  {"brew install --cask miniconda"},
			"windows": {"winget install --id Anaconda.Miniconda3 -e"},
			"debian":  {"curl -L https://repo.anaconda.com/miniconda/Miniconda3-latest-Linux-x86_64.sh -o miniconda.sh", "bash miniconda.sh -b"},
			"rpm":     {"curl -L https://repo.anaconda.com/miniconda/Miniconda3-latest-Linux-x86_64.sh -o miniconda.sh", "bash miniconda.sh -b"},
			"arch":    {"curl -L https://repo.anaconda.com/miniconda/Miniconda3-latest-Linux-x86_64.sh -o miniconda.sh", "bash miniconda.sh -b"},
		},
	},
	{
		Name: "Dotnet SDK",
		Regexes: map[string]*regexp.Regexp{
			"windows": regexp.MustCompile(`(?i)\bdotnet\b.*is not recognized`),
			"default": regexp.MustCompile(`(?i)\bdotnet\b[:\s]+(?:command\s+)?not\s+found`),
		},
		Install: map[string][]string{
			"darwin":  {"brew install --cask dotnet-sdk"},
			"windows": {"winget install --id Microsoft.DotNet.SDK.8 -e"},
			"debian":  {"curl -sSL https://dot.net/v1/dotnet-install.sh | bash /dev/stdin --install-dir /usr/local/bin"},
			"rpm":     {"sudo dnf install -y dotnet-sdk-8.0"},
			"arch":    {"sudo pacman -Sy --needed --noconfirm dotnet-sdk"},
		},
	},
	{
		Name: "Homebrew",
		Regexes: map[string]*regexp.Regexp{
			"default": regexp.MustCompile(`(?i)\bbrew\b[:\s]+(?:command\s+)?not\s+found`),
		},
		Install: map[string][]string{
			"darwin": {
				"/bin/bash -c \"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\"",
				"echo 'eval \"$(/opt/homebrew/bin/brew shellenv)\"' >> ~/.zprofile && eval \"$(/opt/homebrew/bin/brew shellenv)\"",
			},
			"debian": {
				"/bin/bash -c \"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\"",
				"(echo; echo 'eval \"$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)\"') >> ~/.bashrc && eval \"$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)\"",
			},
			"rpm": {
				"/bin/bash -c \"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\"",
				"(echo; echo 'eval \"$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)\"') >> ~/.bashrc && eval \"$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)\"",
			},
			"arch": {
				"/bin/bash -c \"$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)\"",
				"(echo; echo 'eval \"$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)\"') >> ~/.bashrc && eval \"$(/home/linuxbrew/.linuxbrew/bin/brew shellenv)\"",
			},
		},
		PathAugment: map[string][]string{
			"darwin": {"/opt/homebrew/bin", "/usr/local/bin"},
			"linux":  {"/home/linuxbrew/.linuxbrew/bin"},
		},
	},
	{
		Name: "Foundry (Forge)",
		Regexes: map[string]*regexp.Regexp{
			"windows": regexp.MustCompile(`(?i)\bforge\b.*is not recognized`),
			"default": regexp.MustCompile(`(?i)\bforge\b[:\s]+(?:command\s+)?not\s+found`),
		},
		Install: map[string][]string{
			"darwin": {
				"curl -L https://foundry.paradigm.xyz | bash",
				"source ~/.zshrc && foundryup",
			},
			"windows": {
				"powershell -c \"iwr https://foundry.paradigm.xyz/install.ps1 -useb | iex\"",
				"foundryup",
			},
			"debian": {
				"curl -L https://foundry.paradigm.xyz | bash",
				"foundryup",
			},
			"rpm": {
				"curl -L https://foundry.paradigm.xyz | bash",
				"foundryup",
			},
			"arch": {
				"curl -L https://foundry.paradigm.xyz | bash",
				"foundryup",
			},
		},
	},
	{
		Name: "Make",
		Regexes: map[string]*regexp.Regexp{
			"windows": regexp.MustCompile(`(?i)\bmake\b.*is not recognized`),
			"default": regexp.MustCompile(`(?i)\bmake\b[:\s]+(?:command\s+)?not\s+found`),
		},
		Install: map[string][]string{
			"darwin":  {"brew install make"},
			"windows": {"winget install -e --id Ezwinports.Make"},
			"debian":  {"sudo apt-get update", "sudo apt-get install -y build-essential"},
			"rpm":     {"sudo dnf groupinstall -y \"Development Tools\""},
			"arch":    {"sudo pacman -Sy --needed --noconfirm make"},
		},
	},
	{
		Name: "Yay (AUR Helper)",
		Regexes: map[string]*regexp.Regexp{
			"default": regexp.MustCompile(`(?i)\byay\b[:\s]+(?:command\s+)?not\s+found`),
		},
		Install: map[string][]string{
			"arch": {
				"sudo pacman -Sy --needed --noconfirm base-devel git",
				"git clone https://aur.archlinux.org/yay.git /tmp/yay-install",
				"cd /tmp/yay-install && makepkg -si --noconfirm",
				"rm -rf /tmp/yay-install",
			},
		},
	},
	{
		Name: "Scoop",
		Regexes: map[string]*regexp.Regexp{
			"windows": regexp.MustCompile(`(?i)\bscoop\b.*is not recognized`),
		},
		Install: map[string][]string{
			"windows": {
				"Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser -Force",
				"iwr -useb get.scoop.sh | iex",
			},
		},
		PathAugment: map[string][]string{
			"windows": {fmt.Sprintf("%s\\scoop\\shims", os.Getenv("USERPROFILE"))},
		},
	},
	{
		Name: "Chocolatey",
		Regexes: map[string]*regexp.Regexp{
			"windows": regexp.MustCompile(`(?i)\bchoco\b.*is not recognized`),
		},
		Install: map[string][]string{
			"windows": {
				"Set-ExecutionPolicy Bypass -Scope Process -Force; [System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072; iex ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))",
			},
		},
		PathAugment: map[string][]string{
			"windows": {fmt.Sprintf("%s\\Chocolatey\\bin", os.Getenv("ProgramData"))},
		},
	},
	{
		Name: "Go (Golang)",
		Regexes: map[string]*regexp.Regexp{
			"windows": regexp.MustCompile(`(?i)\bgo\b.*is not recognized`),
			"default": regexp.MustCompile(`(?i)\bgo\b[:\s]+(?:command\s+)?not\s+found`),
		},
		Install: map[string][]string{
			"darwin":  {"brew install go"},
			"windows": {"winget install --id Google.Go -e"},
			"debian":  {"sudo apt-get update", "sudo apt-get install -y golang-go"},
			"rpm":     {"sudo dnf install -y golang"},
			"arch":    {"sudo pacman -Sy --needed --noconfirm go"},
		},
		PathAugment: map[string][]string{
			"windows": {fmt.Sprintf("%s\\go\\bin", os.Getenv("USERPROFILE"))},
			"linux":   {"/usr/local/go/bin", fmt.Sprintf("%s/go/bin", os.Getenv("HOME"))},
			"darwin":  {"/usr/local/go/bin", fmt.Sprintf("%s/go/bin", os.Getenv("HOME"))},
		},
	},
	{
		Name: "Snap",
		Regexes: map[string]*regexp.Regexp{
			"default": regexp.MustCompile(`(?i)\bsnap\b[:\s]+(?:command\s+)?not\s+found`),
		},
		Install: map[string][]string{
			"debian": {"sudo apt-get update", "sudo apt-get install -y snapd", "sudo systemctl enable --now snapd.socket"},
			"rpm":    {"sudo dnf install -y snapd", "sudo ln -s /var/lib/snapd/snap /snap", "sudo systemctl enable --now snapd.socket"},
			"arch":   {"git clone https://aur.archlinux.org/snapd.git /tmp/snapd", "cd /tmp/snapd && makepkg -si --noconfirm", "sudo systemctl enable --now snapd.socket", "sudo ln -s /var/lib/snapd/snap /snap"},
		},
		PathAugment: map[string][]string{
			"linux": {"/snap/bin", "/var/lib/snapd/snap/bin"},
		},
	},
	{
		Name: "Cargo (Rust)",
		Regexes: map[string]*regexp.Regexp{
			"windows": regexp.MustCompile(`(?i)\bcargo\b.*is not recognized`),
			"default": regexp.MustCompile(`(?i)\bcargo\b[:\s]+(?:command\s+)?not\s+found`),
		},
		Install: map[string][]string{
			"darwin": {
				"curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y",
			},
			"windows": {
				"winget install --id Rustlang.Rustup -e",
			},
			"debian": {
				"curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y",
			},
			"rpm": {
				"curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y",
			},
			"arch": {
				"sudo pacman -Sy --needed --noconfirm rustup",
				"rustup default stable",
			},
		},
		PathAugment: map[string][]string{
			"windows": {fmt.Sprintf("%s\\Color\\.cargo\\bin", os.Getenv("USERPROFILE"))},
			"darwin":  {fmt.Sprintf("%s/.cargo/bin", os.Getenv("HOME"))},
			"linux":   {fmt.Sprintf("%s/.cargo/bin", os.Getenv("HOME"))},
		},
	},
}

func HandleMissingDependencies(output string) (bool, string, error) {
	indent := "    "
	for _, dep := range Dependencies {
		reg := dep.getRegex()
		if reg == nil || !reg.MatchString(output) {
			continue
		}

		cmds := getInstallCommands(dep)
		if len(cmds) == 0 {
			return false, "", nil
		}

		fmt.Println(indent + color.New(color.FgRed).Sprintf("❌ %s is not installed.", dep.Name))

		var confirmInstall bool
		prompt := &survey.Confirm{
			Message: fmt.Sprintf("Would you like to install %s?\n    Commands: %s", dep.Name, strings.Join(cmds, " && ")),
			Default: true,
		}

		if err := survey.AskOne(prompt, &confirmInstall); err != nil || !confirmInstall {
			return false, "", err
		}

		fmt.Println(indent + color.New(color.FgYellow).Sprintf("🛠 Installing %s...", dep.Name))

		for _, cmdStr := range cmds {
			if err := executeInstallCommand(cmdStr); err != nil {
				return false, "", fmt.Errorf("failed to install %s at command '%s': %w", dep.Name, cmdStr, err)
			}
		}

		// Logic to update and return the current PATH
		currentPath := os.Getenv("PATH")
		pathSep := ":"
		if runtime.GOOS == "windows" {
			pathSep = ";"
		}

		if dep.Name == "Homebrew" {
			brewPaths := []string{
				"/home/linuxbrew/.linuxbrew/bin",
				"/opt/homebrew/bin",
				"/usr/local/bin",
			}

			for _, p := range brewPaths {
				if _, err := os.Stat(p); err == nil {
					if !strings.Contains(currentPath, p) {
						currentPath = p + pathSep + currentPath
					}
				}
			}
			os.Setenv("PATH", currentPath)
		}

		//  Generic Path Augmentation
		if augPaths, ok := dep.PathAugment[runtime.GOOS]; ok {
			for _, p := range augPaths {
				if !strings.Contains(strings.ToLower(currentPath), strings.ToLower(p)) {
					currentPath = p + pathSep + currentPath
				}
			}
		}

		// Update the current process environment immediately
		os.Setenv("PATH", currentPath)

		// Return true, the new path, and nil error
		return true, currentPath, nil
	}
	return false, os.Getenv("PATH"), nil
}

func getInstallCommands(dep Dependency) []string {
	family := utils.GetLinuxFamily(false)
	if runtime.GOOS == "linux" {
		if cmds, ok := dep.Install[family]; ok {
			return cmds
		}
	}
	return dep.Install[runtime.GOOS]
}

func executeInstallCommand(installCmd string) error {
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
