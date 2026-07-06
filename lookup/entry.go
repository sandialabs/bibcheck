// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package lookup

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/sandialabs/bibcheck/arxiv"
	"github.com/sandialabs/bibcheck/crossref"
	"github.com/sandialabs/bibcheck/documents"
	"github.com/sandialabs/bibcheck/elsevier"
	"github.com/sandialabs/bibcheck/entries"
	"github.com/sandialabs/bibcheck/internal/wasmhttp"
	"github.com/sandialabs/bibcheck/osti"
)

const (
	SearchStatusNotAttempted string = ""
	SearchStatusDone         string = "done"
	retrieveTimeout                 = 20 * time.Second
	maxFetchErrorBytes       int64  = 4 * 1024
)

type Search struct {
	Status  string
	Exists  bool
	Comment string
	Error   error
}

type DOIOrgResult struct {
	Status string
	ID     string
	Found  bool
	Error  error
}

type OSTIResult struct {
	Status string
	ID     string
	Record *osti.Record
	Error  error
}

type ArxivResult struct {
	Status string
	ID     string
	Entry  *arxiv.Entry
	Error  error
}

type ElsevierResult struct {
	Status string
	Result *elsevier.SearchResult
	Error  error
}

type CrossrefResult struct {
	Status  string
	Work    *crossref.CrossrefWork
	Comment string
	Error   error
}

type OnlineResult struct {
	Status   string
	Metadata *documents.Metadata
	Error    error
}

type SummarizeResult struct {
	Status  string
	Matches bool
	Comment string
	Error   error
}

type Result struct {
	Text string

	Arxiv    ArxivResult
	Crossref CrossrefResult
	DOIOrg   DOIOrgResult
	Elsevier ElsevierResult
	OSTI     OSTIResult
	Online   OnlineResult
	Web      Search

	Summary SummarizeResult
}

type EntryConfig struct {
	ElsevierClient *elsevier.Client
	CrossrefClient *crossref.Client
}

func retrieveUrl(url string) ([]byte, string, error) {
	client := &http.Client{
		Timeout: retrieveTimeout,
	}
	fetchURL := wasmhttp.FetchURL(url)
	log.Println("GET", url)
	req, err := http.NewRequest(http.MethodGet, fetchURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("http.NewRequest error: %w", err)
	}
	wasmhttp.ConfigureRequest(req)

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("http.Client.Do error: %w", err)
	}
	defer resp.Body.Close()

	if err := fetchResponseError(resp, wasmhttp.UsesFetchProxy()); err != nil {
		return nil, "", err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read body error: %w", err)
	}

	// Try to get content type from header first
	contentType := resp.Header.Get("Content-Type")

	// If header is missing or generic, detect from content
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = http.DetectContentType(body)
	}

	return body, contentType, nil
}

func fetchResponseError(resp *http.Response, usingProxy bool) error {
	if usingProxy {
		switch resp.Header.Get(wasmhttp.FetchResultHeader) {
		case wasmhttp.FetchResultProxyError:
			detail, err := io.ReadAll(io.LimitReader(resp.Body, maxFetchErrorBytes+1))
			if err != nil {
				return fmt.Errorf("/api/fetch returned %s and its error body could not be read: %w", resp.Status, err)
			}
			truncated := int64(len(detail)) > maxFetchErrorBytes
			if truncated {
				detail = detail[:maxFetchErrorBytes]
			}
			message := strings.TrimSpace(string(detail))
			if message == "" {
				message = resp.Status
			} else if truncated {
				message += "..."
			}
			return fmt.Errorf("/api/fetch error: %s", message)
		case wasmhttp.FetchResultUpstream:
			// The status and body came from the requested URL.
		default:
			return fmt.Errorf("/api/fetch returned %s without a valid %s header; the endpoint may be unavailable or misconfigured",
				resp.Status, wasmhttp.FetchResultHeader)
		}
	}

	if resp.StatusCode >= 400 {
		if usingProxy {
			return fmt.Errorf("upstream HTTP error: %s", resp.Status)
		}
		return fmt.Errorf("HTTP error: %s", resp.Status)
	}
	return nil
}

