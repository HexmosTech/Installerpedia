package live_generation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"ipm/types"
    tracker "ipm/tracker"
    endpoints "ipm/endpoints"

)
var IPM_GENERATE_REPO_METHOD_API_URL = endpoints.Endpoints.GenerateRepoMethod.Get()

// fetchGeneratedRepoMethod calls the single-method endpoint with OS context
func fetchGeneratedRepoMethod(repoName, sourceType, osFamily string) (string, error) {
    payload, err := json.Marshal(map[string]string{
        "repo":        repoName,
        "source_type": sourceType,
        "os_family":   osFamily,
    })
    if err != nil {
        return "", err
    }

    resp, err := http.Post(IPM_GENERATE_REPO_METHOD_API_URL, "application/json", bytes.NewBuffer(payload))
    if err != nil {
        return "", err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
    }

    body, err := io.ReadAll(resp.Body)
    return string(body), err
}

// ProcessGeneratedRepoMethod handles the raw JSON and converts it to the shared types.InstallationMethod
func ProcessGeneratedRepoMethod(repoName, sourceType, osFamily string) ([]types.InstallMethod, error) {
    rawJson, err := fetchGeneratedRepoMethod(repoName, sourceType, osFamily)
    if err != nil {
        return nil, err
    }

    // Clean Markdown fences if the AI included them
    cleanJson := strings.TrimSpace(rawJson)
    cleanJson = strings.TrimPrefix(cleanJson, "```json")
    cleanJson = strings.TrimPrefix(cleanJson, "```")
    cleanJson = strings.TrimSuffix(cleanJson, "```")

    // The endpoint returns {"installation_methods": [...]}
    var wrapper struct {
        Methods []types.InstallMethod `json:"installation_methods"`
    }

    if err := json.Unmarshal([]byte(cleanJson), &wrapper); err != nil {
        return nil, fmt.Errorf("failed to unmarshal method: %w", err)
    }

	// --- TRACKER CALL ---
    // Log that we generated a single method specifically for this OS
    tracker.TrackInstallationJsonGenerated(repoName, cleanJson, sourceType+"_single_method_"+osFamily)
    // --------------------

    return wrapper.Methods, nil
}