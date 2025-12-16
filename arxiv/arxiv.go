// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package arxiv

// https://info.arxiv.org/help/api/basics.html
// https://info.arxiv.org/help/api/user-manual.html

import (
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/sandialabs/bibcheck/config"
)

var ErrDoesNotExist = errors.New("no arxiv entry found")

// Client represents a client for the arXiv API
type Client struct {
	httpClient *http.Client
}

// Feed represents the Atom feed response from arXiv
type Feed struct {
	XMLName xml.Name `xml:"feed"`
	Entries []Entry  `xml:"entry"`
}

// Entry represents a single article entry in the feed
type Entry struct {
	ID              string     `xml:"id"`
	Published       string     `xml:"published"`
	Updated         string     `xml:"updated"`
	Title           string     `xml:"title"`
	Summary         string     `xml:"summary"`
	Authors         []Author   `xml:"author"`
	Links           []Link     `xml:"link"`
	Categories      []Category `xml:"category"`
	PrimaryCategory Category   `xml:"primary_category"`
	Comment         string     `xml:"comment"`
	JournalRef      string     `xml:"journal_ref"`
	DOI             string     `xml:"doi"`
}

// Author represents an article author
type Author struct {
	Name        string `xml:"name"`
	Affiliation string `xml:"affiliation"`
}

// Link represents a link to article resources
type Link struct {
	Href  string `xml:"href,attr"`
	Rel   string `xml:"rel,attr"`
	Type  string `xml:"type,attr"`
	Title string `xml:"title,attr"`
}

// Category represents an article category
type Category struct {
	Term   string `xml:"term,attr"`
	Scheme string `xml:"scheme,attr"`
}

func (e *Entry) ToString() string {
	s := ""
	if len(e.Authors) > 0 {
		authorNames := []string{}
		for _, author := range e.Authors {
			authorNames = append(authorNames, author.Name)
		}
		s += strings.Join(authorNames, ", ") + ". "
	}
	if e.Title != "" {
		s += e.Title + ". "
	}
	if e.Published != "" {
		s += "published " + e.Published + ". "
	}
	if e.Updated != "" && e.Updated != e.Published {
		s += "updated " + e.Updated + ". "
	}

	return s
}

// NewClient creates a new arXiv API client with proper identification
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetByID retrieves metadata for a specific arXiv ID
func (c *Client) GetByID(arxivID string) (*Entry, error) {
	// Extract just the ID part if full URL is provided
	id := extractArxivID(arxivID)

	// Construct API URL using id_list parameter
	apiURL := fmt.Sprintf("http://export.arxiv.org/api/query?id_list=%s", id)

	// Make the request with proper headers
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Set User-Agent header for politeness
	req.Header.Set("User-Agent", config.UserAgent())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Parse the XML response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	var feed Feed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("parsing XML: %w", err)
	}

	if len(feed.Entries) == 0 {
		return nil, ErrDoesNotExist
	}

	return &feed.Entries[0], nil
}

// extractArxivID extracts the ID from various input formats
func extractArxivID(input string) string {
	// Remove common URL prefixes
	input = strings.TrimPrefix(input, "https://arxiv.org/abs/")
	input = strings.TrimPrefix(input, "http://arxiv.org/abs/")
	input = strings.TrimPrefix(input, "arxiv.org/abs/")
	return input
}