// analyze bib entry `text`
func Entry(text string, mode string,
	class entries.Classifier,
	extract documents.MetaExtractor,
	entryParser entries.Parser,
	cfg *EntryConfig,
) (*Result, error) {

	EA := &Result{
		Text: text,
	}

	// check DOI if present
	// The existence or not of the DOI is not very useful alone, so continue on
	if doi := entries.ExtractDOI(text); doi != "" {
		log.Println("Detected DOI", doi)
		EA.DOIOrg.ID = doi
		if found, err := CheckDOI(doi); err != nil {
			EA.DOIOrg.Error = fmt.Errorf("CheckDOI error: %w", err)
		} else {
			log.Println("DOI found:", found)
			EA.DOIOrg.Found = found
			EA.DOIOrg.Status = SearchStatusDone
		}
	}

	// Check OSTI if present
	// Finding the ID should provide enough info to evaluate the entry
	if osti := entries.ExtractOSTI(text); osti != "" {
		log.Printf("Detected OSTI %s", osti)
		EA.OSTI.ID = osti
		if rec, err := GetOSTIRecord(osti, text); err != nil {
			EA.OSTI.Error = fmt.Errorf("GetOSTIRecord error: %w", err)
		} else {
			EA.OSTI.Record = rec
			EA.OSTI.Status = SearchStatusDone
			return EA, nil
		}
	}

	// Check arXiv if present
	// Finding the ID should provide enough info to evaluate the entry
	if id := entries.ExtractArxiv(text); id != "" {
		log.Printf("Detected arXiv %s", id)
		if entry, err := GetArxivMetadata(id, text); err != nil {
			EA.Arxiv.Error = fmt.Errorf("arxiv check error: %w", err)
		} else {
			EA.Arxiv.Entry = entry
			EA.Arxiv.ID = id
			EA.Arxiv.Status = SearchStatusDone
			return EA, nil
		}
	}

	// if we got here, this wasn't OSTI or arxiv
	// pursue some general lookup strategies:
	// * elsevier
	// * crossref

	// Elsevier search
	if cfg != nil && cfg.ElsevierClient != nil {

		log.Println("Extracting metadata for Elsevier search...")
		var wg sync.WaitGroup
		var authorsErr, titleErr, pubErr error
		var authors *entries.Authors
		var title, pub string
		wg.Add(3)
		go func() {
			defer wg.Done()
			authors, authorsErr = entryParser.ParseAuthors(text)
			log.Printf("authors: %v", authors.Authors)
		}()
		go func() {
			defer wg.Done()
			title, titleErr = entryParser.ParseTitle(text)
			log.Printf("title: %v", title)
		}()
		go func() {
			defer wg.Done()
			pub, pubErr = entryParser.ParsePub(text)
			log.Printf("pub: %v", pub)
		}()
		wg.Wait()

		if authorsErr != nil {
			log.Printf("ParseAuthors error: %v\n", authorsErr)
			EA.Elsevier.Error = fmt.Errorf("ParseAuthors error: %w", authorsErr)
		} else if titleErr != nil {
			log.Printf("ParseTitle error: %v\n", titleErr)
			EA.Elsevier.Error = fmt.Errorf("ParseTitle error: %w", titleErr)
		} else if pubErr != nil {
			log.Printf("ParsePub error: %v\n", pubErr)
			EA.Elsevier.Error = fmt.Errorf("ParsePub error: %w", pubErr)
		} else if len(authors.Authors) > 0 && title != "" && pub != "" {
			resp, err := cfg.ElsevierClient.Search(&elsevier.SearchQuery{
				Title:   title,
				Authors: strings.Join(authors.Authors, " AND "),
				Pub:     pub,
			})

			if err != nil {
				log.Printf("elsevier.Search error: %v", err)
				EA.Elsevier.Error = fmt.Errorf("elsevier.Search error: %w", err)
			} else if len(resp.Results) < 1 {
				EA.Elsevier.Status = SearchStatusDone
				log.Printf("elsevier.Search returned no results")
			} else {
				EA.Elsevier.Status = SearchStatusDone
				best := resp.Results[0]
				EA.Elsevier.Result = best
			}
		} else {
			log.Println("unable to parse sufficient metadata for Elsevier search")
			EA.Elsevier.Error = fmt.Errorf("unable to parse sufficient metadata for elsevier search")
		}
	}

	// crossref search
	var crossrefClient *crossref.Client
	if cfg != nil {
		crossrefClient = cfg.CrossrefClient
	}
	if work, comment, err := crossrefQueryBibliographic(crossrefClient, text); err != nil {
		EA.Crossref.Error = err
	} else {
		if work == nil {
			log.Printf("crossref.org query returned no record: %s", comment)
		}
		EA.Crossref.Work = work
		EA.Crossref.Status = SearchStatusDone
		EA.Crossref.Comment = comment
	}

	// if we have an elsevier or crossref result we're satisfied
	if EA.Crossref.Work != nil || EA.Elsevier.Result != nil {
		return EA, nil
	}

	// otherwise, let's try to treat this as a generic online resource
	if online, err := entryParser.ParseOnline(text); err != nil {
		EA.Online.Error = fmt.Errorf("ParseOnline error: %v", err)
	} else if parsedURL, err := url.Parse(online.URL); err != nil {
		EA.Online.Error = fmt.Errorf("ParseOnline provided a URL that did not parse: %v", err)
	} else if online.URL != "" {
		log.Printf("Online:\n  URL:     %s\n  Title:   %s\n  Authors: %s",
			online.URL,
			online.Title,
			strings.Join(online.Authors, ", "),
		)

		// TODO: we can somehow do format=markdown for github, which might produce better results

		if body, contentType, err := retrieveUrl(online.URL); err != nil {
			log.Printf("retrieve url error: %s", err)
			EA.Online.Error = fmt.Errorf("retrieve url error: %w", err)
		} else {
			log.Println("retrieved URL content type:", contentType)
			contentTypeLower := strings.ToLower(contentType)

			if strings.HasSuffix(strings.ToLower(parsedURL.Path), ".pdf") &&
				!strings.Contains(contentTypeLower, "application/pdf") {
				EA.Online.Error = fmt.Errorf("URL ending in .pdf returned non-PDF content type: %s", contentType)
			} else if strings.Contains(contentTypeLower, "application/pdf") {
				if meta, err := extract.PDFMetadata(body); err != nil {
					EA.Online.Error = fmt.Errorf("extract.PDFMetadata error: %w", err)
				} else {
					EA.Online.Metadata = meta
					EA.Online.Status = SearchStatusDone
				}
			} else if strings.Contains(contentTypeLower, "text/html") {
				if meta, err := extract.HTMLMetadata(body); err != nil {
					EA.Online.Error = fmt.Errorf("extract.HTMLMetadata error: %w", err)
				} else {
					EA.Online.Metadata = meta
					EA.Online.Status = SearchStatusDone
				}

			} else {
				EA.Online.Error = fmt.Errorf("unexpected content type: %s", contentType)
			}

		}
	}

	return EA, nil
}

