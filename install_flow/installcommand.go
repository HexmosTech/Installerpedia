package install_flow

import (
	"bytes"
	"net/http"

	"encoding/json"
	"fmt"
	"io"
	"ipm/types"
	"ipm/utils"

	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	endpoints "ipm/endpoints"
	live_generation "ipm/live_generation"
	prerequisites "ipm/prerequisites"
	tracker "ipm/tracker"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"golang.org/x/term"
)

var isInterrupted = false
var liveLogs strings.Builder
var activeMethod string
var activeCommands string
var doneTracking = make(chan bool, 1)

type NavigationAction int

const (
	NavigationActionExit NavigationAction = iota
	NavigationActionBack
	NavigationActionForward
	NavigationActionCancel
)

var (
	OS   = runtime.GOOS
	ARCH = runtime.GOARCH
)

const indent = "    "

// 1. Define the regex patterns for specific package managers
var (
	debianRegex   = regexp.MustCompile(`\b(apt|apt-get|dpkg)\b`)
	rpmRegex      = regexp.MustCompile(`\b(dnf|yum|rpm|zypper)\b`)
	archRegex     = regexp.MustCompile(`\b(pacman|yay|paru|makepkg)\b`)
	unixToolRegex = regexp.MustCompile(`\b(brew|bash|sh|zsh)\b`)

	// windowsRegex identifies Windows-specific installers while avoiding false positives on Linux.
	windowsRegex = regexp.MustCompile(
		`\b(winget|choco|scoop)\b` + "|" + // Standard package managers
			`(\b(irm|iwr|iex)\b[^\n|]*\|[^\n]*\b(iex|irm|iwr)\b)` + "|" + // PowerShell pipes (irm...|iex)
			`(\b(irm|iwr|iex)\b[^\n]*https?://[^\n\s]+)` + "|" + // Downloaders + URL
			`(\b(irm|iwr|iex)\b[^\n]*\.ps1)`, // Downloaders + .ps1 script
	)

	serverStartRegex = regexp.MustCompile(`(?i)\b(npm\s+run\s+(dev|start)|npm\s+start|yarn\s+dev|yarn\s+start|pnpm\s+dev|pnpm\s+start|vite|next\s+dev|node\s+server\.js)\b`)
)

func isServerStartCommand(cmd string) bool {
	return serverStartRegex.MatchString(strings.ToLower(cmd))
}

func InstallCommand(query string) {
	var (
		repo *types.RepoDocumentFull
		nav  NavigationAction
	)

	// PAGE 1: select repo
SelectRepo:
	for {
		repo, nav = SelectRepoMeili(query)
		switch nav {
		case NavigationActionExit, NavigationActionBack:
			return
		case NavigationActionForward:
			if repo != nil {
				break SelectRepo
			}
		}
	}

	// Hand off to retry-aware flow (failed methods are removed automatically)
	runInstallFlow(repo)
}

func InstallCommandWithMatches(matches []*types.RepoDocumentFull) {
	// Let user select from matches instead of searching again
	repoOptions := make([]string, len(matches))
	for i, r := range matches {
		repoOptions[i] = r.Name
	}

	var selected string
	prompt := &survey.Select{
		Message: "Select a repo to install:",
		Options: repoOptions,
	}
	survey.AskOne(prompt, &selected)

	// Find the selected repo object
	var repo *types.RepoDocumentFull
	for _, r := range matches {
		if r.Name == selected {
			repo = r
			break
		}
	}

	if repo != nil {
		runInstallFlow(repo)
	}
}

