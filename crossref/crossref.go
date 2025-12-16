// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package crossref

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sandialabs/bibcheck/config"
)

// CrossrefWork represents a work item from the Crossref API
type CrossrefWork struct {
	DOI    string   `json:"DOI"`
	Title  []string `json:"title"`
	Score  float64  `json:"score"`
	Author []struct {
		Given  string `json:"given"`
		Family string `json:"family"`
	} `json:"author"`
	Published struct {
		DateParts [][]int `json:"date-parts"`
	} `json:"published-print"`
	ContainerTitle []string `json:"container-title"`
}

// CrossrefResponse represents the API response structure
type CrossrefResponse struct {
	Status  string `json:"status"`
	Message struct {
		Items        []CrossrefWork `json:"items"`
		TotalResults int            `json:"total-results"`
	} `json:"message"`
}

// ReferenceMatchResult contains the matching results
type ReferenceMatchResult struct {
	BestMatch    *CrossrefWork
	SecondMatch  *CrossrefWork
	IsConclusive bool
	Error        error
}

func (w *CrossrefWork) ToString() string {
	s := ""
	if len(w.Author) > 0 {
		authorNames := []string{}
		for _, author := range w.Author {
			authorNames = append(authorNames, author.Given+" "+author.Family)
		}
		s += strings.Join(authorNames, ", ") + ", "
	}
	if len(w.Title) > 0 {
		s += "\"" + w.Title[0] + "\", "
	}

	if len(w.ContainerTitle) > 0 {
		s += w.ContainerTitle[0] + ", "
	}

	if len(w.Published.DateParts) > 0 {
		dp := w.Published.DateParts[0]
		dps := []string{}
		for _, x := range dp {
			dps = append(dps, fmt.Sprintf("%d", x))
		}
		s += strings.Join(dps, "-") + ". "
	}

	if w.DOI != "" {
		s += "doi:" + w.DOI + ". "
	}

	return s
}

// QueryBibliographic queries the Crossref API for reference matching
func QueryBibliographic(reference string, rows int) (*CrossrefResponse, error) {
	// Build the API URL
	baseURL := "https://api.crossref.org/v1/works"
	params := url.Values{}
	params.Add("query.bibliographic", reference)
	params.Add("rows", fmt.Sprintf("%d", rows)) // Get only 2 results to check for ties

	if config.UserEmail() != "" {
		params.Add("mailto", config.UserEmail()) // Use polite pool
	}

	fullURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create request
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("User-Agent", config.UserAgent())

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var crossrefResp CrossrefResponse
	if err := json.NewDecoder(resp.Body).Decode(&crossrefResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &crossrefResp, nil
}
