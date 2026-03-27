package tracker

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"ipm/types"
	utils "ipm/utils"
	"math"
	"net/http"
	"runtime"
	"strings"
	"time"
)

var (
	OS   = runtime.GOOS
	ARCH = runtime.GOARCH
)

var TrackingEnabled = true

// postHogClient with timeout to prevent hanging requests
var postHogClient = &http.Client{
	Timeout: 15 * time.Second,
}

// Granular control for each event type
var EnabledTrackers = map[string]bool{
	"ipm_install_repo_failed":         true,
	"ipm_install_repo_success":        true,
	"ipm_repo_search_event":           true,
	"ipm_update_success":              true,
	"ipm_installation_json_generated": true,
	"ipm_auto_index":                  true,
	"ipm_install_repo_cancelled":      true,
}

// Internal helper to handle the HTTP request and the global/local toggles
func sendPostHogEvent(eventName string, payload map[string]interface{}) {
	utils.DebugLog("[TRACKING DEBUG] sendPostHogEvent called for: %s at %s\n", eventName, time.Now().Format("15:04:05.000"))

	if !TrackingEnabled {
		utils.DebugLog("[TRACKING DEBUG] Tracking is DISABLED globally\n")
		return
	}
	if !EnabledTrackers[eventName] {
		utils.DebugLog("[TRACKING DEBUG] Event %s is DISABLED in EnabledTrackers\n", eventName)
		return
	}
	utils.DebugLog("[TRACKING DEBUG] Tracking enabled, event enabled. Proceeding...\n")
	utils.DebugLog("Tracking... %s\n", eventName)

	utils.DebugLog("[TRACKING DEBUG] Marshaling payload...\n")
	data, err := json.Marshal(payload)
	if err != nil {
		utils.DebugLog("[TRACKING DEBUG] ERROR: failed to marshal payload: %v\n", err)
		return
	}
	utils.DebugLog("[TRACKING DEBUG] Payload marshaled, size: %d bytes\n", len(data))
	utils.DebugLog("Tracking... %v\n", data)

	utils.DebugLog("[TRACKING DEBUG] Creating HTTP request...\n")
	// Use context.Background() to ensure the request isn't cancelled by signal handlers
	// The HTTP client timeout (15s) will handle the timeout
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://us.i.posthog.com/i/v0/e/", bytes.NewBuffer(data))
	if err != nil {
		utils.DebugLog("[TRACKING DEBUG] ERROR: failed to create request: %v\n", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	utils.DebugLog("[TRACKING DEBUG] HTTP request created. About to send (this will block up to 15s)...\n")

	startTime := time.Now()
	// This will block until the request completes, fails, or times out (15s)
	resp, err := postHogClient.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		utils.DebugLog("[TRACKING DEBUG] ERROR: HTTP request failed after %v: %v\n", duration, err)
		return
	}
	utils.DebugLog("[TRACKING DEBUG] HTTP request completed successfully after %v. Status: %s\n", duration, resp.Status)

	defer resp.Body.Close()
	// Drain the body to allow connection reuse
	utils.DebugLog("[TRACKING DEBUG] Draining response body...\n")
	io.Copy(io.Discard, resp.Body)
	utils.DebugLog("[TRACKING DEBUG] Tracking completed successfully at %s\n", time.Now().Format("15:04:05.000"))
}

func TrackInstallFailed(repoName, method, cmd, errorTrace string) {
	eventName := "ipm_install_repo_failed"
	payload := map[string]interface{}{
		"api_key":     "phc_bC7cMka8DieEik61bxec1xAg3hANE8oNNGoelwXoE9I",
		"event":       eventName,
		"distinct_id": utils.GetMachineDistinctID(),
		"properties": map[string]string{
			"reponame": repoName,
			"method":   method,
			"os":       utils.GetSupplementedOS(),
			"arch":     ARCH,
			"command":  cmd,
			"error":    errorTrace,
		},
	}
	sendPostHogEvent(eventName, payload)
}

func TrackInstallCancelled(repoName, method, cmd, errorTrace string) {
	utils.DebugLog("[TRACKING DEBUG] ===== trackInstallCancelled CALLED at %s =====\n", time.Now().Format("15:04:05.000"))
	utils.DebugLog("[TRACKING DEBUG] Repo: %s, Method: %s, OS: %s, Arch: %s\n", repoName, method)

	eventName := "ipm_install_repo_cancelled"
	payload := map[string]interface{}{
		"api_key":     "phc_bC7cMka8DieEik61bxec1xAg3hANE8oNNGoelwXoE9I",
		"event":       eventName,
		"distinct_id": utils.GetMachineDistinctID(),
		"properties": map[string]string{
			"reponame": repoName,
			"method":   method,
			"os":       utils.GetSupplementedOS(),
			"arch":     ARCH,
			"command":  cmd,
			"error":    errorTrace,
		},
	}
	utils.DebugLog("[TRACKING DEBUG] Payload prepared. About to call sendPostHogEvent (this will BLOCK)...\n")

	startTime := time.Now()
	// Call synchronously - the HTTP client has a 15s timeout, so this will block
	// until the request completes, fails, or times out. This ensures the request
	// actually fires before the signal handler continues.
	sendPostHogEvent(eventName, payload)
	duration := time.Since(startTime)

	utils.DebugLog("[TRACKING DEBUG] sendPostHogEvent returned after %v at %s\n", duration, time.Now().Format("15:04:05.000"))
	utils.DebugLog("[TRACKING DEBUG] ===== trackInstallCancelled COMPLETE =====\n")
}

//utils.GetSupplementedOS(),
// ARCH,

func TrackInstallSuccess(repoName, method string, commands []types.Instruction) {
	eventName := "ipm_install_repo_success"
	payload := map[string]interface{}{
		"api_key":     "phc_bC7cMka8DieEik61bxec1xAg3hANE8oNNGoelwXoE9I",
		"event":       eventName,
		"distinct_id": utils.GetMachineDistinctID(),
		"properties": map[string]string{
			"reponame": repoName,
			"method":   method,
			"os":       utils.GetSupplementedOS(),
			"arch":     ARCH,
			"command":  strings.Join(utils.GetCommands(commands), " && "),
		},
	}
	sendPostHogEvent(eventName, payload)
}

func TrackRepoSearch(query, source string, topResults []string) {
	eventName := "ipm_repo_search_event"
	payload := map[string]interface{}{
		"api_key":     "phc_bC7cMka8DieEik61bxec1xAg3hANE8oNNGoelwXoE9I",
		"event":       eventName,
		"distinct_id": utils.GetMachineDistinctID(),
		"properties": map[string]interface{}{
			"query":             query,
			"source":            source,
			"os":                utils.GetSupplementedOS(),
			"arch":              ARCH,
			"top_results":       topResults,
			"top_results_count": len(topResults),
		},
	}
	sendPostHogEvent(eventName, payload)
}

func TrackAutoIndex(query string, found int, similarity float64, bestMatch string) {
	eventName := "ipm_auto_index"

	// Categorize the result for easier PostHog filtering
	status := "no_results"
	if found > 0 {
		if similarity < 0.60 {
			status = "low_similarity"
		} else {
			status = "high_similarity_abandoned"
		}
	}

	roundedSimilarity := math.Round(similarity*100) / 100
	roundedSimilarity = roundedSimilarity * 100

	payload := map[string]interface{}{
		"api_key":     "phc_bC7cMka8DieEik61bxec1xAg3hANE8oNNGoelwXoE9I",
		"event":       eventName,
		"distinct_id": utils.GetMachineDistinctID(),
		"properties": map[string]interface{}{
			"query":           query,
			"os":              utils.GetSupplementedOS(),
			"arch":            ARCH,
			"results_present": found,
			"best_similarity": roundedSimilarity,
			"best_match_name": bestMatch,
			"match_status":    status,
		},
	}
	sendPostHogEvent(eventName, payload)
}

func TrackUpdateSuccess(oldVersion, newVersion string) {
	eventName := "ipm_update_success"
	payload := map[string]interface{}{
		"api_key":     "phc_bC7cMka8DieEik61bxec1xAg3hANE8oNNGoelwXoE9I",
		"event":       eventName,
		"distinct_id": utils.GetMachineDistinctID(),
		"properties": map[string]string{
			"old_version": oldVersion,
			"new_version": newVersion,
			"os":          utils.GetSupplementedOS(),
			"arch":        ARCH,
		},
	}
	sendPostHogEvent(eventName, payload)
}

func TrackInstallationJsonGenerated(repoName, rawJson, sourceType string) {
	eventName := "ipm_installation_json_generated"
	var jsonParsed interface{}
	err := json.Unmarshal([]byte(rawJson), &jsonParsed)

	if err != nil {
		jsonParsed = rawJson
	}

	payload := map[string]interface{}{
		"api_key":     "phc_bC7cMka8DieEik61bxec1xAg3hANE8oNNGoelwXoE9I",
		"event":       eventName,
		"distinct_id": utils.GetMachineDistinctID(),
		"properties": map[string]interface{}{
			"reponame":    repoName,
			"raw_json":    jsonParsed,
			"source_type": sourceType,
			"os":          utils.GetSupplementedOS(),
			"arch":        ARCH,
			"is_custom":   true,
		},
	}
	sendPostHogEvent(eventName, payload)
}
