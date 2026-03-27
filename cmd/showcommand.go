package main

import (
	"fmt"
	install_flow "ipm/install_flow"
	"strings"

	"github.com/fatih/color"
)

// cmd/show.go or wherever your showCommand lives
func showCommand(query string) {
	repo, action := install_flow.SelectRepoMeili(query)
	if action == install_flow.NavigationActionBack {
		return
	}

	// Your original beautiful printing code — 100% unchanged
	whiteBold := color.New(color.FgWhite, color.Bold).SprintFunc()
	yellowBold := color.New(color.FgYellow, color.Bold).SprintFunc()
	greenBold := color.New(color.FgGreen, color.Bold).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()
	cyanBold := color.New(color.FgCyan, color.Bold).SprintFunc()

	fmt.Printf("\n%s\n", strings.Repeat("=", 60))
	fmt.Printf("%s %s\n", whiteBold("Repo:"), repo.Name)
	fmt.Printf("%s %s\n", whiteBold("Type:"), repo.RepoType)
	fmt.Printf("%s %s\n", whiteBold("Stars:"), cyanBold(fmt.Sprintf("%d", repo.Stars)))

	if repo.Description != "" {
		fmt.Printf("%s %s\n", whiteBold("Description:"), dim(repo.Description))
	}
	if repo.Note != "" {
		fmt.Printf("%s %s\n", whiteBold("Note:"), dim(repo.Note))
	}
	fmt.Println(strings.Repeat("=", 60))

	if len(repo.Prerequisites) > 0 {
		fmt.Println("\n" + yellowBold("Prerequisites:"))
		for _, p := range repo.Prerequisites {
			fmt.Printf(" - %s", whiteBold(p.Name))
			if p.Version != "" {
				fmt.Printf(" (version: %s)", p.Version)
			}
			fmt.Printf("\n")
		}
	}

	if len(repo.InstallationMethods) > 0 {
		fmt.Println("\n" + yellowBold("Installation Methods:"))
		for _, m := range repo.InstallationMethods {
			fmt.Printf(" - %s\n", whiteBold(m.Title))
			for _, instr := range m.Instructions {
				fmt.Printf("     Command: %s\n", greenBold(instr.Command))
				// if instr.Meaning != "" {
				// 	fmt.Printf("         %s\n", dim("→ "+instr.Meaning))
				// }
			}
		}
	}

	if len(repo.PostInstallation) > 0 {
		fmt.Println("\n" + yellowBold("Post Installation Notes:"))
		for i, note := range repo.PostInstallation {
			fmt.Printf(" %d. %s\n", i+1, whiteBold(note))
		}
	}

}
