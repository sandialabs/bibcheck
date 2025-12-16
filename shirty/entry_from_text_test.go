// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package shirty

import (
	"os"
	"strings"
	"testing"
)

func Test_EntryFromText_20231113_siefert_pmbs_1(t *testing.T) {
	impl(t, "../test/20231113_siefert_pmbs.pdf", 1, []string{
		"2017",
		"NVIDIA Tesla V100 GPU Architecture",
		"https://images.nvidia.com/content/volta-architecture/pdf/volta-architecture-whitepaper.pdf",
	})
}

// tough one, split across two pages
func Test_EntryFromText_20231113_siefert_pmbs_3(t *testing.T) {
	impl(t, "../test/20231113_siefert_pmbs.pdf", 3, []string{
		"2020",
		"NVIDIA A100 Tensor Core GPU Architecture",
		"https://images.nvidia.com/aem-dam/en-zz/Solutions/data-center/nvidia-ampere-architecture-whitepaper.pdf",
	})
}

func Test_EntryFromText_20231113_siefert_pmbs_4(t *testing.T) {
	impl(t, "../test/20231113_siefert_pmbs.pdf", 4, []string{
		"2021",
		"introducing AMD CDNA 2 Architecture",
		"https://www.amd.com/system/files/documents/amd-cdna2-white-paper.pdf",
	})
}

func Test_EntryFromText_20231113_siefert_pmbs_18(t *testing.T) {
	impl(t, "../test/20231113_siefert_pmbs.pdf", 18, []string{
		"1991",
		"David H Bailey",
		"Eric Barszcz",
		"John T Barton",
		"David S Browning",
		"Robert L Carter",
		"Leonardo Dagum",
		"Rod A Fatoohi",
		"Paul O Frederickson",
		"Thomas A Lasinski",
		"Rob S Schreiber",
		"The NAS Parallel Benchmarks",
		"Summary and Preliminary Results",
		"Conference on Supercomputing",
		"158",
		"165",
	})
}

func Test_20231113_siefert_pmbs_35(t *testing.T) {
	impl(t, "../test/20231113_siefert_pmbs.pdf", 35, []string{
		"N Wichmann",
		"C Nuss",
		"P Carrier",
		"R Olson",
		"S Anderson",
		"M Davis",
		"R Baker",
		"E Draeger",
		"S Domino",
		"A Agelastos",
		"M Rajan",
		"2015",
		"Performance on Trinity (a Cray XC40) with Acceptance-Applications and Benchmarks",
		"SAND2016-3635C",
		"https://www.osti.gov/biblio/1365199",
		"Sandia National Laboratories",
	})
}

func impl(t *testing.T, path string, id int, expected []string) {

	apiKey, ok := os.LookupEnv("SHIRTY_API_KEY")

	if !ok {
		t.Skip("provide SHIRTY_API_KEY")
	}

	client := NewWorkflow(
		apiKey,
		WithBaseUrl("https://shirty.sandia.gov/api/v1"),
	)

	// can't directly check that the extracted text has the expected values:
	// it might be split across a pagebreak or something
	textractResp, err := client.Textract(path)
	if err != nil {
		t.Errorf("textract error: %v", err)
	}

	entry, err := client.EntryFromText(textractResp.Text, id)
	if err != nil {
		t.Errorf("entry from text error: %v", err)
	}

	// check that the extracted bibliography entry has the expected strings
	for _, e := range expected {
		if !strings.Contains(strings.ToLower(entry), strings.ToLower(e)) {
			t.Errorf("expected to find \"%s\" in %s", e, entry)
		}
	}

}
