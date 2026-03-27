package main

import (
	"encoding/json"
	"fmt"
	"io"
	"ipm/version"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	endpoints "ipm/endpoints"
	install_flow "ipm/install_flow"
	utils "ipm/utils"

	"github.com/fatih/color"
)

var (
	OS   = runtime.GOOS
	ARCH = runtime.GOARCH
)

func printFeatured() {
	url := endpoints.Endpoints.Featured.Get()

	client := http.Client{Timeout: 800 * time.Millisecond}
	resp, err := client.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	var data struct {
		Message string `json:"message"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return
	}

	if data.Message != "" {
		// 1. Define the Golden Yellow style
		gold := color.New(color.FgHiYellow, color.Bold)

		// 2. Use a sleek star instead of a blocky badge
		// 3. Print message directly so terminals auto-link the URL
		fmt.Printf("%s %s\n\n", gold.Sprint("★"), gold.Sprint(data.Message))
	}
}

func handleDebugCommand(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: ipm debug on | ipm debug off")
		fmt.Println("  on   Enable debug mode (writes to ~/.ipm.toml)")
		fmt.Println("  off  Disable debug mode (writes to ~/.ipm.toml)")
		return
	}
	arg := strings.ToLower(args[0])
	switch arg {
	case "on":
		utils.GlobalConfig.Debug = true
		if err := utils.SaveConfig(); err != nil {
			fmt.Printf("Failed to save config: %v\n", err)
			return
		}
		fmt.Println(color.GreenString("Debug mode enabled."))
	case "off":
		utils.GlobalConfig.Debug = false
		if err := utils.SaveConfig(); err != nil {
			fmt.Printf("Failed to save config: %v\n", err)
			return
		}
		fmt.Println("Debug mode disabled.")
	default:
		fmt.Println("Usage: ipm debug on | ipm debug off")
	}
}

func main() {
	// Run auto update for all normal commands
	// if len(os.Args) > 1 && os.Args[1] != "version" && os.Args[1] != "h" && os.Args[1] != "help" {
	// 	printFeatured()
	// }
	utils.LoadConfig() // Initialize configurations
	if len(os.Args) > 1 {
		cmd := strings.ToLower(os.Args[1])
		if cmd != "update" && cmd != "setup" && cmd != "version" && cmd != "help" && cmd != "h" && cmd != "debug" {
			updateCommand()
		}
	}

	showHelp := func() {
		fmt.Println("Usage: ipm <command> <repo name>")
		fmt.Println("Commands:")
		fmt.Println("  install, i   Install a repository")
		fmt.Println("  show, sh     Show repository installation info")
		fmt.Println("  search, s    Search for a repository")
		fmt.Println("  help, h      Show this help message")
		fmt.Println("  update       Check for updates and apply them")
		fmt.Println("  setup        Setup the permanent ipm installation")
		fmt.Println("  version      Show the current version of ipm")
		fmt.Println("  debug        Toggle debug mode: ipm debug on | ipm debug off")
	}

	if len(os.Args) < 2 {
		showHelp()
		os.Exit(0)
	}

	cmd := strings.ToLower(os.Args[1])
	args := os.Args[2:]

	if cmd == "help" || cmd == "h" {
		showHelp()
		return
	}

	if cmd == "update" {
		updateCommand()
		return
	}

	if cmd == "setup" {
		setupCommand()
		return
	}

	if cmd == "version" {
		version.PrintVersion()
		return
	}

	if cmd == "debug" {
		handleDebugCommand(args)
		return
	}

	// Always show help if repo name is missing
	if len(args) == 0 {
		showHelp()
		return
	}

	query := strings.Join(args, " ")

	switch cmd {
	case "install", "i":
		if len(args) >= 2 && args[0] == "--local" {
			filePath := args[1]
			install_flow.InstallLocalFile(filePath)
		} else {
			install_flow.InstallCommand(query)
		}
	case "show", "sh":
		showCommand(query)
	case "search", "s":
		searchCommand(query)
	default:
		showHelp()
	}
}
