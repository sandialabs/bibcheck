// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package analyze

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/sandialabs/bibcheck/documents"
	"github.com/sandialabs/bibcheck/entries"
	"github.com/sandialabs/bibcheck/search"
)

const (
	SearchStatusNotAttempted string = ""
	SearchStatusDone         string = "done"
)

type Search struct {
	Status  string
	Exists  bool
	Comment string
	Error   error
}

type Metadata struct {
	Status string
	Found  bool
	Result string
	Error  error
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

type EntryAnalysis struct {
	Exists bool // overall result

	Arxiv    ArxivResult
	Crossref Metadata
	DOIOrg   DOIOrgResult
	OSTI     OSTIResult
	URL      Search
	Web      Search
	Elsevier Metadata
}

type EntryConfig struct {
	ElsevierClient *elsevier.Client
}

// analyze bib entry `text`
func Entry(text string, mode string,
	comp entries.Comparer,
	class entries.Classifier,
	extract documents.MetaExtractor,
	entryParser entries.Parser,
	searcher search.Searcher,
	cfg *EntryConfig,
) (*EntryAnalysis, error) {

	EA := &EntryAnalysis{}

	if mode == "crossref" {
		result, err := CrossrefBibliography(text)
		if err != nil {
			EA.Crossref.Error = err
		} else {
			crossrefExists := (result.Entry != nil)
			EA.Exists = crossrefExists

			EA.Crossref.Status = SearchStatusDone
			EA.Crossref.Found = crossrefExists

			if crossrefExists {
				EA.Crossref.Result = result.Entry.ToString()
			} else {
				EA.Crossref.Result = result.Comment
			}
		}
		return EA, nil
	}

	// check DOI if present
	// The existence or not of the DOI is not very useful alone, so continue on
	if doi, err := entryParser.ParseDOI(text); err != nil {
		EA.DOIOrg.Error = fmt.Errorf("ParseDOI error: %w", err)
	} else if doi == "" {
		EA.DOIOrg.Error = fmt.Errorf("ParseDOI extracted empty doi")
	} else {
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
	if osti, err := entryParser.ParseOSTI(text); err != nil {
		log.Printf("OSTI extract error: %v", err)
		EA.OSTI.Error = fmt.Errorf("ParseOSTI error: %w", err)
	} else if osti == "" {
		EA.DOIOrg.Error = fmt.Errorf("ParseOSTI extracted empty OSTI ID")
	} else {
		fmt.Println("Detected OSTI", osti)
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
	if id, err := entryParser.ParseArxiv(text); err != nil {
		log.Printf("ParseArxiv error: %v", err)
		EA.Arxiv.Error = fmt.Errorf("ParseArxiv error: %w", err)
	} else if id == "" {
		EA.Arxiv.Error = fmt.Errorf("ParseArxiv extracted empty Arxiv ID")
	} else if id != "" {
		fmt.Println("Detected arXiv", id)
		if entry, err := GetArxivMetadata(id, text); err != nil {
			EA.Arxiv.Error = fmt.Errorf("arxiv check error: %w", err)
		} else {
			EA.Arxiv.Entry = entry
			EA.Arxiv.ID = id
			EA.Arxiv.Status = SearchStatusDone
			return EA, nil
		}
	}

	// extract records
	var wg sync.WaitGroup

	url, urlError := entryParser.ParseURL(text)
	if urlError != nil {
		log.Printf("extract URL error: %v", urlError)
	} else if url != "" {
		fmt.Println("Detected URL:", url)
	}

	log.Println("Extracting metadata for Elsevier search...")
	var authorsErr, titleErr, pubErr error
	var authors *entries.Authors
	var title, pub string
	wg.Add(3)
	go func() {
		defer wg.Done()
		authors, authorsErr = entryParser.ParseAuthors(text)
	}()
	go func() {
		defer wg.Done()
		title, titleErr = entryParser.ParseTitle(text)
	}()
	go func() {
		defer wg.Done()
		pub, pubErr = entryParser.ParsePub(text)
	}()
	wg.Wait()

	// Elsevier search
	if cfg != nil && cfg.ElsevierClient != nil {

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
				Title:   "title",
				Authors: strings.Join(authors.Authors, " AND "),
				Pub:     pub,
			})

			if err != nil {
				log.Printf("elsevier.Search error: %v", err)
				EA.Elsevier.Error = fmt.Errorf("elsevier.Search error: %w", err)
			} else if len(resp.Results) < 1 {
				log.Printf("elsevier.Search returned no results")
				EA.Elsevier.Found = false
				EA.Elsevier.Result = "no matching results from Elsevier"
			} else {
				best := resp.Results[0]
				EA.Elsevier.Found = true
				EA.Elsevier.Result = best.ToString()
			}

		} else {
			log.Println("unable to parse sufficient metadata for Elsevier search")
			EA.Elsevier.Result = "unable to parse sufficient metadata for Elsevier search"
		}
	}

	website, err := entryParser.ParseWebsite(text)
	if err != nil {
		msg := fmt.Sprintf("ParseWebsite error: %v", err)
		log.Println(msg)
	} else {
		fmt.Println("Website:")
		fmt.Println("  URL:    ", website.URL)
		fmt.Println("  Title:  ", website.Title)
		fmt.Println("  Authors:", strings.Join(website.Authors, ", "))
	}

	// try direct URL access
	if website != nil && website.URL != "" {
		fmt.Println("direct URL access...")
		exists, comment, err := CompareURL(url, text, comp, extract)
		if err != nil {
			EA.URL.Error = err
			return EA, nil
		}
		EA.Exists = exists
		EA.URL.Comment = comment
		EA.URL.Exists = exists
		EA.URL.Status = SearchStatusDone

		if EA.Exists {
			return EA, nil
		}
	}

	// try web search
	if website != nil && searcher != nil {
		exists, comment, err := searcher.SearchWebsite(website)
		if err != nil {
			EA.Web.Error = err
		} else {
			EA.Exists = exists
			EA.Web.Status = SearchStatusDone
			EA.Web.Comment = comment
			EA.Web.Exists = exists
		}
	}

	software, err := entryParser.ParseSoftware(text)
	if err != nil {
		log.Printf("ParseSoftware error: %v", err)
	}

	// try direct URL access
	if software != nil && software.HomepageUrl != "" {
		fmt.Println("direct URL access...")
		exists, comment, err := CompareURL(software.HomepageUrl, text, comp, extract)
		if err != nil {
			EA.URL.Error = err
			return EA, nil
		}
		EA.Exists = exists
		EA.URL.Comment = comment
		EA.URL.Exists = exists
		EA.URL.Status = SearchStatusDone

		if EA.Exists {
			return EA, nil
		}
	}

	if software != nil && searcher != nil {
		exists, comment, err := searcher.SearchSoftware(software)
		if err != nil {
			EA.Web.Error = err
		} else {
			EA.Exists = exists
			EA.Web.Status = SearchStatusDone
			EA.Web.Comment = comment
			EA.Web.Exists = exists
		}
	} else {
		EA.Web.Status = SearchStatusNotAttempted
		EA.Web.Comment = "Search capability not available"
	}

	result, err := CrossrefBibliography(text)
	if err != nil {
		EA.Crossref.Error = err
	} else {
		crossrefExists := result.Entry != nil
		EA.Exists = crossrefExists
		EA.Crossref.Status = SearchStatusDone
		EA.Crossref.Found = crossrefExists

		if crossrefExists {
			EA.Crossref.Result = result.Entry.ToString()
		} else {
			EA.Crossref.Result = result.Comment
		}

		if crossrefExists {
			return EA, nil
		}
	}

	if result != nil && searcher != nil {
		exists, comment, err := searcher.SearchEntry(text)
		if err != nil {
			EA.Web.Error = err
		} else {
			EA.Exists = exists
			EA.Web.Status = SearchStatusDone
			EA.Web.Comment = comment
			EA.Web.Exists = exists
		}
	}
	return EA, nil
}

