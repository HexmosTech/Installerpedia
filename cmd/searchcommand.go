package main

import (
	"fmt"
	install_flow "ipm/install_flow"
	"ipm/internal"
	"ipm/tracker"
	"ipm/types"
	utils "ipm/utils"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/dustin/go-humanize"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"golang.org/x/term"
)

func searchCommand(query string) {
	matches, err := internal.FuzzySearchMeili(query)
	if err != nil {
		panic(err)
	}

	if len(matches) == 0 {
		fmt.Println("No matching repos found.")
		return
	}

	// 🔍 Collect top 3 search results for tracking
	top := make([]string, 0, 3)
	for i, r := range matches {
		if i >= 3 {
			break
		}
		top = append(top, r.Name)
	}

	// 📡 Track search event
	tracker.TrackRepoSearch(query, "search", top)

	// Render table (same as before)
	termWidth, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		termWidth = 80
	}

	nameW, starsW, typeW := 25, 8, 12
	descW := int(float64(termWidth) * 0.25)

	var buf strings.Builder
	t := table.NewWriter()
	t.SetOutputMirror(&buf)
	t.SetStyle(table.StyleLight)
	t.Style().Options.SeparateRows = true

	t.AppendHeader(table.Row{"REPO NAME", "DESCRIPTION", "STARS", "REPO TYPE"})
	t.SetColumnConfigs([]table.ColumnConfig{
		{Name: "REPO NAME", WidthMax: nameW},
		{Name: "DESCRIPTION", WidthMax: descW, Transformer: func(v interface{}) string {
			return text.WrapSoft(fmt.Sprint(v), descW)
		}},
		{Name: "STARS", WidthMax: starsW},
		{Name: "REPO TYPE", WidthMax: typeW},
	})

	for _, repo := range matches {
		desc := repo.Description
		if desc == "" {
			desc = "-"
		} else {
			desc = utils.TruncateDesc(desc)
		}
		t.AppendRow(table.Row{
			repo.Name,
			desc,
			humanize.Comma(int64(repo.Stars)),
			repo.RepoType,
		})
	}

	t.Render()
	fmt.Print(buf.String())

	// After rendering the search results
	var installChoice string
	prompt := &survey.Select{
		Message: "What do you want to do?",
		Options: []string{
			"Install a repo",
			"Exit",
		},
		Default: "Install a repo",
	}
	err = survey.AskOne(prompt, &installChoice)
	if err != nil {
		fmt.Println("Prompt canceled.")
		return
	}

	switch installChoice {
	case "Install a repo":
		// Pass search results to installCommand so user can pick without searching again
		// Convert []RepoDocumentFull -> []*RepoDocumentFull
		matchesPtrs := make([]*types.RepoDocumentFull, len(matches))
		for i := range matches {
			matchesPtrs[i] = &matches[i]
		}

		install_flow.InstallCommandWithMatches(matchesPtrs)
	case "Exit":
		return
	}
}
