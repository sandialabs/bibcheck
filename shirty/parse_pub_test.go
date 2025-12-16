// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package shirty

import (
	"os"
	"strings"
	"testing"
)

func Test_ParsePub_1(t *testing.T) {
	parse_pub_impl(t,
		"2017. NVIDIA Tesla V100 GPU Architecture. https://images.nvidia.com/ content/volta-architecture/pdf/volta-architecture-whitepaper.pdf",
		"",
	)
}

func Test_ParsePub_2(t *testing.T) {
	parse_pub_impl(t,
		"David H Bailey, Eric Barszcz, John T Barton, David S Browning, Robert L Carter, Leonardo Dagum, Rod A Fatoohi, Paul O Frederickson, Thomas A Lasinski, Rob S Schreiber, et al . 1991. The NAS Parallel Benchmarks—Summary and Preliminary Results. In Proceedings of the 1991 ACM/IEEE Conference on Supercomputing. 158–165.",
		"Proceedings of the 1991 ACM/IEEE Conference on Supercomputing",
	)
}

func Test_ParsePub_3(t *testing.T) {
	parse_pub_impl(t,
		"Christian Bell, Dan Bonachea, Yannick Cote, Jason Duell, Paul Hargrove, Parry Husbands, Costin Iancu, Michael Welcome, and Katherine, Yelick. 2003. An Evaluation of Current High-Performance Networks. In Proceedings International Parallel and Distributed Processing Symposium. IEEE",
		"Proceedings International Parallel and Distributed Processing Symposium",
	)
}

func Test_ParsePub_4(t *testing.T) {
	parse_pub_impl(t,
		"Mahesh Rajan, Doug Doerfler, and Simon Hammond. 2015. Trinity Benchmarks on Intel Xeon Phi (Knights Corner). Technical Report SAND2015-0454C. Sandia National Laboratories. https://www.osti.gov/biblo/1504115.",
		"Technical Report SAND2015-0454C",
	)
}

func parse_pub_impl(t *testing.T, entry, expected string) {

	apiKey, ok := os.LookupEnv("SHIRTY_API_KEY")

	if !ok {
		t.Skip("provide SHIRTY_API_KEY")
	}

	client := NewWorkflow(
		apiKey,
		WithBaseUrl("https://shirty.sandia.gov/api/v1"),
	)

	actual, err := client.ParsePub(entry)
	if err != nil {
		t.Errorf("ParseTitle error: %v", err)
	}

	if !strings.EqualFold(expected, actual) {
		t.Errorf("expected %s, got %s", expected, actual)
	}

}
