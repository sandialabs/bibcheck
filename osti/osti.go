// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package osti

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/cwpearson/bibliography-checker/config"
)

const (
	baseURL        = "https://www.osti.gov/api/v1"
	defaultTimeout = 30 * time.Second
)

// Client represents an OSTI API client
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// Record represents an OSTI record
// https://www.osti.gov/api/v1/docs#general-guidelines-responseformats
type Record struct {
	OstiID          string   `json:"osti_id"`
	Title           string   `json:"title"`
	Authors         []string `json:"authors"`
	PublicationDate string   `json:"publication_date"`
	ConferenceInfo  string   `json:"conference_info"`
	DOI             string   `json:"doi"`
	ResearchOrg     string   `json:"research_org"`
	SponsorOrg      string   `json:"sponsor_org"`
	ContributorOrg  string   `json:"contributor_org"`
	// Add more fields as needed based on actual API response
}

func (r *Record) ToString() string {
	s := ""
	if len(r.Authors) > 0 {
		s += strings.Join(r.Authors, ", ") + ". "
	}
	if r.Title != "" {
		s += r.Title + ". "
	}
	if r.ConferenceInfo != "" {
		s += "in " + r.ConferenceInfo + ". "
	}
	if r.PublicationDate != "" {
		s += "published " + r.PublicationDate + ". "
	}
	if r.DOI != "" {
		s += "doi:" + r.DOI + ". "
	}

	return s
}

var (
	ErrDoesNotExist = errors.New("osti record does not exist")
)

// RecordsResponse represents the response when fetching multiple records
type RecordsResponse struct {
	Records []Record `json:"records"`
	Total   int      `json:"total"`
	Page    int      `json:"page"`
	PerPage int      `json:"per_page"`
}

// NewClient creates a new OSTI API client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		baseURL: baseURL,
	}
}

// NewClientWithTimeout creates a new OSTI API client with custom timeout
func NewClientWithTimeout(timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		baseURL: baseURL,
	}
}

// GetRecord retrieves a specific OSTI record by ID
func (c *Client) GetRecord(ostiID string) (*Record, error) {

	fmt.Println("check OSTI record", ostiID)
	endpoint := fmt.Sprintf("%s/records/%s", c.baseURL, ostiID)

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Set Accept header for JSON response
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", config.UserAgent())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrDoesNotExist
	} else if resp.StatusCode != http.StatusOK {
		// body, _ := io.ReadAll(resp.Body)
		// return nil, fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("API error: status %d", resp.StatusCode)
	}

	// s, _ := io.ReadAll(resp.Body)
	// log.Println(string(s))

	var record []Record
	if err := json.NewDecoder(resp.Body).Decode(&record); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &record[0], nil
}

// ListRecordsOptions represents options for listing records
type ListRecordsOptions struct {
	Page    int
	PerPage int
	Query   string
	// Add more filter options as needed
}

// ListRecords retrieves a list of OSTI records
func (c *Client) ListRecords(opts *ListRecordsOptions) (*RecordsResponse, error) {
	endpoint := fmt.Sprintf("%s/records", c.baseURL)

	// Build query parameters
	params := url.Values{}
	if opts != nil {
		if opts.Page > 0 {
			params.Set("page", strconv.Itoa(opts.Page))
		}
		if opts.PerPage > 0 {
			params.Set("per_page", strconv.Itoa(opts.PerPage))
		}
		if opts.Query != "" {
			params.Set("q", opts.Query)
		}
	}

	if len(params) > 0 {
		endpoint = fmt.Sprintf("%s?%s", endpoint, params.Encode())
	}

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var recordsResp RecordsResponse
	if err := json.NewDecoder(resp.Body).Decode(&recordsResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &recordsResp, nil
}

// SearchRecords performs a search query and returns matching records
func (c *Client) SearchRecords(query string) (*RecordsResponse, error) {
	return c.ListRecords(&ListRecordsOptions{
		Query: query,
	})
}

// GetRecordsByPage retrieves records with pagination
func (c *Client) GetRecordsByPage(page, perPage int) (*RecordsResponse, error) {
	return c.ListRecords(&ListRecordsOptions{
		Page:    page,
		PerPage: perPage,
	})
}
