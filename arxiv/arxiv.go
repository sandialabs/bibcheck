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
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/sandialabs/bibcheck/config"
	"github.com/sandialabs/bibcheck/internal/ratelimit"
	"github.com/sandialabs/bibcheck/internal/wasmhttp"
)

const (
	defaultTimeout    = 30 * time.Second
	requestsPerSecond = 1.0 / 3.0
	burstSize         = 1
	maxConcurrent     = 1
)

var ErrDoesNotExist = errors.New("no arxiv entry found")

// Client is a rate-limited client for the arXiv API. A Client is safe for
// concurrent use and should be shared by all work in one process.
type Client struct {
	httpClient *http.Client
	delay      func(time.Duration)
	limiter    *ratelimit.Client
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient replaces the HTTP client used for upstream requests.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) { c.httpClient = client }
}

// WithDelayCallback registers a callback for requests delayed by rate limiting.
func WithDelayCallback(callback func(time.Duration)) Option {
	return func(c *Client) { c.delay = callback }
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

// NewClient creates an arXiv API client limited to one request start every
// three seconds and one concurrent upstream request.
func NewClient(options ...Option) *Client {
	c := &Client{
		httpClient: &http.Client{Timeout: defaultTimeout},
		delay: func(delay time.Duration) {
			log.Printf("arXiv request delayed by rate limit: %s", delay)
		},
	}
	for _, option := range options {
		option(c)
	}
	c.limiter = ratelimit.NewClient(c.httpClient, ratelimit.Policy{
		RequestsPerSecond: requestsPerSecond,
		Burst:             burstSize,
		MaxConcurrent:     maxConcurrent,
	}, c.delay)
	return c
}

// Do performs a rate-limited HTTP request. WASM requests rely on the shared
// fetch proxy's limiter instead of applying a second browser-local limit.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if wasmhttp.UsesFetchProxy() {
		return c.httpClient.Do(req)
	}
	return c.limiter.Do(req)
}

// GetByID retrieves metadata for a specific arXiv ID
func (c *Client) GetByID(arxivID string) (*Entry, error) {
	// Extract just the ID part if full URL is provided
	id := extractArxivID(arxivID)

	// Construct API URL using id_list parameter
	apiURL := fmt.Sprintf("http://export.arxiv.org/api/query?id_list=%s", id)

	// Make the request with proper headers
	// Route this through the proxy when it's in wasm (mixed-content / CORS restrictions)
	req, err := http.NewRequest("GET", wasmhttp.FetchURL(apiURL), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Set User-Agent header for politeness
	req.Header.Set("User-Agent", config.UserAgent())
	wasmhttp.ConfigureRequest(req)

	resp, err := c.Do(req)
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
