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

type EntryAnalysis struct {
	Exists bool // overall result

	Arxiv    Metadata
	Crossref Metadata
	DOIOrg   Metadata
	OSTI     Metadata
	URL      Search
	Web      Search
	Elsevier Metadata
}

type EntryConfig struct {
	ElsevierApiKey string
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

	// extract records

	var url string
	var urlError error
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		url, urlError = entryParser.ParseURL(text)
	}()
	wg.Wait()

	if urlError != nil {
		log.Printf("extract URL error: %v", urlError)
	} else if url != "" {
		fmt.Println("Detected URL:", url)
	}

	// check DOI if present
	doi, err := entryParser.ParseDOI(text)
	if err != nil {
		log.Printf("DOI extract error: %v", err)
		EA.DOIOrg.Error = fmt.Errorf("ParseDOI error: %w", err)
	} else if doi != "" {
		log.Println("Detected DOI", doi)
		doiExists, err := CheckDOI(doi)
		if err != nil {
			EA.DOIOrg.Error = fmt.Errorf("doi.org check error: %w", err)
		} else {
			EA.Exists = false
			EA.DOIOrg.Status = SearchStatusDone
			EA.DOIOrg.Found = doiExists
			if doiExists {
				EA.DOIOrg.Result = "doi.org entry found"
			}
			if !doiExists {
				EA.DOIOrg.Result = "no doi.org entry for " + doi
				return EA, nil
			}
		}
	}

	// Check OSTI if present
	osti, err := entryParser.ParseOSTI(text)
	if err != nil {
		log.Printf("OSTI extract error: %v", err)
		EA.OSTI.Error = fmt.Errorf("ParseOSTI error: %w", err)
	} else if osti != "" {
		fmt.Println("Detected OSTI:", osti)
		rec, err := GetOSTIRecord(osti, text)
		if err != nil {
			EA.OSTI.Error = fmt.Errorf("OSTI check error: %w", err)
		} else {
			ostiExists := rec != nil
			EA.Exists = ostiExists
			EA.OSTI.Status = SearchStatusDone
			EA.OSTI.Found = ostiExists
			if ostiExists {
				EA.OSTI.Result = rec.ToString()
			} else {
				EA.OSTI.Result = "No OSTI record for " + osti
			}

			return EA, nil
		}
	}

	// Check arXiv if present
	arxiv, err := entryParser.ParseArxiv(text)
	if err != nil {
		log.Printf("arXiv parse error: %v", err)
		EA.Arxiv.Error = err
	} else if arxiv != "" {
		fmt.Println("Detected arXiv:", arxiv)
		rec, err := GetArxivMetadata(arxiv, text)
		if err != nil {
			EA.Arxiv.Error = fmt.Errorf("arxiv check error: %w", err)
		} else {
			arxivExists := rec != nil
			EA.Exists = arxivExists
			EA.Arxiv.Status = SearchStatusDone
			EA.Arxiv.Found = arxivExists
			if arxivExists {
				EA.Arxiv.Result = rec.ToString()
			} else {
				EA.Arxiv.Result = "No arxiv paper for " + arxiv
			}
			return EA, nil
		}
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
		client := elsevier.NewClient(cfg.ElsevierApiKey)

		resp, err := client.Search(&elsevier.SearchQuery{
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

	kind, err := class.Classify(text)
	if err != nil {
		log.Printf("error classifying citation: %v", err)
		log.Println("assuming unknown kind")
		kind = entries.KindUnknown
	}
	fmt.Printf("Kind: %s\n", kind)

	// Search web
	if kind == entries.KindWebsite {

		// try direct URL access
		if url != "" {
			fmt.Println("direct URL access...")
			exists, comment, err := CompareURL(url, text, comp, extract)
			if err != nil {
				EA.SetURLSearchError(err)
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

		website, err := entryParser.ParseWebsite(text)
		if err != nil {
			EA.SetWebSearchError(err)
			return EA, nil
		}

		fmt.Println("Website:")
		fmt.Println("  URL:    ", website.URL)
		fmt.Println("  Title:  ", website.Title)
		fmt.Println("  Authors:", strings.Join(website.Authors, ", "))

		if searcher != nil {
			exists, comment, err := searcher.SearchWebsite(website)
			if err != nil {
				EA.SetWebSearchError(err)
			} else {

				EA.Exists = exists
				EA.Web.Status = SearchStatusDone
				EA.Web.Comment = comment
				EA.Web.Exists = exists
			}
		}

		return EA, nil
	} else if kind == entries.KindSoftwarePackage {

		software, err := entryParser.ParseSoftware(text)
		if err != nil {
			EA.SetWebSearchError(err)
			return EA, nil
		}

		// try direct URL access
		if software.HomepageUrl != "" {
			fmt.Println("direct URL access...")
			exists, comment, err := CompareURL(software.HomepageUrl, text, comp, extract)
			if err != nil {
				EA.SetURLSearchError(err)
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

		if searcher != nil {
			exists, comment, err := searcher.SearchSoftware(software)
			if err != nil {
				EA.SetWebSearchError(err)
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
		return EA, nil
	} else {

		result, err := CrossrefBibliography(text)
		if err != nil {
			EA.SetCrossrefSearchError(err)
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

		if searcher != nil {
			exists, comment, err := searcher.SearchEntry(text)
			if err != nil {
				EA.SetWebSearchError(err)
			} else {
				EA.Exists = exists
				EA.Web.Status = SearchStatusDone
				EA.Web.Comment = comment
				EA.Web.Exists = exists
			}
		}
		return EA, nil
	}
}

// analyze entry `id` from base-64 encoded pdf file `encoded`
func EntryFromBase64(encoded string, id int, mode string,
	comp entries.Comparer,
	class entries.Classifier,
	docExtract documents.EntryFromRawExtractor,
	docMeta documents.MetaExtractor,
	entryParser entries.Parser,
	searcher search.Searcher,
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

	return Entry(text, mode, comp, class, docMeta, entryParser, searcher)
}

// analyze entry `id` from document text `text`
func EntryFromText(text string, id int, mode string,
	comp entries.Comparer,
	class entries.Classifier,
	docExtract documents.EntryFromTextExtractor,
	docMeta documents.MetaExtractor,
	entryParser entries.Parser,
	searcher search.Searcher,
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

	return Entry(text, mode, comp, class, docMeta, entryParser, searcher)
}
