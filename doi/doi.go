// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package doi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/cwpearson/bibliography-checker/config"
)

// DOIResponse represents the JSON response from the DOI REST API
type DOIResponse struct {
	ResponseCode int     `json:"responseCode"`
	Handle       string  `json:"handle"`
	Values       []Value `json:"values,omitempty"`
	Message      string  `json:"message,omitempty"`
}

// Value represents a DOI record element
type Value struct {
	Index     int         `json:"index"`
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp string      `json:"timestamp"`
	TTL       interface{} `json:"ttl"` // Can be int or string
}

// ValueData represents the data field structure
type ValueData struct {
	Format string      `json:"format"`
	Value  interface{} `json:"value"`
}

// ResolveDOI resolves a DOI and returns all record elements
func ResolveDOI(doi string) (*DOIResponse, error) {
	return resolveDOIWithParams(doi, nil)
}

// ResolveDOIByType resolves a DOI and returns only elements of specified types
func ResolveDOIByType(doi string, types ...string) (*DOIResponse, error) {
	params := url.Values{}
	for _, t := range types {
		params.Add("type", t)
	}
	return resolveDOIWithParams(doi, params)
}

// ResolveDOIByIndex resolves a DOI and returns only elements at specified indexes
func ResolveDOIByIndex(doi string, indexes ...int) (*DOIResponse, error) {
	params := url.Values{}
	for _, idx := range indexes {
		params.Add("index", fmt.Sprintf("%d", idx))
	}
	return resolveDOIWithParams(doi, params)
}

// ResolveDOIWithOptions resolves a DOI with custom options
func ResolveDOIWithOptions(doi string, options DOIOptions) (*DOIResponse, error) {
	params := url.Values{}

	if options.Pretty {
		params.Add("pretty", "true")
	}
	if options.Auth {
		params.Add("auth", "true")
	}
	if options.Cert {
		params.Add("cert", "true")
	}
	if options.Callback != "" {
		params.Add("callback", options.Callback)
	}
	for _, t := range options.Types {
		params.Add("type", t)
	}
	for _, idx := range options.Indexes {
		params.Add("index", fmt.Sprintf("%d", idx))
	}

	return resolveDOIWithParams(doi, params)
}

// DOIOptions represents optional parameters for DOI resolution
type DOIOptions struct {
	Pretty   bool     // Pretty-print JSON output
	Auth     bool     // Authoritative query (bypass cache)
	Cert     bool     // Certified query (require authenticated response)
	Callback string   // JSONP callback function name
	Types    []string // Filter by element types
	Indexes  []int    // Filter by element indexes
}

var (
	DoesNotExistError = errors.New("doi does not exist")
)

// resolveDOIWithParams performs the actual HTTP request to resolve a DOI
func resolveDOIWithParams(doi string, params url.Values) (*DOIResponse, error) {
	// Clean the DOI (remove any leading "https://doi.org/" if present)
	doi = strings.TrimPrefix(doi, "https://doi.org/")
	doi = strings.TrimPrefix(doi, "http://doi.org/")
	doi = strings.TrimPrefix(doi, "doi.org/")

	// Build the URL
	baseURL := fmt.Sprintf("https://doi.org/api/handles/%s", doi)
	if len(params) > 0 { // len for nil maps is 0
		baseURL += "?" + params.Encode()
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create request
	req, err := http.NewRequest("GET", baseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("User-Agent", config.UserAgent())

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse JSON response
	var doiResp DOIResponse
	if err := json.Unmarshal(body, &doiResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Check response code
	switch doiResp.ResponseCode {
	case 1:
		// Success
		return &doiResp, nil
	case 2:
		return nil, fmt.Errorf("DOI resolution error: %s", doiResp.Message)
	case 100:
		return nil, DoesNotExistError
	case 200:
		return nil, fmt.Errorf("values not found for DOI: %s", doi)
	default:
		return nil, fmt.Errorf("unknown response code %d: %s", doiResp.ResponseCode, doiResp.Message)
	}
}

// GetURL extracts the URL from a DOI response (convenience function)
func GetURL(resp *DOIResponse) (string, error) {
	for _, value := range resp.Values {
		if value.Type == "URL" {
			if data, ok := value.Data.(map[string]interface{}); ok {
				if val, ok := data["value"].(string); ok {
					return val, nil
				}
			}
		}
	}
	return "", fmt.Errorf("no URL found in DOI response")
}
