// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package crossref

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/sandialabs/bibcheck/config"
	"github.com/sandialabs/bibcheck/internal/wasmhttp"
)

const baseURL = "https://api.crossref.org/v1/works"

// CrossrefWork represents a work item from the Crossref API.
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

type CrossrefResponse struct {
	Status  string `json:"status"`
	Message struct {
		Items        []CrossrefWork `json:"items"`
		TotalResults int            `json:"total-results"`
	} `json:"message"`
}

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
		dps := []string{}
		for _, x := range w.Published.DateParts[0] {
			dps = append(dps, fmt.Sprintf("%d", x))
		}
		s += strings.Join(dps, "-") + ". "
	}
	if w.DOI != "" {
		s += "doi:" + w.DOI + ". "
	}
	return s
}

// QueryBibliographic queries Crossref for reference matching.
func (c *Client) QueryBibliographic(ctx context.Context, reference string, rows int) (*CrossrefResponse, error) {
	return queryBibliographic(ctx, reference, rows, c.Do)
}

func queryBibliographic(
	ctx context.Context,
	reference string,
	rows int,
	do func(*http.Request) (*http.Response, error),
) (*CrossrefResponse, error) {
	params := url.Values{}
	params.Add("query.bibliographic", reference)
	params.Add("rows", fmt.Sprintf("%d", rows))
	if config.UserEmail() != "" {
		params.Add("mailto", config.UserEmail()) // puts us in the polite pool
	}
	requestURL := baseURL + "?" + encodeQuery(params)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", config.UserAgent())
	wasmhttp.ConfigureRequest(req)
	resp, err := do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}
	var result CrossrefResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &result, nil
}

// encodeQuery leaves the @ in Crossref's mailto parameter unescaped, matching
// the form used in Crossref's API documentation. Other parameters retain the
// standard query escaping provided by url.Values.Encode.
func encodeQuery(params url.Values) string {
	query := params.Encode()
	if email := params.Get("mailto"); email != "" {
		escaped := url.QueryEscape(email)
		mailto := strings.ReplaceAll(escaped, "%40", "@")
		query = strings.Replace(query, "mailto="+escaped, "mailto="+mailto, 1)
	}
	return query
}
