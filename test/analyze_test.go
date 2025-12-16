// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package analyze_test

import (
	"os"
	"testing"

	"github.com/sandialabs/bibcheck/lookup"
	"github.com/sandialabs/bibcheck/openrouter"
	"github.com/sandialabs/bibcheck/shirty"
)

type Entry struct {
	ID       int  `yaml:"id"`
	Expected bool `yaml:"expected"`
}

type Document struct {
	Name    string  `yaml:"name"`
	Entries []Entry `yaml:"entries"`
}

// first entry
func Test_20231113_siefert_pmbs_1_true(t *testing.T) {
	impl(t, "20231113_siefert_pmbs.pdf", 1, true)
}

// last entry
// should be 2016, not 2015
// author list is in the wrong order
func Test_20231113_siefert_pmbs_35_true(t *testing.T) {
	impl(t, "20231113_siefert_pmbs.pdf", 35, true)
}

func impl(t *testing.T, path string, id int, expected bool) {

	var ea *lookup.EntryAnalysis
	var err error

	if apiKey, ok := os.LookupEnv("SHIRTY_API_KEY"); ok {

		client := shirty.NewWorkflow(
			apiKey,
			shirty.WithBaseUrl("https://shirty.sandia.gov/api/v1"),
		)

		var tResp *shirty.TextractResponse
		tResp, err = client.Textract(path)
		if err != nil {
			t.Errorf("textract error: %v", err)
		}

		ea, err = lookup.EntryFromText(tResp.Text, id, "auto",
			client, client, client, client, nil)

	} else if apiKey, ok := os.LookupEnv("OPENROUTER_API_KEY"); ok {

		client := openrouter.NewClient(apiKey)

		var encoded string
		encoded, err = lookup.Encode(path)
		if err != nil {
			t.Errorf("encode error: %v", err)
		}

		ea, err = lookup.EntryFromBase64(encoded, id, "auto",
			client, client, client, client, nil)

	} else {
		t.Skip("provide either SHIRTY_API_KEY or OPENROUTER_API_KEY")
	}

	if err != nil {
		t.Errorf("analyze error: %v", err)
	}

	lookup.Print(ea)

	exists := ea.Arxiv.Entry != nil || ea.Crossref.Work != nil ||
		ea.Elsevier.Result != nil || ea.OSTI.Record != nil || ea.Online.Metadata != nil

	if exists != expected {
		t.Errorf("Document %s Entry %d: expected %v got %v", path, id, expected, exists)
	}
}
