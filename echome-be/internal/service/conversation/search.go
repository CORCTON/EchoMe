package conversation

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"go.uber.org/zap"
)

const tavilyAPIURL = "https://api.tavily.com/search"

type TavilySearchRequest struct {
	Query         string `json:"query"`
	SearchDepth   string `json:"search_depth"`
	IncludeAnswer bool   `json:"include_answer"`
	MaxResults    int    `json:"max_results"`
}

type TavilySearchResult struct {
	URL     string  `json:"url"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
}

type TavilySearchResponse struct {
	Answer  string               `json:"answer"`
	Results []TavilySearchResult `json:"results"`
}

func performSearch(query string, apiKey string) (string, error) {
	searchReq := TavilySearchRequest{
		Query:         query,
		SearchDepth:   "basic",
		IncludeAnswer: true,
		MaxResults:    3,
	}

	reqBody, err := json.Marshal(searchReq)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", tavilyAPIURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var searchResp TavilySearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		zap.L().Error("Failed to unmarshal tavily response", zap.String("body", string(body)))
		return "", err
	}

	var searchContext string
	if searchResp.Answer != "" {
		searchContext += "Search Answer: " + searchResp.Answer + "\n\n"
	}

	for _, result := range searchResp.Results {
		searchContext += "URL: " + result.URL + "\n"
		searchContext += "Content: " + result.Content + "\n\n"
	}

	return searchContext, nil
}
