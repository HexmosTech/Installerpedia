package prerequisites

import (
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strings"

	"ipm/types"
	utils "ipm/utils"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
)

var Dependencies = []types.Dependency{
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
			"default": regexp.MustCompile(`(?i)(\bpip3?\b[:\s]+(?:command\s+)?not\s+found|no module named pip)`),
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
	{
        Name: "Docker Compose",
        Regexes: map[string]*regexp.Regexp{
            "windows": regexp.MustCompile(`(?i)\bdocker-compose\b.*is not recognized`),
            "default": regexp.MustCompile(`(?i)\bdocker-compose\b[:\s]+(?:command\s+)?not\s+found`),
        },
        Install: map[string][]string{
            "darwin":  {"brew install docker-compose"},
            "windows": {"winget install --id Docker.DockerDesktop -e"},
            "debian":  {"sudo apt-get update", "sudo apt-get install -y docker-compose-plugin"},
            "rpm":     {"sudo dnf install -y docker-compose-plugin"},
            "arch":    {"sudo pacman -Sy --needed --noconfirm docker-compose"},
        },
    },
	{
        Name: "UV (Python Manager)",
        Regexes: map[string]*regexp.Regexp{
            "windows": regexp.MustCompile(`(?i)\buv\b.*is not recognized`),
            "default": regexp.MustCompile(`(?i)\buv\b[:\s]+(?:command\s+)?not\s+found`),
        },
        Install: map[string][]string{
            "darwin":  {"brew install uv"},
            "windows": {"powershell -c \"irm https://astral.sh/uv/install.ps1 | iex\""},
            "debian":  {"curl -LsSf https://astral.sh/uv/install.sh | sh"},
            "rpm":     {"curl -LsSf https://astral.sh/uv/install.sh | sh"},
            "arch":    {"curl -LsSf https://astral.sh/uv/install.sh | sh"},
        },
        PathAugment: map[string][]string{
            "windows": {fmt.Sprintf("%s\\AppData\\Roaming\\uv\\bin", os.Getenv("USERPROFILE"))},
            "darwin":  {fmt.Sprintf("%s/.local/bin", os.Getenv("HOME"))},
            "linux":   {fmt.Sprintf("%s/.local/bin", os.Getenv("HOME"))},
        },
    },
	{
        Name: "Helm",
        Regexes: map[string]*regexp.Regexp{
            "windows": regexp.MustCompile(`(?i)\bhelm\b.*is not recognized`),
            "default": regexp.MustCompile(`(?i)\bhelm\b[:\s]+(?:command\s+)?not\s+found`),
        },
        Install: map[string][]string{
            "darwin":  {"brew install helm"},
            "windows": {"winget install --id Helm.Helm -e"},
            "debian":  {"curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash"},
            "rpm":     {"curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash"},
            "arch":    {"sudo pacman -Sy --needed --noconfirm helm"},
        },
    },
	{
        Name: "Composer",
        Regexes: map[string]*regexp.Regexp{
            "windows": regexp.MustCompile(`(?i)\bcomposer\b.*is not recognized`),
            "default": regexp.MustCompile(`(?i)\bcomposer\b[:\s]+(?:command\s+)?not\s+found`),
        },
        Install: map[string][]string{
            "darwin": {
                "php -r \"copy('https://getcomposer.org/installer', 'composer-setup.php');\"",
                "php composer-setup.php --install-dir=/usr/local/bin --filename=composer",
                "php -r \"unlink('composer-setup.php');\"",
            },
            "windows": {"winget install --id PHP.Composer -e"},
            "debian": {
                "php -r \"copy('https://getcomposer.org/installer', 'composer-setup.php');\"",
                "sudo php composer-setup.php --install-dir=/usr/local/bin --filename=composer",
                "php -r \"unlink('composer-setup.php');\"",
            },
            "rpm": {
                "php -r \"copy('https://getcomposer.org/installer', 'composer-setup.php');\"",
                "sudo php composer-setup.php --install-dir=/usr/local/bin --filename=composer",
                "php -r \"unlink('composer-setup.php');\"",
            },
            "arch": {"sudo pacman -Sy --needed --noconfirm composer"},
        },
    },
	{
        Name: "Unzip",
        Regexes: map[string]*regexp.Regexp{
            "windows": regexp.MustCompile(`(?i)\bunzip\b.*is not recognized`),
            "default": regexp.MustCompile(`(?i)\bunzip\b[:\s]+(?:command\s+)?not\s+found`),
        },
        Install: map[string][]string{
            "darwin":  {"brew install unzip"},
            "windows": {"winget install -e --id GnuWin32.Unzip"},
            "debian":  {"sudo apt-get update", "sudo apt-get install -y unzip"},
            "rpm":     {"sudo dnf install -y unzip"},
            "arch":    {"sudo pacman -Sy --needed --noconfirm unzip"},
        },
    },
	{
        Name: "PHP",
        Regexes: map[string]*regexp.Regexp{
            "windows": regexp.MustCompile(`(?i)\bphp\b.*is not recognized`),
            "default": regexp.MustCompile(`(?i)\bphp\b[:\s]+(?:command\s+)?not\s+found`),
        },
        Install: map[string][]string{
            "darwin":  {"brew install php"},
            "windows": {"winget install --id PHP.PHP -e"},
            "debian":  {"sudo apt-get update", "sudo apt-get install -y php-cli php-common php-curl"},
            "rpm":     {"sudo dnf install -y php-cli php-common php-curl"},
            "arch":    {"sudo pacman -Sy --needed --noconfirm php"},
        },
    },
	{
		Name: "Bun",
		Regexes: map[string]*regexp.Regexp{
			"windows": regexp.MustCompile(`(?i)\bbun\b.*is not recognized`),
			"default": regexp.MustCompile(`(?i)\bbun\b[:\s]+(?:command\s+)?not\s+found`),
		},
		Install: map[string][]string{
			"darwin": {
				"curl -fsSL https://bun.sh/install | bash",
			},
			"windows": {
				"powershell -c \"irm bun.sh/install.ps1 | iex\"",
			},
			"debian": {
				"curl -fsSL https://bun.sh/install | bash",
			},
			"rpm": {
				"curl -fsSL https://bun.sh/install | bash",
			},
			"arch": {
				"curl -fsSL https://bun.sh/install | bash",
			},
		},
		PathAugment: map[string][]string{
			"darwin": {fmt.Sprintf("%s/.bun/bin", os.Getenv("HOME"))},
			"linux":  {fmt.Sprintf("%s/.bun/bin", os.Getenv("HOME"))},
			"windows": {fmt.Sprintf("%s\\.bun\\bin", os.Getenv("USERPROFILE"))},
		},
	},
}

func HandleMissingDependencies(output string) (bool, string, error) {
	indent := "    "
	for _, dep := range Dependencies {
		reg := dep.GetRegex()
		if reg == nil || !reg.MatchString(output) {
			continue
		}

		cmds := utils.GetInstallCommands(dep)
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
			if err := utils.ExecuteInstallCommand(cmdStr); err != nil {
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



