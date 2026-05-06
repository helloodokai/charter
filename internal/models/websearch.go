package models

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ddgResult struct {
	Title   string `json:"Title"`
	AbstractURL string `json:"AbstractURL"`
	Abstract string `json:"Abstract"`
	Text    string `json:"Text"`
}

type ddgResponse struct {
	RelatedTopics []ddgResult `json:"RelatedTopics"`
	Abstract     string      `json:"Abstract"`
	AbstractURL  string      `json:"AbstractURL"`
	AbstractText string      `json:"AbstractText"`
}

// WebSearch performs a DuckDuckGo search and returns summarized results.
func WebSearch(ctx context.Context, query string) (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	
	endpoint := fmt.Sprintf("https://api.duckduckgo.com/?q=%s&format=json&no_html=1&skip_disambig=1",
		url.QueryEscape(query))
	
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("creating search request: %w", err)
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("search request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("search API returned %d: %s", resp.StatusCode, string(body))
	}
	
	var ddg ddgResponse
	if err := json.NewDecoder(resp.Body).Decode(&ddg); err != nil {
		return "", fmt.Errorf("decoding search response: %w", err)
	}
	
	var b strings.Builder
	if ddg.AbstractText != "" {
		fmt.Fprintf(&b, "%s\nSource: %s\n", ddg.AbstractText, ddg.AbstractURL)
	}
	for i, topic := range ddg.RelatedTopics {
		if i >= 5 {
			break
		}
		if topic.Text != "" {
			fmt.Fprintf(&b, "- %s\n  Source: %s\n", topic.Text, topic.AbstractURL)
		}
	}
	
	if b.Len() == 0 {
		return "", nil
	}
	
	return b.String(), nil
}