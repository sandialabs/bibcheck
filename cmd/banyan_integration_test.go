// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

// sampleBanyanBibliography mirrors SAMPLE_BIB in ../../check_bib.py.
const sampleBanyanBibliography = "[1] A. Vaswani et al., 2017, Attention Is All You Need, " +
	"https://arxiv.org/abs/1706.03762 " +
	"[2] K. He, X. Zhang, S. Ren, J. Sun, 2015, Deep Residual Learning for " +
	"Image Recognition, arXiv:1512.03385 " +
	"[3] J. Devlin, M. Chang, K. Lee, K. Toutanova, 2018, BERT: Pre-training " +
	"of Deep Bidirectional Transformers for Language Understanding, " +
	"arXiv:1810.04805 " +
	"[4] D. P. Kingma and J. Ba, 2014, Adam: A Method for Stochastic " +
	"Optimization, arXiv:1412.6980 " +
	"[5] J. Jumper et al., 2021, Highly accurate protein structure prediction " +
	"with AlphaFold, Nature 596, 583-589, " +
	"https://doi.org/10.1038/s41586-021-03819-2 " +
	"[6] J. Smith et al., 2020, A Study of Things, arXiv:2003.99999 " +
	"[7] Q. Fictional and R. Imaginary, 2022, Deep Learning for Unicorn " +
	"Detection, Journal of Made-Up Results 12(3), 45-67." +
	"[8] A. Vaswani et al., 2017, Attention Is All You Need, " +
	"https://arxiv.org/abs/1707.03762 " +
	"[9] K. He, X. Zhang, S. Ren, J. Sun, 2015, Deep Residual Learning for " +
	"https://arxiv.org/abs/1706.03762 "

func TestBanyanIntegrationCheckBibSample(t *testing.T) {
	t.Setenv("ELSEVIER_API_KEY", "")
	t.Setenv("OPENROUTER_API_KEY", "")
	t.Setenv("SHIRTY_API_KEY", "")

	oldTransport := http.DefaultTransport
	http.DefaultTransport = cannedBanyanBibliographyTransport{}
	defer func() { http.DefaultTransport = oldTransport }()

	out, err := executeRootCommandWithStdout(
		strings.NewReader(sampleBanyanBibliography),
		"text", "--format", "json", "--workers", "1", "-",
	)
	if err != nil {
		t.Fatalf("bibcheck text failed: %v", err)
	}

	var doc jsonDocumentView
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, out)
	}

	if doc.Format != "json" {
		t.Fatalf("format = %q, want json", doc.Format)
	}
	if doc.TotalEntries != 9 || len(doc.Entries) != 9 {
		t.Fatalf("got %d total entries and %d rendered entries, want 9 and 9", doc.TotalEntries, len(doc.Entries))
	}

	wantVerdicts := map[int]string{
		1: "VERIFIED",
		2: "VERIFIED",
		3: "VERIFIED",
		4: "VERIFIED",
		5: "DOI-OK",
		6: "SUSPECT",
		7: "SUSPECT",
		8: "MISMATCH",
		9: "MISMATCH",
	}
	for _, entry := range doc.Entries {
		got, why := banyanBibliographyVerdict(entry)
		if got != wantVerdicts[entry.Number] {
			t.Fatalf("entry %d verdict = %s, want %s (%s)\nentry: %+v", entry.Number, got, wantVerdicts[entry.Number], why, entry)
		}
	}
}

func executeRootCommandWithStdout(stdin io.Reader, args ...string) (string, error) {
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		return "", err
	}
	defer r.Close()
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	rootCmd.SetIn(stdin)
	rootCmd.SetArgs(args)
	_, execErr := rootCmd.ExecuteC()

	if closeErr := w.Close(); closeErr != nil && execErr == nil {
		execErr = closeErr
	}
	data, readErr := io.ReadAll(r)
	if readErr != nil && execErr == nil {
		execErr = readErr
	}
	return string(data), execErr
}

type cannedBanyanBibliographyTransport struct{}

func (cannedBanyanBibliographyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	switch req.URL.Host {
	case "export.arxiv.org":
		return stringResponse(req, http.StatusOK, cannedArxivFeed(req.URL.Query().Get("id_list"))), nil
	case "doi.org":
		return stringResponse(req, http.StatusOK, cannedDOIResponse(req.URL.Path)), nil
	case "api.crossref.org":
		return stringResponse(req, http.StatusOK, `{"status":"ok","message":{"total-results":0,"items":[]}}`), nil
	default:
		return nil, fmt.Errorf("unexpected HTTP request: %s %s", req.Method, req.URL.String())
	}
}

func stringResponse(req *http.Request, status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Status:     fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}

