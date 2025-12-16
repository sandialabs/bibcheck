// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package shirty

import (
	"os"
	"strings"
	"testing"
)

func Test_ParseTitle_1(t *testing.T) {
	parse_title_impl(t,
		"2017. NVIDIA Tesla V100 GPU Architecture. https://images.nvidia.com/ content/volta-architecture/pdf/volta-architecture-whitepaper.pdf",
		"NVIDIA Tesla V100 GPU Architecture",
	)
}

func Test_ParseTitle_2(t *testing.T) {
	parse_title_impl(t,
		"David H Bailey, Eric Barszcz, John T Barton, David S Browning, Robert L Carter, Leonardo Dagum, Rod A Fatoohi, Paul O Frederickson, Thomas A Lasinski, Rob S Schreiber, et al . 1991. The NAS Parallel Benchmarks—Summary and Preliminary Results. In Proceedings of the 1991 ACM/IEEE Conference on Supercomputing. 158–165.",
		"The NAS Parallel Benchmarks—Summary and Preliminary Results",
	)
}

func Test_ParseTitle_3(t *testing.T) {
	parse_title_impl(t,
		"Christian Bell, Dan Bonachea, Yannick Cote, Jason Duell, Paul Hargrove, Parry Husbands, Costin Iancu, Michael Welcome, and Katherine, Yelick. 2003. An Evaluation of Current High-Performance Networks. In Proceedings International Parallel and Distributed Processing Symposium. IEEE",
		"An Evaluation of Current High-Performance Networks",
	)
}

func parse_title_impl(t *testing.T, entry, expected string) {

	apiKey, ok := os.LookupEnv("SHIRTY_API_KEY")

	if !ok {
		t.Skip("provide SHIRTY_API_KEY")
	}

	client := NewWorkflow(
		apiKey,
		WithBaseUrl("https://shirty.sandia.gov/api/v1"),
	)

	actual, err := client.ParseTitle(entry)
	if err != nil {
		t.Errorf("ParseTitle error: %v", err)
	}

	if !strings.EqualFold(expected, actual) {
		t.Errorf("expected %s, got %s", expected, actual)
	}

}
