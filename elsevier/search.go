package elsevier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type SearchQuery struct {
	Authors string `json:"authors"`
	Title   string `json:"title"`
	Pub     string `json:"pub"` // Title of a journal or book
}

// SearchResponse represents the top-level API response
type SearchResponse struct {
	ResultsFound int            `json:"resultsFound"`
	Results      []SearchResult `json:"results"`
	Message      string         `json:"message"`
}

// SearchResult represents a single search result item
type SearchResult struct {
	Authors         []string `json:"authors"`
	DOI             string   `json:"doi"`
	LoadDate        string   `json:"loadDate"`
	OpenAccess      bool     `json:"openAccess"`
	Pages           Pages    `json:"pages"`
	PII             string   `json:"pii"`
	PublicationDate string   `json:"publicationDate"`
	SourceTitle     string   `json:"sourceTitle"`
	Title           string   `json:"title"`
	URI             string   `json:"uri"`
	VolumeIssue     string   `json:"volumeIssue"`
}

// Pages represents page information for a result
type Pages struct {
	First string `json:"first"`
	Last  string `json:"last"`
}

func (sr *SearchResult) ToString() string {

	s := ""

	if len(sr.Authors) > 0 {
		s += strings.Join(sr.Authors, ", ") + "."
	}

	if sr.Title != "" {
		s += " " + sr.Title + "."
	}

	if sr.SourceTitle != "" {
		s += " In " + sr.SourceTitle

		if sr.VolumeIssue != "" {
			s += " (" + sr.VolumeIssue + ")"
		}
		if sr.Pages.First != "" {
			s += " " + sr.Pages.First
		}
		if sr.Pages.Last != "" {
			s += "-" + sr.Pages.Last
		}

		s += "."

	}

	return s
}

// Search searches for articles in ScienceDirect API v2
//
// query: https://dev.elsevier.com/sd_article_meta_tips.html
func (c *Client) Search(query *SearchQuery) (*SearchResponse, error) {
	endpoint := fmt.Sprintf("%s/content/search/sciencedirect", c.baseUrl)

	// Build query parameters
	queryParams := url.Values{}
	queryParams.Set("apiKey", c.apiKey)

	queryData, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("json marshal error: %v", err)
	}

	// Create request
	reqURL := fmt.Sprintf("%s?%s", endpoint, queryParams.Encode())
	req, err := http.NewRequest(http.MethodPut, reqURL, bytes.NewReader(queryData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set required headers
	req.Header.Set("X-ELS-APIKey", c.apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "github.com/sandialabs/bibcheck")

	// Execute request
	client := &http.Client{Timeout: c.timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http client error: %w", err)
	}
	defer resp.Body.Close()

	// Handle error responses
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, resp.Status)
	}

	// Parse response
	var result SearchResponse
	// s, _ := io.ReadAll(resp.Body)
	// fmt.Print(string(s))
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}
