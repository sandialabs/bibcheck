// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package analyze

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cwpearson/bibliography-checker/config"
	"github.com/cwpearson/bibliography-checker/documents"
	"github.com/cwpearson/bibliography-checker/entries"
)

func GetURL(url string) ([]byte, string, error) {
	// Create client with timeout
	client := &http.Client{
		Timeout: time.Second * 10,
	}

	// Create request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("User-Agent", config.UserAgent())

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("bad status: %s", resp.Status)
	}

	// Read body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read body: %w", err)
	}

	contentType := resp.Header.Get("Content-Type")
	return body, contentType, nil
}

// returns (exists, comment, error)
func CompareURL(url, rawEntry string,
	comp entries.Comparer,
	extract documents.MetaExtractor,
) (bool, string, error) {

	content, contentType, err := GetURL(url)
	if err != nil {
		return false, "", fmt.Errorf("get URL error: %w", err)
	}

	fmt.Println("recieved", contentType, "from", url)

	// extract metadata from contents
	var meta *documents.Metadata
	if contentType == "application/pdf" {
		meta, err = extract.PDFMetadata(content)
		if err != nil {
			return false, "", fmt.Errorf("openrouter client error: %w", err)
		}
	} else {
		meta, err = extract.HTMLMetadata(content)
		if err != nil {
			return false, "", fmt.Errorf("openrouter client error: %w", err)
		}
	}

	pdfMetaFields := []string{}
	if meta.Title != "" {
		pdfMetaFields = append(pdfMetaFields, "Title: "+meta.Title)
	}
	if len(meta.Authors) > 0 {
		pdfMetaFields = append(pdfMetaFields, "Authors: "+strings.Join(meta.Authors, ", "))
	}
	if meta.ContributingOrg != "" {
		pdfMetaFields = append(pdfMetaFields, "Contributing Organization: "+meta.ContributingOrg)
	}
	metaText := strings.Join(pdfMetaFields, "\n")

	// compare metadata to entry
	ec, err := comp.Compare(metaText, rawEntry)
	if err != nil {
		return false, "", fmt.Errorf("OpenRouter client error: %w", err)
	}

	return ec.IsEquivalent, ec.Explanation, nil
}
