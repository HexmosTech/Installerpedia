package live_generation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"ipm/types"
	"ipm/utils"
	"net/http"
	"net/url"
	"strings"
	"time"

	endpoints "ipm/endpoints"
	tracker "ipm/tracker"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"
)

var IPM_ADD_API_URL = endpoints.Endpoints.AddEntry.Get()
var IPM_GENERATE_REPO_API_URL = endpoints.Endpoints.GenerateRepo.Get()
type IPMPayload struct {
	Repo                string      `json:"repo"`
	RepoType            string      `json:"repo_type"`
	HasInstallation     bool        `json:"has_installation"`
	Keywords            []string    `json:"keywords"`
	Prerequisites       interface{} `json:"prerequisites"`
	InstallationMethods interface{} `json:"installation_methods"`
	PostInstallation    interface{} `json:"post_installation"`
	ResourcesOfInterest interface{} `json:"resources_of_interest"` // Add this line
	Description         string      `json:"description"`
	Stars               int         `json:"stars"`
}

type RepologyProject struct {
	Repo    string `json:"repo"`
	Version string `json:"version"`
	Summary string `json:"summary"`
}
type RepologyResponse map[string][]RepologyProject

func fetchRepologyRepos(query string) (RepologyResponse, error) {
    url := fmt.Sprintf("https://repology.org/api/v1/projects/?search=%s", url.QueryEscape(query))
    
    // LOG: Track the outgoing request
    utils.DebugLog("Fetching Repology data from: %s", url)

    client := &http.Client{Timeout: 10 * time.Second}
    req, _ := http.NewRequest("GET", url, nil)
    req.Header.Set("User-Agent", "Mozilla/5.0 (IPM-CLI-Tool; contact@example.com)")

    resp, err := client.Do(req)
    if err != nil {
        // CHANGED: fmt -> utils.DebugLog
        utils.DebugLog("❌ Repology Connection Error: %v", err)
        return nil, err
    }
    defer resp.Body.Close()

    // LOG: Check status code
    utils.DebugLog("Repology API response status: %d", resp.StatusCode)

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("bad status: %d", resp.StatusCode)
    }

    var fullResult RepologyResponse
    if err := json.NewDecoder(resp.Body).Decode(&fullResult); err != nil {
        // CHANGED: fmt -> utils.DebugLog
        utils.DebugLog("❌ Repology Decode Error: %v", err)
        return nil, err
    }

    // LOG: Total results found before limiting
    utils.DebugLog("Repology returned %d total project(s)", len(fullResult))

    // --- Limit to 10 entries ---
    limitedResult := make(RepologyResponse)
    count := 0
    for pkg, project := range fullResult {
        if count >= 10 {
            break
        }
        limitedResult[pkg] = project
        count++
    }

    // LOG: Final count being returned to the TUI
    utils.DebugLog("Returning %d limited Repology entries to selector", len(limitedResult))

    return limitedResult, nil
}

// Define a minimal struct for GitHub's API response
type GitHubSearchResponse struct {
	Items []struct {
		FullName    string `json:"full_name"`
		Description string `json:"description"`
		Stars       int    `json:"stargazers_count"`
	} `json:"items"`
}