func cannedArxivFeed(id string) string {
	switch id {
	case "1706.03762":
		return arxivFeed(arxivEntry{
			ID:        "http://arxiv.org/abs/1706.03762v7",
			Published: "2017-06-12T17:57:34Z",
			Updated:   "2023-08-02T00:41:18Z",
			Title:     "Attention Is All You Need",
			Authors:   []string{"Ashish Vaswani", "Noam Shazeer"},
		})
	case "1512.03385":
		return arxivFeed(arxivEntry{
			ID:        "http://arxiv.org/abs/1512.03385v1",
			Published: "2015-12-10T15:15:43Z",
			Updated:   "2015-12-10T15:15:43Z",
			Title:     "Deep Residual Learning for Image Recognition",
			Authors:   []string{"Kaiming He", "Xiangyu Zhang"},
		})
	case "1810.04805":
		return arxivFeed(arxivEntry{
			ID:        "http://arxiv.org/abs/1810.04805v2",
			Published: "2018-10-11T23:51:14Z",
			Updated:   "2019-05-24T20:16:31Z",
			Title:     "BERT: Pre-training of Deep Bidirectional Transformers for Language Understanding",
			Authors:   []string{"Jacob Devlin", "Ming-Wei Chang"},
		})
	case "1412.6980":
		return arxivFeed(arxivEntry{
			ID:        "http://arxiv.org/abs/1412.6980v9",
			Published: "2014-12-22T00:00:00Z",
			Updated:   "2017-01-30T00:00:00Z",
			Title:     "Adam: A Method for Stochastic Optimization",
			Authors:   []string{"Diederik P. Kingma", "Jimmy Ba"},
		})
	case "1707.03762":
		return arxivFeed(arxivEntry{
			ID:        "http://arxiv.org/abs/1707.03762v1",
			Published: "2017-07-12T00:00:00Z",
			Updated:   "2017-07-12T00:00:00Z",
			Title:     "Densely Connected Convolutional Networks",
			Authors:   []string{"Gao Huang", "Zhuang Liu"},
		})
	default:
		return arxivFeed()
	}
}

type arxivEntry struct {
	ID        string
	Published string
	Updated   string
	Title     string
	Authors   []string
}

func arxivFeed(entries ...arxivEntry) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?><feed xmlns="http://www.w3.org/2005/Atom">`)
	for _, entry := range entries {
		b.WriteString("<entry>")
		fmt.Fprintf(&b, "<id>%s</id>", entry.ID)
		fmt.Fprintf(&b, "<published>%s</published>", entry.Published)
		fmt.Fprintf(&b, "<updated>%s</updated>", entry.Updated)
		fmt.Fprintf(&b, "<title>%s</title>", entry.Title)
		for _, author := range entry.Authors {
			fmt.Fprintf(&b, "<author><name>%s</name></author>", author)
		}
		b.WriteString("</entry>")
	}
	b.WriteString("</feed>")
	return b.String()
}

func cannedDOIResponse(path string) string {
	if path == "/api/handles/10.1038/s41586-021-03819-2" {
		return `{"responseCode":1,"handle":"10.1038/s41586-021-03819-2","values":[]}`
	}
	return `{"responseCode":100,"handle":"","message":"DOI does not exist"}`
}

func banyanBibliographyVerdict(entry jsonEntryView) (string, string) {
	statuses := map[string]string{}
	details := map[string]string{}
	for _, source := range entry.Sources {
		statuses[source.Name] = source.Status
		details[source.Name] = source.Detail
	}

	for _, name := range []string{"arXiv", "OSTI", "Crossref", "Elsevier", "Online"} {
		if isBanyanPositiveStatus(statuses[name]) {
			if !banyanTitleMatches(details[name], entry.OriginalText) {
				return "MISMATCH", fmt.Sprintf("%s record does not match: %s", name, details[name])
			}
			return "VERIFIED", fmt.Sprintf("%s: %s", name, details[name])
		}
	}
	if statuses["DOI"] == "found" {
		return "DOI-OK", "DOI resolves at doi.org"
	}

	var negatives []string
	var errors []string
	for name, status := range statuses {
		if isBanyanNegativeStatus(status) {
			negatives = append(negatives, name)
		}
		if status == "error" {
			errors = append(errors, fmt.Sprintf("%s: %s", name, details[name]))
		}
	}
	if len(errors) > 0 {
		return "RETRY", strings.Join(errors, "; ")
	}
	if len(negatives) > 0 {
		return "SUSPECT", "no source could confirm it"
	}
	return "UNKNOWN", "no checkable identifier and no match"
}

func isBanyanPositiveStatus(status string) bool {
	return status == "found" || status == "matched"
}

func isBanyanNegativeStatus(status string) bool {
	return status == "not-found" || status == "no-match"
}

func banyanTitleMatches(detail, cited string) bool {
	title := banyanArxivTitle(detail)
	if title == "" {
		return true
	}
	words := strings.Fields(strings.ToLower(title))
	if len(words) == 0 {
		return true
	}
	citedLower := strings.ToLower(cited)
	hits := 0
	checked := 0
	for _, word := range words {
		word = strings.Trim(word, ".,:;!?\"'()[]{}")
		if len(word) <= 2 {
			continue
		}
		checked++
		if strings.Contains(citedLower, word) {
			hits++
		}
	}
	if checked == 0 {
		return true
	}
	return float64(hits)/float64(checked) >= 0.5
}

func banyanArxivTitle(detail string) string {
	pub := strings.Index(detail, ". published ")
	if pub == -1 {
		return ""
	}
	before := detail[:pub]
	dot := strings.LastIndex(before, ". ")
	if dot == -1 {
		return ""
	}
	return before[dot+2:]
}
