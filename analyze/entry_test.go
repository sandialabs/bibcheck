package analyze

import (
	"os"
	"testing"

	"github.com/cwpearson/bibliography-checker/shirty"
)

func newShirtyWorkflow() *shirty.Workflow {
	if apiKey, ok := os.LookupEnv("SHIRTY_API_KEY"); ok {
		return shirty.NewWorkflow(apiKey)
	}
	return nil
}

func Test_DOIExists_1(t *testing.T) {
	if w := newShirtyWorkflow(); w != nil {
		text := `Brice Goglin, Emmanuel Jeannot, Farouk Mansouri, and Guillaume
Mercier. 2018. Hardware Topology Management in MPI Applications
through Hierarchical Communicators. Parallel Comput. 76 (2018),
70–90. https://doi.org/10.1016/j.parco.2018.05.006`
		expected := "10.1016/j.parco.2018.05.006"

		if EA, err := Entry(text, "", w, w, w, w, nil, nil); err != nil {
			t.Fatalf("Entry error: %v", err)
		} else {
			if EA.DOIOrg.ID != expected {
				t.Fatalf("Found wrong DOI: expected=%v actual=%v", expected, EA.DOIOrg.ID)
			}
			if !EA.DOIOrg.Found {
				t.Fatalf("Should have found existing DOI")
			}
		}
	} else {
		t.Skip("no SHIRTY_API_KEY not provided")
	}
}

func Test_ArxivExists_1(t *testing.T) {

	if w := newShirtyWorkflow(); w != nil {
		text := `Sivasankaran Rajamanickam, Seher Acer, Luc Berger-Vergiat, Vinh Dang, Nathan Ellingwood, Evan Harvey, Brian
Kelley, Christian R Trott, Jeremiah Wilke, and Ichitaro Yamazaki. 2021. Kokkos kernels: Performance portable
sparse/dense linear algebra and graph kernels. arXiv preprint arXiv:2103.11991 -, - (2021), 1–12`
		expected := "https://arxiv.org/abs/2103.11991"

		if EA, err := Entry(text, "", w, w, w, w, nil, nil); err != nil {
			t.Fatalf("Entry error: %v", err)
		} else {
			if EA.Arxiv.ID != expected {
				t.Fatalf("Found wrong Arxiv ID: expected=%v actual=%v", expected, EA.Arxiv.ID)
			}
		}
	} else {
		t.Skip("no SHIRTY_API_KEY not provided")
	}

}

// Test_NotArxiv_1 makes sure we don't find an arxiv entry when there isn't one
func Test_NotArxiv_1(t *testing.T) {

	if w := newShirtyWorkflow(); w != nil {
		text := `Brice Goglin, Emmanuel Jeannot, Farouk Mansouri, and Guillaume
Mercier. 2018. Hardware Topology Management in MPI Applications
through Hierarchical Communicators. Parallel Comput. 76 (2018),
70–90. https://doi.org/10.1016/j.parco.2018.05.006`

		if EA, err := Entry(text, "", w, w, w, w, nil, nil); err != nil {
			t.Fatalf("Entry error: %v", err)
		} else if EA.Arxiv.Entry != nil {
			t.Fatalf("made up an Arxiv entry")
		}
	} else {
		t.Skip("no SHIRTY_API_KEY not provided")
	}

}
