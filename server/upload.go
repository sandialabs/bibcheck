// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package server

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/labstack/echo/v4"
	"github.com/sandialabs/bibcheck/analyze"
	"github.com/sandialabs/bibcheck/shirty"
)

func handleUploadGet(c echo.Context) error {
	tmpl := template.Must(template.ParseFiles("server/templates/upload.html"))
	return tmpl.Execute(c.Response(), nil)
}

func generateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

type BibEntry struct {
	ID string `json:"id"`

	TextStatus string `json:"text_status"` // "pending", "active", "completed", "error"
	Text       string `json:"text"`

	AnalysisStatus string `json:"analysis_status"` // "pending", "active", "completed", "error"
	AnalysisFound  string `json:"analysis_found"`  // "found" "not-found"
	Analysis       string `json:"analysis"`
}

type Document struct {
	ID       string
	Filename string
	Entries  []*BibEntry
	mu       sync.RWMutex
}

var documents = make(map[string]*Document)
var docMu sync.RWMutex

func doAnalyze(shirtyApiKey string, doc *Document) {

	client := shirty.NewWorkflow(shirtyApiKey)

	textractResp, err := client.Textract(filepath.Join("uploads", doc.Filename))
	if err != nil {
		log.Fatalf("textract error: %v", err)
	}
	numEntries, err := client.NumBibEntries(textractResp.Text)
	if err != nil {
		log.Fatalf("bibliography size error: %v\n", err)
		return
	}
	log.Println(numEntries)

	// pre-populate empty entries
	doc.mu.Lock()
	for i := range numEntries {
		doc.Entries = append(doc.Entries, &BibEntry{
			ID:             fmt.Sprintf("%d", i+1),
			TextStatus:     "pending",
			AnalysisStatus: "pending",
		})
	}
	doc.mu.Unlock()

	// extract bib entries
	for i, ent := range doc.Entries {

		doc.mu.Lock()
		ent.TextStatus = "active"
		doc.mu.Unlock()

		text, err := client.EntryFromText(textractResp.Text, i+1)
		if err != nil {
			log.Printf("error extracting citation %d: %v", i+1, err)
			ent.TextStatus = "error"
			continue
		}

		doc.mu.Lock()
		ent.TextStatus = "completed"
		ent.Text = text
		doc.mu.Unlock()
	}

	// analyze each entry
	for _, ent := range doc.Entries {
		doc.mu.Lock()
		ent.AnalysisStatus = "active"
		doc.mu.Unlock()

		ea, err := analyze.Entry(ent.Text, "auto", client, client, client, nil)
		doc.mu.Lock()
		if err != nil {
			ent.AnalysisStatus = "error"
		} else {
			ent.AnalysisStatus = "completed"
			// if ea.Exists {
			// 	ent.AnalysisFound = "found"
			// } else {
			// 	ent.AnalysisFound = "not-found"
			// }

			if ea.Arxiv.Status == analyze.SearchStatusDone && ea.Arxiv.Entry != nil {
				ent.Analysis += fmt.Sprintf("arxiv: %s\n", ea.Arxiv.Entry.ToString())
			}
			if ea.OSTI.Status == analyze.SearchStatusDone && ea.OSTI.Record != nil {
				ent.Analysis += fmt.Sprintf("OSTI: %s\n", ea.OSTI.Record.ToString())
			}
			if ea.Crossref.Status == analyze.SearchStatusDone && ea.Crossref.Work != nil {
				ent.Analysis += fmt.Sprintf("crossref: %s\n", ea.Crossref.Work.ToString())
			}
			if ea.DOIOrg.Status == analyze.SearchStatusDone && ea.DOIOrg.Found {
				ent.Analysis += fmt.Sprintf("doi.org: %s\n", "exists")
			}
			if ea.Online.Status == analyze.SearchStatusDone && ea.Online.Metadata != nil {
				ent.Analysis += fmt.Sprintf("URL: %s\n", ea.Online.Metadata.ToString())
			}
		}
		doc.mu.Unlock()
	}

}

func (s *Server) handleUploadPost(c echo.Context) error {

	log.Println("handleUploadPost")

	// Get file from form
	file, err := c.FormFile("pdf")
	if err != nil {
		return err
	}

	// Generate unique ID
	docID := generateID()

	// Create uploads directory if not exists
	os.MkdirAll("uploads", 0755)

	// Save file
	filename := fmt.Sprintf("%s_%s", docID, file.Filename)
	filepath := filepath.Join("uploads", filename)

	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		return err
	}

	// Create document
	doc := &Document{
		ID:       docID,
		Filename: filename,
		Entries:  []*BibEntry{},
	}

	docMu.Lock()
	documents[docID] = doc
	docMu.Unlock()

	// Start analysis in background
	go doAnalyze(s.shirtyApiKey, doc)

	return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/analyze/%s", docID))
}
