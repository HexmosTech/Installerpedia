package internal

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/joho/godotenv"
	"github.com/meilisearch/meilisearch-go"
	"ipm/types"
)

const MEILI_SEARCH_API_KEY = "1038cd79387c4c2923df4e90e8f7ac3e760ab842fed759fb9f68ae8f7a95d0f8"


// Global client (lazy-init; uses Connect to verify server)
var meiliClient meilisearch.ServiceManager

func getMeiliClient() meilisearch.ServiceManager {
	if meiliClient != nil {
		return meiliClient
	}

	// Load .env if exists (ignored in prod if already set)
	_ = godotenv.Load() // <-- this is the magic line

	apiKey := MEILI_SEARCH_API_KEY
	if apiKey == "" {
		panic("MEILI_SEARCH_API_KEY is missing (check .env or environment)")
	}

	host := "https://search.apps.hexmos.com" // default fallback

	var err error
	meiliClient, err = meilisearch.Connect(
		host,
		meilisearch.WithAPIKey(apiKey),
	)
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to Meilisearch: %v", err))
	}

	return meiliClient
}

// RepoDocument matches your JSON structure exactly

// FuzzySearchMeili: Scoped to installerpedia only; fuzzy matches on query
func FuzzySearchMeili(query string) ([]types.RepoDocumentFull, error) {
	client := getMeiliClient()
	index := client.Index("freedevtools")

	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}

	req := &meilisearch.SearchRequest{
		Limit:           30,
		Filter:          "category = installerpedia",
		MatchingStrategy: "all",
		
	}

	searchRes, err := index.Search(query, req)
	if err != nil {
		return nil, fmt.Errorf("meilisearch search failed: %w", err)
	}

	var results []types.RepoDocumentFull
	for _, hit := range searchRes.Hits {
		blob, err := json.Marshal(hit)
		if err != nil {
			continue
		}
		var doc types.RepoDocumentFull
		if err := json.Unmarshal(blob, &doc); err != nil {
			continue
		}
		results = append(results, doc)
	}

	// fallback: multi-word → single word
	if len(results) == 0 && strings.Contains(query, " ") {
		parts := strings.Fields(query)
		return FuzzySearchMeili(parts[0])
	}

	return results, nil
}


// internal/search/meili.go

// internal/search/meili.go

func GetFullRepoByID(id string) (*types.RepoDocumentFull, error) {
	if id == "" {
		return nil, fmt.Errorf("empty id")
	}

	res, err := getMeiliClient().Index("freedevtools").Search("", &meilisearch.SearchRequest{
		Filter:               fmt.Sprintf("id = \"%s\"", id), // ← QUOTES AROUND THE ID!
		Limit:                1,
		AttributesToRetrieve: []string{"*"},
	})
	if err != nil {
		return nil, err
	}
	if len(res.Hits) == 0 {
		return nil, fmt.Errorf("repo not found (id: %s)", id)
	}

	blob, _ := json.Marshal(res.Hits[0])
	var full types.RepoDocumentFull
	if err := json.Unmarshal(blob, &full); err != nil {
		return nil, err
	}
	return &full, nil
}

// internal/search/meili.go

// GetFullRepoByName — fetches full details by exact name match (works 100% with public key)
func GetFullRepoByName(name string) (*types.RepoDocumentFull, error) {
	if name == "" {
		return nil, fmt.Errorf("empty name")
	}

	res, err := getMeiliClient().Index("freedevtools").Search("", &meilisearch.SearchRequest{
		Filter:               fmt.Sprintf("name = \"%s\"", name), // ← filter by name (always works)
		Limit:                1,
		AttributesToRetrieve: []string{"*"}, // get every field
	})
	if err != nil {
		return nil, fmt.Errorf("full details search failed: %w", err)
	}
	if len(res.Hits) == 0 {
		return nil, fmt.Errorf("repo not found (name: %s)", name)
	}

	blob, _ := json.Marshal(res.Hits[0])
	var full types.RepoDocumentFull
	if err := json.Unmarshal(blob, &full); err != nil {
		return nil, err
	}
	return &full, nil
}
