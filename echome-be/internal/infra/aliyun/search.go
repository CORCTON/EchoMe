package aliyun

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"io"
	"net/http"

	"go.uber.org/zap"
)

// PerformSearch 执行搜索操作
func (a *AliClient) PerformSearch(ctx context.Context, query string, apiKey string) (string, error) {
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

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // 关闭TLS验证
			},
		},
	}
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
