package install_flow

import (
	"fmt"
	"ipm/types"
	"ipm/version"
	"log"
	"net/url"
	"os/exec"
	"runtime"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	utils "ipm/utils"
)

func handleInstallationExhausted(repo *types.RepoDocumentFull, attempted []types.InstallMethod) {
	bold := color.New(color.Bold).SprintFunc()
	green := color.New(color.FgGreen, color.Bold).SprintFunc()
	fmt.Println("Help us fix " + bold(repo.Name) + " for you and the community.")

	var raiseIssue bool
	prompt := &survey.Confirm{
		// Highlight that it is pre-filled and only requires one final click.
		Message: "Open GitHub to submit a pre-filled report? (Just click " + green("'Create'") + " in the browser)",
		Default: true,
	}

	if err := survey.AskOne(prompt, &raiseIssue); err != nil || !raiseIssue {
		fmt.Println("Understood. You can always report this later if you change your mind.")
		return
	}

	// This ensures they know EXACTLY what to do the moment the browser pops up.
	fmt.Println("\n" + green("🚀 Opening GitHub..."))
	fmt.Println("Just scroll down and press the green " + bold("Create") + " button.")

	openGitHubIssue(repo, attempted)
}

func buildIssueBody(repo *types.RepoDocumentFull, attempted []types.InstallMethod) string {
	var b strings.Builder
	detailedName := utils.GetLinuxFamily(true)

	// Determine if this is a "failed attempt" or a "no compatible methods" report
	isIncompatibilityReport := len(attempted) == 0
	statusHeader := "All attempted installation methods failed."
	if isIncompatibilityReport {
		statusHeader = "None of the provided installation methods are compatible with my system."
	}

	for _, m := range attempted {
		title := m.Title
		if title == "" {
			title = "(untitled method)"
		}

		b.WriteString("#### " + title + "\n")
		for _, instr := range m.Instructions {
			b.WriteString("- `" + instr.Command + "`\n")
		}
		b.WriteString("\n")
	}

	// Use detailedName if available, otherwise fallback to OS/ARCH
	systemInfo := detailedName
	if systemInfo == "" {
		systemInfo = fmt.Sprintf("%s / %s", OS, ARCH)
	}

	return fmt.Sprintf(`
### Repo
%s

### ipm version
%s

### OS / System
%s

### What happened
%s

### Installation methods tried
%s

### Notes
(Anything else you want to add)
`,
		repo.Name,
		version.GetVersion(),
		systemInfo,
		statusHeader,
		b.String(),
	)
}

func openGitHubIssue(repo *types.RepoDocumentFull, attempted []types.InstallMethod) {
	detailedName := utils.GetLinuxFamily(true)

	// Create a more descriptive title
	title := fmt.Sprintf("Installation failed for %s", repo.Name)
	if len(attempted) == 0 {
		title = fmt.Sprintf("Incompatible methods for %s on %s", repo.Name, detailedName)
	}

	params := url.Values{}
	params.Add("title", title)
	params.Add("labels", "Installerpedia,bug,enhancement")
	params.Add("body", buildIssueBody(repo, attempted))

	// ... [Rest of the browser opening logic remains exactly the same] ...
	baseURL := "https://github.com/HexmosTech/Installerpedia/issues/new"
	issueURL := baseURL + "?" + params.Encode()

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		if isWSL() {
			cmd = exec.Command("powershell.exe", "-NoProfile", "-Command", fmt.Sprintf("Start-Process '%s'", issueURL))
		} else {
			cmd = exec.Command("xdg-open", issueURL)
		}
	case "darwin":
		cmd = exec.Command("open", issueURL)
	case "windows":
		cmd = exec.Command("powershell", "-NoProfile", "-Command", fmt.Sprintf("Start-Process '%s'", issueURL))
	default:
		fmt.Printf("Open this URL to report the issue:\n%s\n", issueURL)
		return
	}

	if output, err := cmd.CombinedOutput(); err != nil {
		log.Printf("[ERROR]: %s", string(output))
		fmt.Printf("Failed to open browser automatically. Please visit:\n%s\n", issueURL)
	} else {
		fmt.Println("Issue page opened in your browser.")
	}
}

// Helper function to detect WSL environment
func isWSL() bool {
	releaseData, err := exec.Command("uname", "-r").Output()
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(releaseData)), "microsoft")
}
