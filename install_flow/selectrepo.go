package install_flow

import (
	"bytes"
	"encoding/json"
	"fmt"
	"ipm/internal"
	"ipm/types"
	"ipm/utils"
	"math"
	"net/http"
	"os"
	"strings"

	endpoints "ipm/endpoints"
	live_generation "ipm/live_generation"
	tracker "ipm/tracker"

	"github.com/AlecAivazis/survey/v2"
	"github.com/agext/levenshtein"
	"github.com/dustin/go-humanize"
)

var IPM_AUTO_INDEX_API_URL = endpoints.Endpoints.AutoIndex.Get()

func triggerAutoIndexAPI(query string) {
	payload, err := json.Marshal(map[string]string{"search_term": query})
	if err != nil {
		return
	}

	resp, err := http.Post(IPM_AUTO_INDEX_API_URL, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return
	}
	defer resp.Body.Close()
}

// checkAutoIndex handles the similarity calculation and tracking
func checkAutoIndex(query string, matches []types.RepoDocumentFull) {
	var highestScore float64 = 0.0
	resultsFound := len(matches)
	var bestMatchName string = ""
	if resultsFound > 0 {
		for _, res := range matches {
			// Levenshtein.Distance returns an int of edits needed
			dist := levenshtein.Distance(strings.ToLower(query), strings.ToLower(res.Name), nil)

			// Convert distance to a 0.0 - 1.0 similarity ratio
			maxLen := math.Max(float64(len(query)), float64(len(res.Name)))
			score := 1.0
			if maxLen > 0 {
				score = 1.0 - (float64(dist) / maxLen)
			}

			if score > highestScore {
				highestScore = score
				bestMatchName = res.Name // Capture the name
			}
		}
	}

	if highestScore < 0.60 {
		triggerAutoIndexAPI(query)
	}
	tracker.TrackAutoIndex(query, resultsFound, highestScore, bestMatchName)
}

func SelectRepoMeili(query string) (*types.RepoDocumentFull, NavigationAction) {
	matches, err := internal.FuzzySearchMeili(query)

	handleSearchAnalytics(query, matches)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Search error: %v\n", err)
		return nil, NavigationActionBack
	}

	if len(matches) == 0 {
		fmt.Printf("\n🔍 Couldn't find a proper result for '%s', trying a broader search...\n", query)
		repo := live_generation.HandleGitHubFallback(query) // Catch the return
		if repo != nil {
			return repo, NavigationActionForward // Forward to installation
		}
		return nil, NavigationActionBack
	}

	// 2. Immediate return ONLY if search result name exactly matches the query
	if len(matches) == 1 && strings.EqualFold(matches[0].Name, query) {
		return &matches[0], NavigationActionForward
	}

	// 3. PASS 'query' to both functions below
	// Note: You will need to update the signatures of these two helper functions to accept 'query string'
	options := formatRepoOptions(matches, query)
	return promptRepoSelection(options, matches, query)
}

// --- Helper Functions ---

// ... (handleSearchAnalytics and formatRepoOptions remain unchanged) ...

func promptRepoSelection(options []string, repoList []types.RepoDocumentFull, query string) (*types.RepoDocumentFull, NavigationAction) {
	var selected string
	indent := "    "

	prompt := &survey.Select{
		Message:  "Select a repository:",
		Options:  options,
		PageSize: 10,
	}

	err := survey.AskOne(prompt, &selected)
	if err != nil || selected == indent+"[Back]" {
		checkAutoIndex(query, repoList)
		return nil, NavigationActionBack
	}

	// NEW: Handle the GitHub Search Option
	if selected == indent+"[🌍 Can’t find what you’re looking for? Try a broader search]" {
		repo := live_generation.HandleGitHubFallback(query)
		if repo != nil {
			return repo, NavigationActionForward // This triggers runInstallFlow
		}
		return nil, NavigationActionBack
	}

	for i, opt := range options {
		// We check against the index to ensure we don't match the special buttons
		if i < len(repoList) && opt == selected {
			return &repoList[i], NavigationActionForward
		}
	}

	return nil, NavigationActionBack
}

// --- Helper Functions ---

func handleSearchAnalytics(query string, matches []types.RepoDocumentFull) {
	top := make([]string, 0, 3)
	for i, r := range matches {
		if i >= 3 {
			break
		}
		top = append(top, r.Name)
	}
	tracker.TrackRepoSearch(query, "install", top)
}

func formatRepoOptions(matches []types.RepoDocumentFull, query string) []string {
	indent := "    "
	descIndent := indent + indent

	options := make([]string, 0, len(matches)+2)
	for _, r := range matches {
		// Handle stars conditionally
		starDisplay := ""
		if r.Stars != -1 {
			starDisplay = fmt.Sprintf(" ⭐ %s", humanize.Comma(int64(r.Stars)))
		}

		preview := fmt.Sprintf(
			"%s%s\t%s [%s]\n%s%s",
			indent, r.Name, starDisplay, r.RepoType,
			descIndent+indent, utils.TruncateDesc(r.Description),
		)
		options = append(options, preview)
	}

	options = append(options, indent+"[🌍 Can’t find what you’re looking for? Try a broader search]")
	options = append(options, indent+"[Back]")
	return options
}
