package entries

import "testing"

func TestExtractDOI(t *testing.T) {
	t.Run("url", func(t *testing.T) {
		got := ExtractDOI(`Available at https://doi.org/10.1016/j.parco.2018.05.006.`)
		if got != "10.1016/j.parco.2018.05.006" {
			t.Fatalf("unexpected DOI: %q", got)
		}
	})

	t.Run("label", func(t *testing.T) {
		got := ExtractDOI(`DOI: 10.1145/3458817.3476165`)
		if got != "10.1145/3458817.3476165" {
			t.Fatalf("unexpected DOI: %q", got)
		}
	})

	t.Run("ignore non-doi", func(t *testing.T) {
		if got := ExtractDOI(`https://www.osti.gov/biblio/1504115 and arXiv:2103.11991`); got != "" {
			t.Fatalf("expected empty DOI, got %q", got)
		}
	})
}

func TestExtractArxiv(t *testing.T) {
	t.Run("identifier", func(t *testing.T) {
		got := ExtractArxiv(`arXiv preprint arXiv:2103.11991 -`)
		if got != "https://arxiv.org/abs/2103.11991" {
			t.Fatalf("unexpected arXiv URL: %q", got)
		}
	})

	t.Run("pdf url", func(t *testing.T) {
		got := ExtractArxiv(`See https://arxiv.org/pdf/2103.11991.pdf.`)
		if got != "https://arxiv.org/abs/2103.11991" {
			t.Fatalf("unexpected arXiv URL: %q", got)
		}
	})

	t.Run("missing", func(t *testing.T) {
		if got := ExtractArxiv(`Parallel Comput. 76 (2018), 70-90.`); got != "" {
			t.Fatalf("expected empty arXiv, got %q", got)
		}
	})
}

func TestExtractOSTI(t *testing.T) {
	t.Run("biblio url", func(t *testing.T) {
		got := ExtractOSTI(`https://www.osti.gov/biblio/1365199`)
		if got != "1365199" {
			t.Fatalf("unexpected OSTI ID: %q", got)
		}
	})

	t.Run("biblo typo url", func(t *testing.T) {
		got := ExtractOSTI(`https://www.osti.gov/biblo/1504115.`)
		if got != "1504115" {
			t.Fatalf("unexpected OSTI ID: %q", got)
		}
	})

	t.Run("label", func(t *testing.T) {
		got := ExtractOSTI(`Technical report, OSTI ID: 1234567`)
		if got != "1234567" {
			t.Fatalf("unexpected OSTI ID: %q", got)
		}
	})
}