// analyze entry `id` from base-64 encoded pdf file `encoded`
func EntryFromBase64(encoded string, id int, mode string,
	comp entries.Comparer,
	class entries.Classifier,
	docExtract documents.EntryFromRawExtractor,
	docMeta documents.MetaExtractor,
	entryParser entries.Parser,
	searcher search.Searcher,
	cfg *EntryConfig,
) (*EntryAnalysis, error) {

	if mode == "" {
		mode = "auto"
	}

	fmt.Printf("=== Entry %d ===\n", id)

	// Extract citation text
	text, err := docExtract.EntryFromRaw(encoded, id)
	if err != nil {
		return nil, fmt.Errorf("error extracting citation %d: %w", id, err)
	}
	fmt.Println(text)

	return Entry(text, mode, comp, class, docMeta, entryParser, searcher, cfg)
}

// analyze entry `id` from document text `text`
func EntryFromText(text string, id int, mode string,
	comp entries.Comparer,
	class entries.Classifier,
	docExtract documents.EntryFromTextExtractor,
	docMeta documents.MetaExtractor,
	entryParser entries.Parser,
	searcher search.Searcher,
	cfg *EntryConfig,
) (*EntryAnalysis, error) {

	if mode == "" {
		mode = "auto"
	}

	fmt.Printf("=== Entry %d ===\n", id)

	// Extract citation text
	text, err := docExtract.EntryFromText(text, id)
	if err != nil {
		return nil, fmt.Errorf("error extracting citation %d: %w", id, err)
	}
	fmt.Println(text)

	return Entry(text, mode, comp, class, docMeta, entryParser, searcher, cfg)
}
