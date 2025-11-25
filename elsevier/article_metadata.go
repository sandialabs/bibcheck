package elsevier

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// ArticleMetadataParams contains optional parameters for the article metadata search
type ArticleMetadataParams struct {
	View             string // STANDARD or COMPLETE (default: STANDARD)
	Field            string // Comma-separated list of specific fields to return
	Start            int    // Results offset (starting position)
	Count            int    // Maximum number of results to return
	SuppressNavLinks bool   // Suppress top-level navigation links
}

// ArticleMetadataResponse represents the API response structure
type ArticleMetadataResponse struct {
	SearchResults struct {
		TotalResults string                 `json:"opensearch:totalResults"`
		StartIndex   string                 `json:"opensearch:startIndex"`
		ItemsPerPage string                 `json:"opensearch:itemsPerPage"`
		Query        map[string]interface{} `json:"opensearch:Query"`
		Link         []Link                 `json:"link"`
		Entry        []ArticleEntry         `json:"entry"`
	} `json:"search-results"`
}

// Link represents a link in the response
type Link struct {
	Ref  string `json:"@ref"`
	Href string `json:"@href"`
	Type string `json:"@type,omitempty"`
}

// ArticleEntry represents a single article in the search results
type ArticleEntry struct {
	Identifier      string              `json:"dc:identifier"`
	Title           string              `json:"dc:title"`
	Creator         []string            `json:"dc:creator"`
	PublicationName string              `json:"prism:publicationName"`
	CoverDate       string              `json:"prism:coverDate"`
	DOI             string              `json:"prism:doi"`
	PII             string              `json:"pii"`
	OpenAccess      bool                `json:"openaccess"`
	Link            []Link              `json:"link"`
	Authors         []map[string]string `json:"authors,omitempty"`
}

// https://dev.elsevier.com/sd_article_meta_tips.html
type Query struct {
	Authors []string
	Title   string
}

// returns a string suitable to provide to the SearchArticleMetadata function
func (q *Query) toString() string {

	s := ""

	if len(q.Authors) > 0 {
		s += "aut(" + strings.Join(q.Authors, " AND ") + ")"
	}

	if q.Title != "" {
		s += "ttl(" + q.Title + ")"
	}

	return s
}

// ArticleMetadataRaw searches performs an Article Metadata query in ScienceDirect
//
// query: https://dev.elsevier.com/sd_article_meta_tips.html
func (c *Client) ArticleMetadataRaw(query string, params *ArticleMetadataParams) (*ArticleMetadataResponse, error) {
	endpoint := fmt.Sprintf("%s/content/metadata/article", c.baseUrl)

	// Build query parameters
	queryParams := url.Values{}
	queryParams.Set("query", query)
	queryParams.Set("apiKey", c.apiKey)

	if params != nil {
		if params.View != "" {
			queryParams.Set("view", params.View)
		}
		if params.Field != "" {
			queryParams.Set("field", params.Field)
		}
		if params.Start > 0 {
			queryParams.Set("start", strconv.Itoa(params.Start))
		}
		if params.Count > 0 {
			queryParams.Set("count", strconv.Itoa(params.Count))
		}
		if params.SuppressNavLinks {
			queryParams.Set("suppressNavLinks", "true")
		}
	}

	// Create request
	reqURL := fmt.Sprintf("%s?%s", endpoint, queryParams.Encode())
	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
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
	var result ArticleMetadataResponse
	// s, _ := io.ReadAll(resp.Body)
	// fmt.Print(string(s))
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// ArticleMetadata searches for articles in ScienceDirect
//
// query: https://dev.elsevier.com/sd_article_meta_tips.html
func (c *Client) ArticleMetadata(query *Query, params *ArticleMetadataParams) (*ArticleMetadataResponse, error) {
	return c.ArticleMetadataRaw(query.toString(), params)
}