func checkForRepoUpdates(repoName string) (*types.RepoDocumentFull, bool) {
	fmt.Printf("🔍 %s\n", color.CyanString("Checking for updates for %s...", repoName))

	defer func() { recover() }()

	payload, err := json.Marshal(map[string]string{"repo": repoName})
	if err != nil {
		return nil, false
	}

	client := &http.Client{Timeout: 10 * time.Second} // Increased timeout for Gemini processing
	url := endpoints.Endpoints.CheckRepoUpdates.Get()

	resp, err := client.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		// log.Printf("⚠️ Update request failed: %v", err)
		return nil, false
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var result struct {
			HasUpdate bool                   `json:"has_update"`
			Data      types.RepoDocumentFull `json:"data"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			// log.Printf("⚠️ Failed to decode update response: %v", err)
			return nil, false
		}

		if result.HasUpdate {
			fmt.Printf("%s\n", color.CyanString("✨ %s: New version found and refined.", repoName))

			formattedBytes, err := json.MarshalIndent(result.Data, "", "    ")
			formattedResult := string(formattedBytes)

			if err != nil {
				// Fallback to a basic string representation if marshaling fails
				formattedResult = fmt.Sprintf("%+v", result.Data)
			}

			tracker.TrackInstallationJsonGenerated(repoName, formattedResult, "github_sync")
			return &result.Data, true
		} else {
			fmt.Printf("%s\n", color.CyanString("%s: Already up to date.", repoName))
		}
	}

	return nil, false
}

// Core flow after a repo is selected
func runInstallFlow(repo *types.RepoDocumentFull) {

	// Auto-Update Repo ---------------

	//if updatedRepo, hasUpdate := checkForRepoUpdates(repo.Name); hasUpdate {
	//    fmt.Printf("✨ %s\n", color.GreenString("Updated installation steps detected. Proceeding with the latest version."))
	//    updatedRepo.Name = repo.Name
	//	repo = updatedRepo
	//}

	// ---------------------------------

	attemptedMethods := make([]types.InstallMethod, 0)

	// --- SIGNAL HANDLING START ---
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		utils.DebugLog("[SIGNAL DEBUG] Signal received at %s\n", time.Now().Format("15:04:05.000"))
		<-sigChan
		utils.DebugLog("[SIGNAL DEBUG] Processing interrupt signal at %s\n", time.Now().Format("15:04:05.000"))
		isInterrupted = true
		// Capture the logs accumulated so far
		errorTrace := liveLogs.String()
		if errorTrace == "" {
			errorTrace = "User cancelled before any command output was generated."
		}
		utils.DebugLog("[SIGNAL DEBUG] Error trace captured, length: %d chars\n", len(errorTrace))
		utils.DebugLog("[SIGNAL DEBUG] About to call trackInstallCancelled (this will BLOCK until HTTP completes)...\n")

		// 2. Existing logic for reporting bugs if some methods already failed
		if len(attemptedMethods) > 0 && utils.GlobalConfig.ReportBugs {
			fmt.Println("\n" + color.YellowString("We noticed that your previous installation attempt did not complete successfully."))
			handleInstallationExhausted(repo, attemptedMethods)
		}
		startTime := time.Now()
		// 1. Fire the cancellation event (blocks until HTTP request completes or times out)
		tracker.TrackInstallCancelled(
			repo.Name,
			activeMethod,
			activeCommands,
			errorTrace,
		)
		duration := time.Since(startTime)
		utils.DebugLog("[SIGNAL DEBUG] trackInstallCancelled returned after %v at %s\n", duration, time.Now().Format("15:04:05.000"))

		fmt.Println("\n" + color.YellowString("\n⚠️  Installation cancelled by user (Ctrl+C)."))
		utils.DebugLog("[SIGNAL DEBUG] Sending to doneTracking channel at %s\n", time.Now().Format("15:04:05.000"))
		doneTracking <- true
		utils.DebugLog("[SIGNAL DEBUG] Sent to doneTracking channel at %s\n", time.Now().Format("15:04:05.000"))

	}()
	defer signal.Stop(sigChan)
	// --- SIGNAL HANDLING END ---

	bold := color.New(color.FgCyan, color.Bold).SprintFunc()
	fmt.Println("\n📦 Installing:", bold(repo.Name))
	fmt.Println(strings.Repeat("─", len(repo.Name)+15))

	if err := prerequisites.CheckPrerequisites(repo.Prerequisites, "global"); err != nil {
		fmt.Printf("❌ %v\n", err)
		fmt.Println("Installation aborted.\n")
		return
	}

	remainingMethods := append([]types.InstallMethod(nil), repo.InstallationMethods...)

	// TRACKER: Keep track of methods generated by AI in this session
	var aiGeneratedMethods []types.InstallMethod

	hasAttemptedAutoGeneration := false

	// Helper to check if we have any compatible methods left
	hasCompatible := func(methods []types.InstallMethod) bool {
		sysFamily := utils.GetLinuxFamily(false)
		if sysFamily == "" {
			sysFamily = OS
		}
		for _, m := range methods {
			combinedCmds := ""
			for _, instr := range m.Instructions {
				combinedCmds += " " + strings.ToLower(instr.Command)
			}

			// Determine method's family
			methodFam := ""
			if debianRegex.MatchString(combinedCmds) {
				methodFam = "debian"
			}
			if rpmRegex.MatchString(combinedCmds) {
				methodFam = "rpm"
			}
			if archRegex.MatchString(combinedCmds) {
				methodFam = "arch"
			}
			if windowsRegex.MatchString(combinedCmds) {
				methodFam = "windows"
			}

			// Handle the Unix-bridge (Brew)
			if unixToolRegex.MatchString(combinedCmds) {
				if runtime.GOOS != "windows" {
					methodFam = sysFamily // Force a match for Mac/Linux
				} else {
					methodFam = "unix-tool"
				}
			}

			// 1. Block Windows tools on Mac/Linux
			if runtime.GOOS != "windows" && methodFam == "windows" {
				continue
			}
			// 2. Block Unix tools on Windows
			if runtime.GOOS == "windows" && methodFam == "unix-tool" {
				continue
			}

			// 3. Final compatibility check
			if methodFam == "" || methodFam == sysFamily {
				return true
			}
		}
		return false
	}
	for {
		isBetterMethod := false // Reset for this iteration
		if len(remainingMethods) == 0 || !hasCompatible(remainingMethods) {
			// Only auto-trigger if we haven't tried generating for this specific "all incompatible" state yet
			if !hasAttemptedAutoGeneration {
				fmt.Printf("\n🔍 %s\n", color.CyanString("No compatible methods found for your system. Searching for alternatives..."))
				newMethods, err := live_generation.ProcessGeneratedRepoMethod(repo.Name, "github", utils.GetSupplementedOS())
				hasAttemptedAutoGeneration = true // Mark that we've tried

				if err == nil && len(newMethods) > 0 {
					aiGeneratedMethods = append(aiGeneratedMethods, newMethods...)
					fmt.Printf("✨ %s\n", color.GreenString("Found a potentially better method: %s", newMethods[0].Title))
					remainingMethods = append(newMethods, remainingMethods...)
					// Loop continues, now hasCompatible will be true, showing the new method to the user
					continue
				}
				// If AI fails, the loop continues below to show the selection list (which will now show the Report option)
			}

			// If we are here, it means we either just failed AI generation
			// OR the AI-generated method was tried and it failed too.
			if len(remainingMethods) == 0 {
				fmt.Println("\n" + color.YellowString("⚠️  All installation methods failed and no new methods found."))
				handleInstallationExhausted(repo, attemptedMethods)
				return
			}
		}

		method, nav := selectInstallMethodFromList(remainingMethods, len(attemptedMethods))
		if nav == NavigationActionExit {
			if len(attemptedMethods) > 0 && utils.GlobalConfig.ReportBugs {
				fmt.Println("\n" + color.YellowString("\nWe noticed that your previous installation attempt did not complete successfully."))
				handleInstallationExhausted(repo, attemptedMethods)
			} else {
				fmt.Println("Installation cancelled. You can run it again anytime.")
			}
			return
		}
		// Check if the selected method is one we just got from AI
		for _, aiM := range aiGeneratedMethods {
			if aiM.Title == method.Title {
				isBetterMethod = true
				break
			}
		}
		activeMethod = method.Title // Tracking the last active method
		activeCommands = strings.Join(utils.GetCommands(method.Instructions), " && ")

		if method.Title == "REPORT_FAILURE" {
			handleInstallationExhausted(repo, attemptedMethods)
			continue // Return to menu after reporting
		}

		if nav == NavigationActionBack {
			fmt.Println("Returning to repository selection...")
			return
		}

		// Execute the method and get the navigation result
		navResult, err := runInstallationWithRetry(repo, method)

		if navResult == NavigationActionBack {
			continue
		}

		if navResult == NavigationActionCancel {
			// If they had previous errors, show the report prompt
			if len(attemptedMethods) > 0 && utils.GlobalConfig.ReportBugs {
				fmt.Println("\n" + color.YellowString("We noticed that your previous installation attempt did not complete successfully."))
				handleInstallationExhausted(repo, attemptedMethods)
			}
			return
		}

		if err == nil {
			if isBetterMethod {
				callUpdateMethodsAPI(repo.Name, []types.InstallMethod{method})
			}
			displayPostInstallation(repo)
			fmt.Println("\n" + bold("✅ Installation complete."))
			return
		}

		// REAL FAILURE: Remove from the list so the loop moves toward exhausting remainingMethods
		attemptedMethods = append(attemptedMethods, method)
		for i, m := range remainingMethods {
			if m.Title == method.Title {
				remainingMethods = append(remainingMethods[:i], remainingMethods[i+1:]...)
				break
			}
		}

		// If we just exhausted the list, the next iteration of the loop
		// will trigger the "Searching for better methods" block above.
		if len(remainingMethods) > 0 {
			fmt.Println("\n⚠️ Installation failed. Please try choosing another method.\n")
		}
	}
}

func runInstallationWithRetry(repo *types.RepoDocumentFull, method types.InstallMethod) (NavigationAction, error) {
	nav := confirmAndRunInstallation(repo, method)

	if nav == NavigationActionForward {
		return NavigationActionForward, nil // Success
	}

	if nav == NavigationActionBack {
		return NavigationActionBack, nil // User wants to go back to method list
	}
	if nav == NavigationActionCancel {
		return NavigationActionCancel, nil // Pass the explicit cancel up with NO error
	}

	return NavigationActionExit, fmt.Errorf("installation failed")
}

func selectInstallMethodFromList(methods []types.InstallMethod, attemptedCount int) (types.InstallMethod, NavigationAction) {
	type scoredMethod struct {
		method  types.InstallMethod
		label   string
		isMatch bool
	}

	indent := "    "
	sysFamily := utils.GetLinuxFamily(false)
	if sysFamily == "" {
		sysFamily = OS
	}

	detailedName := utils.GetLinuxFamily(true)

	promptMsg := "Choose an installation method:"
	if detailedName != "" {
		promptMsg += color.HiBlackString(" [%s]", detailedName)
	} else {
		promptMsg += color.HiBlackString(" [%s]", OS)
	}

	var items []scoredMethod
	allIncompatible := true

	for i, m := range methods {
		title := m.Title
		if title == "" {
			title = fmt.Sprintf("Method %d", i+1)
		}

		familyTag := ""
		combinedCmds := ""
		for _, instr := range m.Instructions {
			combinedCmds += " " + strings.ToLower(instr.Command)
		}

		currentMethodFamily := ""
		if debianRegex.MatchString(combinedCmds) {
			familyTag = " (for Debian)"
			currentMethodFamily = "debian"
		} else if rpmRegex.MatchString(combinedCmds) {
			familyTag = " (for RPM)"
			currentMethodFamily = "rpm"
		} else if archRegex.MatchString(combinedCmds) {
			familyTag = " (for Arch)"
			currentMethodFamily = "arch"
		} else if windowsRegex.MatchString(combinedCmds) {
			familyTag = " (for Windows)"
			currentMethodFamily = "windows"
		} else if unixToolRegex.MatchString(combinedCmds) {
			familyTag = " (for macOS/Linux)"
			// If we are on Mac or Linux, we treat this as a native match
			// by aligning it with the current OS/sysFamily.
			if runtime.GOOS != "windows" {
				currentMethodFamily = sysFamily
			} else {
				currentMethodFamily = "unix-tool"
			}
		}

		isMatch := sysFamily == "" || currentMethodFamily == "" || currentMethodFamily == sysFamily
		if runtime.GOOS != "windows" && currentMethodFamily == "windows" {
			isMatch = false
		}

		if runtime.GOOS == "windows" && currentMethodFamily == "unix-tool" {
			isMatch = false
		}

		if isMatch {
			allIncompatible = false
		}

		displayLabel := fmt.Sprintf("%s%s%s — %d commands", indent, title, familyTag, len(m.Instructions))
		if !isMatch {
			displayLabel = color.RedString(displayLabel + " [Incompatible]")
		}

		items = append(items, scoredMethod{m, displayLabel, isMatch})
	}

	// Friendly labels for the novice user
	incompatibleLabel := indent + color.HiWhiteString("[!] None of the methods are compatible for my system (Report issue)")
	failureLabel := indent + color.RedString("[!] Installation failed (Report bug to GitHub)")

	var sortedItems []scoredMethod
	var options []string

	// --- OPTION ARRANGEMENT ---

	// 1. If everything is greyed out, put the HELP option at the VERY TOP
	if allIncompatible {
		options = append(options, incompatibleLabel)
	}

	// 2. Add compatible methods
	for _, item := range items {
		if item.isMatch {
			options = append(options, item.label)
			sortedItems = append(sortedItems, item)
		}
	}
	// 3. Add incompatible methods
	for _, item := range items {
		if !item.isMatch {
			options = append(options, item.label)
			sortedItems = append(sortedItems, item)
		}
	}

	// 4. If some things were compatible but failed, put the failure report at the bottom
	if !allIncompatible && attemptedCount > 0 {
		options = append(options, failureLabel)
	}

	options = append(options, indent+"[Cancel]")

	var selected string
	prompt := &survey.Select{
		Message:  promptMsg,
		Options:  options,
		PageSize: 10,
	}

	if err := survey.AskOne(prompt, &selected); err != nil {
		if err.Error() == "interrupt" {
			utils.DebugLog("[EXIT DEBUG] selectInstallMethodFromList: interrupt detected, waiting for doneTracking at %s\n", time.Now().Format("15:04:05.000"))
			<-doneTracking // <--- WAIT FOR POSTHOG TO FINISH
			utils.DebugLog("[EXIT DEBUG] selectInstallMethodFromList: received doneTracking, exiting at %s\n", time.Now().Format("15:04:05.000"))
			os.Exit(0)
		}
		return types.InstallMethod{}, NavigationActionExit
	}

	if selected == indent+"[Cancel]" {
		return types.InstallMethod{}, NavigationActionExit
	}

	// Handle Report/Help selections
	if selected == incompatibleLabel || selected == failureLabel {
		return types.InstallMethod{Title: "REPORT_FAILURE"}, NavigationActionForward
	}

	// Handle selection validation
	// Since sortedItems index doesn't strictly match options index anymore (due to top-level help),
	// we match based on the text of the selection.
	for _, item := range sortedItems {
		if item.label == selected {
			if !item.isMatch {
				fmt.Printf("\n%s %s\n",
					color.YellowString("!"),
					color.New(color.Bold).Sprint("This method is for a different system. Please choose a compatible one or request support."))
				return selectInstallMethodFromList(methods, attemptedCount)
			}
			return item.method, NavigationActionForward
		}
	}

	return types.InstallMethod{}, NavigationActionExit
}

func displayPostInstallation(repo *types.RepoDocumentFull) {
	if len(repo.PostInstallation) == 0 {
		return
	}

	symbol := "⚡"                                  // Post-installation symbol
	white := color.New(color.FgWhite).SprintFunc() // Steps in white

	// Heading
	fmt.Println("\n" + symbol + " Post Installation Steps:\n")

	for _, step := range repo.PostInstallation {
		// Indented step
		fmt.Println(indent+"•", white(step))
	}
	fmt.Println()
}

func confirmAndRunInstallation(repo *types.RepoDocumentFull, method types.InstallMethod) NavigationAction {

	greenBold := color.New(color.FgGreen, color.Bold).SprintFunc()
	redBold := color.New(color.FgRed, color.Bold).SprintFunc()
	// cyan := color.New(color.FgCyan).SprintFunc()

	// Get terminal width
	termWidth, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		termWidth = 80
	}

	maxWidth := int(float64(termWidth) * 0.65) // 65% of screen width

	// Now calculate column widths inside that table size
	cmdWidth := int(float64(maxWidth) * 0.40)
	// meaningWidth := maxWidth - cmdWidth - 10 // padding + borders

	// Build table
	var buf strings.Builder
	t := table.NewWriter()
	t.SetStyle(table.StyleRounded)
	t.Style().Options.SeparateRows = true
	t.Style().Color.Header = text.Colors{text.Bold}
	t.SetOutputMirror(&buf)

	// FORCE total table width = maxWidth
	t.SetAllowedRowLength(maxWidth)

	// Column wrapping
	t.SetColumnConfigs([]table.ColumnConfig{
		{
			Name:     "Command",
			WidthMax: cmdWidth,
			Transformer: func(val interface{}) string {
				return text.WrapSoft(fmt.Sprint(val), cmdWidth)
			},
		},
		// {
		// 	Name:     "Meaning",
		// 	WidthMax: meaningWidth,
		// 	Transformer: func(val interface{}) string {
		// 		return text.WrapSoft(fmt.Sprint(val), meaningWidth)
		// 	},
		// },
	})

	t.AppendHeader(table.Row{"Command"}) // "Meaning"

	for _, instr := range method.Instructions {
		cmdText := instr.Command
		if instr.Optional {
			cmdText = color.CyanString("[OPTIONAL] ") + cmdText
		}
		t.AppendRow(table.Row{cmdText})
	}
	bold := color.New(color.Bold).SprintFunc()
	fmt.Println(bold("\nℹ The following commands will be executed:\n"))

	t.Render()

	// Prepend indentation to each line
	for _, line := range strings.Split(buf.String(), "\n") {
		fmt.Println(indent + line)
	}
	fmt.Println() // blank line

	// Confirmation options
	options := []string{
		indent + "[Yes]",
		indent + "[Back]",
		indent + "[Cancel installation]",
	}

	var selected string
	prompt := &survey.Select{
		Message:  "Confirm installation:",
		Options:  options,
		PageSize: 3,
	}

	if err := survey.AskOne(prompt, &selected); err != nil {
		if err.Error() == "interrupt" {
			utils.DebugLog("[EXIT DEBUG] confirmAndRunInstallation: interrupt detected, waiting for doneTracking at %s\n", time.Now().Format("15:04:05.000"))
			// Block forever to let the signal goroutine handle the exit
			<-doneTracking // Wait for the tracking goroutine to signal it's done
			utils.DebugLog("[EXIT DEBUG] confirmAndRunInstallation: received doneTracking, exiting at %s\n", time.Now().Format("15:04:05.000"))
			os.Exit(0)
		}
		return NavigationActionExit
	}
	switch selected {
	case indent + "[Back]":
		return NavigationActionBack

	case indent + "[Cancel installation]":
		fmt.Println(redBold("Installation cancelled. You can run it again anytime."))
		return NavigationActionCancel

	case indent + "[Yes]":
		// continue below
	}

	fmt.Println(bold("\n🚀 Running installation steps..."))
	currentIndex := 0
	for currentIndex < len(method.Instructions) {
		// Group consecutive mandatory instructions
		var mandatoryBatch []types.Instruction
		for currentIndex < len(method.Instructions) && !method.Instructions[currentIndex].Optional {
			mandatoryBatch = append(mandatoryBatch, method.Instructions[currentIndex])
			currentIndex++
		}

		if len(mandatoryBatch) > 0 {
			if err := runScriptWithStatus(repo, method, mandatoryBatch, true); err != nil {
				if isInterrupted {
					<-doneTracking
					os.Exit(0)
				}
				fmt.Println(indent + redBold("Installation aborted due to error."))
				return NavigationActionExit
			}
		}

		// Handle optional instruction (if any)
		if currentIndex < len(method.Instructions) && method.Instructions[currentIndex].Optional {
			optInstr := method.Instructions[currentIndex]
			currentIndex++

			var runOpt bool
			promptMsg := fmt.Sprintf("Run optional command: %s?", color.CyanString(optInstr.Command))
			if optInstr.Meaning != "" {
				promptMsg = fmt.Sprintf("Run optional command: %s? [%s]", color.CyanString(optInstr.Command), color.HiGreenString(optInstr.Meaning))
			}

			prompt := &survey.Confirm{
				Message: promptMsg,
				Default: true,
			}
			if err := survey.AskOne(prompt, &runOpt); err != nil {
				if err.Error() == "interrupt" {
					<-doneTracking
					os.Exit(0)
				}
				// On other errors, just skip this optional one
				continue
			}

			if runOpt {
				if err := runScriptWithStatus(repo, method, []types.Instruction{optInstr}, true); err != nil {
					if isInterrupted {
						<-doneTracking
						os.Exit(0)
					}
					fmt.Println(indent + redBold("Optional command failed. Continuing..."))
				}
			} else {
				fmt.Println(indent + "Skipping optional command.")
			}
		}
	}

	if isInterrupted {
		<-doneTracking
		os.Exit(0)
	}

	tracker.TrackInstallSuccess(
		repo.Name,
		method.Title,
		method.Instructions,
	)

	fmt.Println(greenBold("\n✔ Installation completed successfully!"))
	return NavigationActionForward

}

func runScriptWithStatus(repo *types.RepoDocumentFull, method types.InstallMethod, commands []types.Instruction, silent bool) error {
	liveLogs.Reset() // Live logs variable to track in-between cancellation logs
	// Detect server start command at the end and skip execution
	var skippedServerCmd string
	pythonFixed := false

	if len(commands) > 0 {
		lastCmd := strings.TrimSpace(commands[len(commands)-1].Command)

		if isServerStartCommand(lastCmd) {
			skippedServerCmd = lastCmd
			commands = commands[:len(commands)-1]
		}
	}
	isArgumentError := func(output string) bool {
		l := strings.ToLower(output)
		return strings.Contains(l, "the following arguments are required") ||
			strings.Contains(l, "required arguments") ||
			strings.Contains(l, "missing required") ||
			(strings.Contains(l, "usage:") && strings.Contains(l, "error:"))
	}

	execDir := ""          //  PERSISTENT
	pipSwapped := false    //  Track if we've already tried the pip/pip3 swap
	brewSwapped := false   //  Track if we've already tried the MacPorts swap
	var updatedPath string //  Define this to persist the new PATH across retries
	buildAndRun := func(useSudo bool) (string, error) {
		var script strings.Builder
		var cmd *exec.Cmd
		var tempFileName string

		if runtime.GOOS == "windows" {
			// Force UTF-8 Output
			script.WriteString("chcp 65001 > $null\r\n")
			for _, instr := range commands {
				script.WriteString(instr.Command + "\r\n")
			}

			// Create a temporary script
			tempScript, err := os.CreateTemp("", "ipm-*.ps1")
			if err != nil {
				return "", err
			}
			tempFileName = tempScript.Name()
			tempScript.WriteString(script.String())
			tempScript.Close()

			if useSudo {
				// Verb RunAs triggers the UAC prompt
				elevateCmd := fmt.Sprintf("Start-Process powershell -ArgumentList '-NoProfile', '-ExecutionPolicy', 'Bypass', '-File', '%s' -Verb RunAs -Wait", tempFileName)
				cmd = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", elevateCmd)
			} else {
				cmd = exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", tempFileName)
			}
			baseEnv := os.Environ()
			if updatedPath != "" {
				baseEnv = append(baseEnv, "PATH="+updatedPath)
			}
			cmd.Env = baseEnv // Apply environment to the powershell process
		} else {
			// Linux/Darwin: Retain existing bash logic
			script.WriteString("set -e\n")
			script.WriteString("set -o pipefail\n")
			for _, instr := range commands {
				script.WriteString(instr.Command + "\n")
			}

			// Create a temporary script
			tempScript, err := os.CreateTemp("", "ipm-*.sh")
			if err != nil {
				return "", err
			}
			tempFileName = tempScript.Name()
			tempScript.WriteString(script.String())
			tempScript.Close()
			os.Chmod(tempFileName, 0755)

			if useSudo {
				cmd = exec.Command("sudo", "bash", tempFileName)
			} else {
				cmd = exec.Command("bash", tempFileName)
			}
			// Force English UTF-8 locale for Linux and macOS
			baseEnv := os.Environ()
			if updatedPath != "" {
				// Prepend updatedPath to existing PATH for this command
				baseEnv = append(baseEnv, "PATH="+updatedPath)
			}
			cmd.Env = append(baseEnv, "LC_ALL=en_US.UTF-8", "LANG=en_US.UTF-8", "LANGUAGE=en_US.UTF-8")
		}

		if tempFileName != "" {
			defer os.Remove(tempFileName)
		}

		if execDir != "" {
			cmd.Dir = execDir
		}

		// Connect Stdin to os.Stdin to allow interactive input
		cmd.Stdin = os.Stdin

		stdout, _ := cmd.StdoutPipe()
		stderr, _ := cmd.StderrPipe()

		if err := cmd.Start(); err != nil {
			return "", err
		}

		var out strings.Builder
		var mu sync.Mutex
		atStartOfLine := true
		stream := func(r io.Reader) {
			buf := make([]byte, 1024)
			for {
				n, err := r.Read(buf)
				if n > 0 {
					data := buf[:n]

					mu.Lock()
					out.Write(data)
					liveLogs.Write(data)

					// Print to console with indentation
					if atStartOfLine {
						fmt.Print(indent)
						atStartOfLine = false
					}

					// Replace \n with \n+indent
					indented := bytes.ReplaceAll(data, []byte("\n"), []byte("\n"+indent))

					// If data ends with \n, the next Read should handle the indent
					if bytes.HasSuffix(indented, []byte(indent)) {
						indented = indented[:len(indented)-len(indent)]
						atStartOfLine = true
					}
					os.Stdout.Write(indented)
					mu.Unlock()
				}
				if err != nil {
					break
				}
			}
		}

		go stream(stdout)
		go stream(stderr)

		err := cmd.Wait()
		return out.String(), err
	}

	for { // 🔁 RETRY LOOP (NO RECURSION)
		output, err := buildAndRun(false)

		if err == nil {
			if !silent {
				fmt.Println(indent + color.New(color.FgGreen).Sprint("✓ All commands completed"))
			}

			if skippedServerCmd != "" {
				fmt.Println(color.New(color.FgYellow).Sprint("ℹ Server start command skipped."))
				fmt.Println(indent + "Run this to start the server:")
				fmt.Println(indent + color.New(color.Bold).Sprint(skippedServerCmd))
			}

			return nil
		}

		// --- MACPORTS / BREW FALLBACK HANDLER (FOR OLDER MACOS) ---
		if runtime.GOOS == "darwin" && !brewSwapped {
			hasBrew := false
			for _, instr := range commands {
				if strings.Contains(instr.Command, "brew install") {
					hasBrew = true
					break
				}
			}

			if hasBrew {
				var useMacPorts bool
				prompt := &survey.Confirm{
					Message: "Homebrew command failed. Would you like to try again using MacPorts (sudo port)?",
					Default: true,
				}

				if survey.AskOne(prompt, &useMacPorts) == nil && useMacPorts {
					brewSwapped = true
					for i, instr := range commands {
						// Replace brew install with sudo port install
						commands[i].Command = strings.ReplaceAll(instr.Command, "brew install", "sudo port install")
					}
					fmt.Println(indent + color.New(color.FgYellow).Sprint("🔄 Retrying with MacPorts..."))
					continue // 🔁 Retry loop with MacPorts commands
				}
			}
		}

		// GIT CLONE DESTINATION EXISTS HANDLER
		if ok, folder := utils.IsGitCloneDestExistsError(output); ok {

			var choice string
			options := []string{
				"Create isolated folder and retry clone",
				"Skip clone and use existing folder",
			}

			prompt := &survey.Select{
				Message: "Target folder already exists. How should ipm proceed?",
				Options: options,
			}

			if err := survey.AskOne(prompt, &choice); err != nil {
				return err
			}

			switch choice {

			case options[0]:
				dir := fmt.Sprintf("ipm-%d", time.Now().Unix())
				if err := os.Mkdir(dir, 0755); err != nil {
					return err
				}

				fmt.Println(indent + color.New(color.FgYellow).
					Sprint("📦 Using isolated folder: "+dir))

				execDir = dir
				continue // 🔁 retry, execDir preserved

			case options[1]:
				if folder == "" {
					return fmt.Errorf("unable to determine existing folder name")
				}

				fmt.Println(indent + color.New(color.FgYellow).
					Sprint("📁 Using existing folder: "+folder))

				commands = utils.RemoveGitCloneCommands(commands)
				continue // 🔁 retry, execDir preserved

			}
		}
		// --- PEP 668 / EXTERNALLY MANAGED ENVIRONMENT HANDLER ---
		if utils.IsExternallyManagedError(output) {
			var choice string
				prompt := &survey.Select{
					Message: "System Python is externally managed. How would you like to proceed?",
					Options: []string{"Install via pipx (Recommended for CLI tools)", "Create/Use a virtual environment (.venv)", "Cancel"},
					Default: "Install via pipx (Recommended for CLI tools)",
				}

				if survey.AskOne(prompt, &choice) != nil || choice == "Cancel" {
					return fmt.Errorf("installation aborted by user")
				}

				// --- OPTION 1: PIPX ---
				if choice == "Install via pipx (Recommended for CLI tools)" {
					if !utils.CommandExists("pipx") {
						utils.DebugLog("Installing pipx....")
						utils.InstallPipx()
					}
				
					if utils.CommandExists("pipx") {
						for i, instr := range commands {
							cmd := instr.Command
							utils.DebugLog("Original cmd: %s", cmd)
					
							// 1. Handle the 'python -m pip' variants
							cmd = strings.ReplaceAll(cmd, "python3 -m pip install", "pipx install")
							cmd = strings.ReplaceAll(cmd, "python -m pip install", "pipx install")
					
							// 2. Handle 'pip3' and 'pip' variants
							// We use ReplaceAll to catch every instance in the script
							cmd = strings.ReplaceAll(cmd, "pip3 install", "pipx install")
							cmd = strings.ReplaceAll(cmd, "pip install", "pipx install")
					
							utils.DebugLog("Rewritten cmd: %s", cmd)
							commands[i].Command = cmd
						}
					
						// Lock the fix and refresh the PATH so the shell can find 'pipx'
						pythonFixed = true 
						updatedPath = os.Getenv("PATH") 
						continue 
					}
				}

				venvBase := execDir
				if venvBase == "" {
					venvBase = "."
				}

				pipPath := utils.GetVenvBinPath(venvBase, "pip")

				if pipPath == "" {
					fmt.Println(indent + color.New(color.FgYellow).Sprint("🛠 Creating new virtual environment..."))
					venvCmd := exec.Command("python3", "-m", "venv", ".venv")
					if execDir != "" {
						venvCmd.Dir = execDir
					}

					if vErr := venvCmd.Run(); vErr != nil {
						fmt.Println(indent + color.RedString("❌ Failed to create venv: %v", vErr))
						return vErr
					}
					pipPath = utils.GetVenvBinPath(venvBase, "pip")
				} else {
					fmt.Println(indent + color.New(color.FgCyan).Sprint("✨ Existing .venv found. Reusing it..."))
				}

				// Rewrite commands to use the venv-specific binaries
				// This makes the fix "Generic" for any python/pip instruction
				for i, instr := range commands {
					newCmd := strings.ReplaceAll(instr.Command, "pip install", pipPath+" install")
					// Handle 'python -m pip' or just 'python' calls
					pythonPath := utils.GetVenvBinPath(venvBase, "python")
					newCmd = strings.ReplaceAll(newCmd, "python3", pythonPath)
					newCmd = strings.ReplaceAll(newCmd, "python ", pythonPath+" ")

					commands[i].Command = newCmd
				}

				continue // 🔁 Retry loop with the updated commands

		}

		if isArgumentError(output) {
			fmt.Println(indent + color.New(color.FgYellow).
				Sprint("ℹ Command requires arguments; skipping failure"))
			return nil
		}

		if utils.IsPermissionError(output) {
			var retry bool
			msg := "Permission denied. Retry with sudo?"
			if runtime.GOOS == "windows" {
				msg = "Permission denied. Retry as Administrator?"
			}

			prompt := &survey.Confirm{
				Message: msg,
				Default: true,
			}

			if survey.AskOne(prompt, &retry) == nil && retry {
				fmt.Println(indent + color.New(color.FgYellow).Sprint("🔐 Requesting elevation..."))
				output, err = buildAndRun(true) // This now triggers the RunAs logic above
				if err == nil {
					fmt.Println(indent + color.New(color.FgGreen).
						Sprint("✓ Installation succeeded with elevated privileges"))
					return nil
				}
			}
		}

		// --- PIP / PIP3 ALTERNATIVE HANDLER ---
		// Check if the error is specifically about 'pip' or 'pip3' missing
		isPipMissing := regexp.MustCompile(`(?i)(pip\d?[:\s]+(?:command\s+)?not\s+found|pip\d?.*is not recognized)`).MatchString(output)


		if isPipMissing && !pipSwapped && !utils.IsExternallyManagedError(output) {
			changed := false
			for i, instr := range commands {
				original := instr.Command
				// Toggle between pip and pip3
				if strings.Contains(original, "pip ") {
					commands[i].Command = strings.Replace(original, "pip ", "pip3 ", 1)
					changed = true
				} else if strings.Contains(original, "pip3 ") {
					commands[i].Command = strings.Replace(original, "pip3 ", "pip ", 1)
					changed = true
				}
			}

			if changed {
				pipSwapped = true
				fmt.Println(indent + color.New(color.FgYellow).Sprint("ℹ Pip not found. Trying alternative (pip/pip3)..."))
				continue // 🔁 Retry loop with swapped pip command
			}
		}

		// 1. Check for missing git/npm/etc
		if pythonFixed {
			return err // Return the actual error if it failed even after fix
		}
		shouldRetry, newPath, depErr := prerequisites.HandleMissingDependencies(output)
		if depErr != nil {
			return depErr
		}
		if shouldRetry {
			if newPath != "" {
				updatedPath = newPath // ✅ Store it for the next buildAndRun
			}
			continue
		}

		if isInterrupted == false {
			fmt.Println(indent + color.New(color.FgRed).Sprint("✗ Command failed"))
			tracker.TrackInstallFailed(
				repo.Name,
				method.Title,
				strings.Join(utils.GetCommands(commands), " && "),
				output,
			)
		}
		// Return the original exec error (not depErr which may be nil)
		return err
	}
}

func InstallLocalFile(filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("❌ Error opening local file: %v\n", err)
		return
	}
	defer file.Close()

	var repo types.RepoDocumentFull
	byteValue, _ := io.ReadAll(file)

	if err := json.Unmarshal(byteValue, &repo); err != nil {
		fmt.Printf("❌ Error parsing JSON: %v\n", err)
		return
	}

	// Reuse the existing robust installation logic
	runInstallFlow(&repo)
}

func callUpdateMethodsAPI(repoName string, newMethods []types.InstallMethod) {
	// Define the endpoint URL - adjust based on your actual Endpoints config
	url := endpoints.Endpoints.UpdateEntry.Get()

	payload := map[string]interface{}{
		"repo":        repoName,
		"new_methods": newMethods,
	}

	body, _ := json.Marshal(payload)
	// Fire and forget or simple log on error
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err == nil {
		defer resp.Body.Close()
	}
}