func fetchGeneratedRepo(repoName, sourceType string) (string, error) {
	payload, _ := json.Marshal(map[string]string{"repo": repoName, "source_type": sourceType})
	resp, err := http.Post(IPM_GENERATE_REPO_API_URL, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	return string(body), err
}

func HandleGitHubFallback(query string) *types.RepoDocumentFull {
	fmt.Printf("\n🔍 Searching for '%s'...\n", query)

	// Fetch both sources
	repologyItems, _ := fetchRepologyRepos(query)
	utils.DebugLog("Fetched Repology")

	githubItems, err := fetchGitHubRepos(query)
	utils.DebugLog("Fetched Github")

	utils.DebugLog("Fetched %d Repology packages and %d GitHub items", len(repologyItems), len(githubItems))
	if err != nil {
		fmt.Printf("❌ %v\n", err)
		return nil
	}

	var options []string
	indent := "    "

	// Add Repology first
	for pkg, entries := range repologyItems {
		if len(entries) > 0 {
			options = append(options, fmt.Sprintf("%s%-40s [Repology]\n%s%s",
				indent, pkg, indent+indent+indent, utils.TruncateDesc(entries[0].Summary)))
		}
	}

	// Add GitHub second
	ghOptions := formatGitHubOptions(githubItems)
	for _, opt := range ghOptions {
		if !strings.Contains(opt, "[Back]") {
			options = append(options, opt)
		}
	}
	options = append(options, indent+"[Back]")

	// Pass options to enhanced TUI selector
	selectedLabel := promptGitHubSelection(options, repologyItems, githubItems)
	if selectedLabel == "" {
		return nil
	}

	var sourceType = "github"
	var selectedRepo, selectedDesc string
	var selectedStars int

	// Check Repology matches
	if entries, exists := repologyItems[selectedLabel]; exists {
		sourceType = "repology"
		selectedRepo = selectedLabel
		selectedDesc = entries[0].Summary
	} else {
		// Fallback to GitHub
		for _, item := range githubItems {
			if selectedLabel == item.FullName {
				selectedRepo = item.FullName
				selectedDesc = item.Description
				selectedStars = item.Stars
				break
			}
			utils.DebugLog("User selected: %s | Source: %s | Stars: %d", selectedRepo, sourceType, selectedStars)
		}
	}

	// Processing logic with 3 retries
    var rawJson string
    var fetchErr error

    for i := 1; i <= 3; i++ {
        rawJson, fetchErr = fetchGeneratedRepo(selectedRepo, sourceType)
        if fetchErr != nil {
			utils.DebugLog("Fetch attempt %d failed: %v", i, fetchErr)
		} else {
			utils.DebugLog("Fetch attempt %d succeeded. Data size: %d bytes", i, len(rawJson))
		}
		if fetchErr == nil {
            break
        }
        fmt.Printf("⚠️ Error fetching (attempt %d/3): %v. Retrying...\n", i, fetchErr)
        if i < 3 {
            time.Sleep(1 * time.Second) // Brief pause before retry
        }
    }

    if fetchErr != nil {
        fmt.Printf("❌ Failed to fetch after 3 attempts. Aborting.\n")
        return nil
    }

    cleanJson := strings.TrimSpace(strings.TrimPrefix(strings.TrimSuffix(rawJson, "```"), "```json"))
    
    // Fix: Always initialize the map to prevent "assignment to nil" panic
    tempMap := make(map[string]interface{})
    _ = json.Unmarshal([]byte(cleanJson), &tempMap) 

    // These assignments are now safe because tempMap is initialized
    tempMap["name"] = selectedRepo
    tempMap["description"] = selectedDesc
    tempMap["stars"] = selectedStars
	utils.DebugLog("Metadata merged into JSON for: %s", selectedRepo)

    mergedBytes, _ := json.MarshalIndent(tempMap, "", "    ")
	
	
	var repo types.RepoDocumentFull
	json.Unmarshal(mergedBytes, &repo)

	tracker.TrackInstallationJsonGenerated(selectedRepo, string(mergedBytes), sourceType)
	storeIPMJsonToDB(string(mergedBytes))
	utils.DebugLog("Finalizing HandleGitHubFallback for %s. Returning &repo.", selectedRepo)
	return &repo
}

// --- Bubble Tea based selector with section jumping shortcuts ---

type repoItem struct {
	title       string
	desc        string
	section     string // "repology" or "github" or "back"
	rawLabel    string // for repology: package name, for github: full name, for back: ""
	isBackEntry bool
}

func (r repoItem) Title() string       { return r.title }
func (r repoItem) Description() string { return r.desc }
func (r repoItem) FilterValue() string { return r.title + " " + r.desc }

type repoListModel struct {
	list          list.Model
	sectionRanges map[string][2]int // section -> [start, end)
	choice        *repoItem
	quit          bool
}

func (m repoListModel) Init() tea.Cmd {
	return nil
}

func (m repoListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quit = true
			return m, tea.Quit
		case "r":
			if rg, ok := m.sectionRanges["repology"]; ok {
				m.list.Select(rg[0])
			}
			return m, nil
		case "g":
			if rg, ok := m.sectionRanges["github"]; ok {
				m.list.Select(rg[0])
			}
			return m, nil
		case "enter":
			if it, ok := m.list.SelectedItem().(repoItem); ok {
				if it.isBackEntry {
					m.choice = nil
				} else {
					ci := it
					m.choice = &ci
				}
				return m, tea.Quit
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m repoListModel) View() string {
	// Very close to survey.Select: just render the list with a subtle help footer.
	return m.list.View()
}

func promptGitHubSelection(options []string, repologyItems RepologyResponse, githubItems []struct {
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	Stars       int    `json:"stargazers_count"`
}) string {
	indent := "    "

	// Build list items with section information
	items := make([]list.Item, 0, len(options))
	sectionRanges := map[string][2]int{}

	// Repology section first
	startRepology := len(items)
	for pkg, entries := range repologyItems {
		if len(entries) == 0 {
			continue
		}
		title := fmt.Sprintf("%s%-40s [Repology]", indent, pkg)
		desc := fmt.Sprintf("%s%s", indent+indent+indent, utils.TruncateDesc(entries[0].Summary))
		items = append(items, repoItem{
			title:       title,
			desc:        desc,
			section:     "repology",
			rawLabel:    pkg,
			isBackEntry: false,
		})
	}
	endRepology := len(items)
	if endRepology > startRepology {
		sectionRanges["repology"] = [2]int{startRepology, endRepology}
	}

	// GitHub section
	startGitHub := len(items)
	for _, gh := range githubItems {
		starCount := humanize.Comma(int64(gh.Stars))
		title := fmt.Sprintf("%s%-40s [Github] ⭐ %s", indent, gh.FullName, starCount)
		desc := fmt.Sprintf("%s%s", indent+indent+indent, utils.TruncateDesc(gh.Description))
		items = append(items, repoItem{
			title:       title,
			desc:        desc,
			section:     "github",
			rawLabel:    gh.FullName,
			isBackEntry: false,
		})
	}
	endGitHub := len(items)
	if endGitHub > startGitHub {
		sectionRanges["github"] = [2]int{startGitHub, endGitHub}
	}

	// Back entry
	backItem := repoItem{
		title:       indent + "[Back]",
		desc:        "",
		section:     "back",
		rawLabel:    "",
		isBackEntry: true,
	}
	items = append(items, backItem)

	// Configure list to resemble survey.Select
	const defaultWidth = 80

	// Custom delegate: light cyan (#32DAD6) text and selector for selected item.
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		BorderForeground(lipgloss.Color("#32DAD6")).
		Foreground(lipgloss.Color("#32DAD6")).
		UnsetBackground()
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		BorderForeground(lipgloss.Color("#32DAD6")).
		Foreground(lipgloss.Color("#32DAD6")).
		UnsetBackground()
	delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.UnsetBackground()
	delegate.Styles.NormalDesc = delegate.Styles.NormalDesc.UnsetBackground()
	delegate.SetSpacing(0)

	// Increase height so more entries are visible (each item uses two lines).
	l := list.New(items, delegate, defaultWidth, 40)
	l.SetShowHelp(true)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)

	// Remove solid backgrounds from title, help, and filter areas.
	styles := list.DefaultStyles()
	styles.TitleBar = styles.TitleBar.UnsetBackground()
	styles.Title = styles.Title.UnsetBackground()
	styles.PaginationStyle = styles.PaginationStyle.UnsetBackground()
	styles.HelpStyle = styles.HelpStyle.UnsetBackground()
	l.Styles = styles

	l.FilterInput.Prompt = "Filter: "
	l.Title = "Search Results (press 'r' for Repology, 'g' for GitHub, ↑/↓ to move, enter to select)"

	m := repoListModel{
		list:          l,
		sectionRanges: sectionRanges,
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return ""
	}

	rm, ok := finalModel.(repoListModel)
	if !ok || rm.choice == nil {
		return ""
	}

	return rm.choice.rawLabel
}

func fetchGitHubRepos(query string) ([]struct {
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	Stars       int    `json:"stargazers_count"`
}, error) {
	url := fmt.Sprintf("https://api.github.com/search/repositories?q=%s&per_page=10", query)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("GitHub search failed: %v", err)
	}
	defer resp.Body.Close()

	var result GitHubSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("Failed to parse GitHub response: %v", err)
	}
	return result.Items, nil
}

func formatGitHubOptions(items []struct {
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	Stars       int    `json:"stargazers_count"`
}) []string {
	indent := "    "
	options := make([]string, 0, len(items)+1)
	for _, item := range items {
		starCount := humanize.Comma(int64(item.Stars))

		// Change: Use %-40s to pad the name to 40 characters
		preview := fmt.Sprintf("%s%-40s [Github] ⭐ %s\n%s%s",
			indent, item.FullName, starCount,
			indent+indent+indent, utils.TruncateDesc(item.Description))
		options = append(options, preview)
	}
	return append(options, indent+"[Back]")
}

func storeIPMJsonToDB(jsonStr string) error {
	// Since metadata is already injected, we convert directly to bytes
	body := []byte(jsonStr)

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	req, err := http.NewRequest("POST", IPM_ADD_API_URL, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	// Keep your specific User-Agent as it might be required by your API/WAF
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("could not connect to server: %w", err)
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server error (%d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}