// analyze entry `id` from base-64 encoded pdf file `encoded`
func EntryFromBase64(encoded string, id int, mode string,
	class entries.Classifier,
	docExtract documents.EntryFromRawExtractor,
	docMeta documents.MetaExtractor,
	entryParser entries.Parser,
	cfg *EntryConfig,
) (*Result, error) {

	if mode == "" {
		mode = "auto"
	}

	// Extract citation text
	text, err := docExtract.EntryFromRaw(encoded, id)
	if err != nil {
		return nil, fmt.Errorf("error extracting citation %d: %w", id, err)
	}
	log.Printf("=== Entry %d ===", id)
	log.Print(text)

	return Entry(text, mode, class, docMeta, entryParser, cfg)
}

// analyze entry `id` from a prepared bibliography artifact.
func EntryFromBibliography(b *documents.Bibliography, id int, mode string,
	class entries.Classifier,
	docExtract documents.EntryFromBibliographyExtractor,
	docMeta documents.MetaExtractor,
	entryParser entries.Parser,
	cfg *EntryConfig,
) (*Result, error) {

	if mode == "" {
		mode = "auto"
	}

	// Extract citation text
	text, err := docExtract.EntryFromBibliography(b, id)
	if err != nil {
		return nil, fmt.Errorf("error extracting citation %d: %w", id, err)
	}
	log.Printf("=== Entry %d ===", id)
	log.Print(text)

	return Entry(text, mode, class, docMeta, entryParser, cfg)
}
