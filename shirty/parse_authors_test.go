// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package shirty

import (
	"os"
	"strings"
	"testing"

	"github.com/sandialabs/bibcheck/entries"
)

func Test_ParseAuthors_1(t *testing.T) {
	parse_authors_impl(t,
		"2017. NVIDIA Tesla V100 GPU Architecture. https://images.nvidia.com/ content/volta-architecture/pdf/volta-architecture-whitepaper.pdf", entries.Authors{
			Authors:    []string{},
			Incomplete: false,
		})
}

func Test_ParseAuthors_2(t *testing.T) {
	parse_authors_impl(t, "David H Bailey, Eric Barszcz, John T Barton, David S Browning, Robert L Carter, Leonardo Dagum, Rod A Fatoohi, Paul O Frederickson, Thomas A Lasinski, Rob S Schreiber, et al . 1991. The NAS Parallel Benchmarks—Summary and Preliminary Results. In Proceedings of the 1991 ACM/IEEE Conference on Supercomputing. 158–165.",
		entries.Authors{Authors: []string{
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
		},
			Incomplete: true,
		},
	)
}

func Test_ParseAuthors_3(t *testing.T) {
	parse_authors_impl(t, "Christian Bell, Dan Bonachea, Yannick Cote, Jason Duell, Paul Hargrove, Parry Husbands, Costin Iancu, Michael Welcome, and Katherine, Yelick. 2003. An Evaluation of Current High-Performance Networks. In Proceedings International Parallel and Distributed Processing Symposium. IEEE",
		entries.Authors{Authors: []string{
			"Christian Bell",
			"Dan Bonachea",
			"Yannick Cote",
			"Jason Duell",
			"Paul Hargrove",
			"Parry Husbands",
			"Costin Iancu",
			"Michael Welcome",
			"Katherine Yelick",
		},
			Incomplete: false,
		},
	)
}

// func Test_ParseAuthors_35(t *testing.T) {
// 	impl(t, "../test/20231113_siefert_pmbs.pdf", 35, []string{
// 		"N Wichmann",
// 		"C Nuss",
// 		"P Carrier",
// 		"R Olson",
// 		"S Anderson",
// 		"M Davis",
// 		"R Baker",
// 		"E Draeger",
// 		"S Domino",
// 		"A Agelastos",
// 		"M Rajan",
// 		"2015",
// 		"Performance on Trinity (a Cray XC40) with Acceptance-Applications and Benchmarks",
// 		"SAND2016-3635C",
// 		"https://www.osti.gov/biblio/1365199",
// 		"Sandia National Laboratories",
// 	})
// }

func parse_authors_impl(t *testing.T, entry string, expected entries.Authors) {

	apiKey, ok := os.LookupEnv("SHIRTY_API_KEY")

	if !ok {
		t.Skip("provide SHIRTY_API_KEY")
	}

	client := NewWorkflow(
		apiKey,
		WithBaseUrl("https://shirty.sandia.gov/api/v1"),
	)

	actual, err := client.ParseAuthors(entry)
	if err != nil {
		t.Errorf("ParseAuthors error: %v", err)
	}

	if expected.Incomplete != actual.Incomplete {
		t.Errorf("expected Incomplete = %v, got %v", expected.Incomplete, actual.Incomplete)
		return
	}

	if len(expected.Authors) != len(actual.Authors) {
		t.Errorf("expected %d authors, got %d", len(expected.Authors), len(actual.Authors))
	}

	for ei, e := range expected.Authors {
		if !strings.EqualFold(e, actual.Authors[ei]) {
			t.Errorf("expected %s, got %s", e, actual.Authors[ei])
		}
	}

}
