package analyze

import (
	"os"
	"testing"

	"github.com/cwpearson/bibliography-checker/shirty"
)

func Test_DOIExists_1(t *testing.T) {
	if apiKey, ok := os.LookupEnv("SHIRTY_API_KEY"); ok {
		text := `Brice Goglin, Emmanuel Jeannot, Farouk Mansouri, and Guillaume
Mercier. 2018. Hardware Topology Management in MPI Applications
through Hierarchical Communicators. Parallel Comput. 76 (2018),
70–90. https://doi.org/10.1016/j.parco.2018.05.006`
		expected := "10.1016/j.parco.2018.05.006"

		w := shirty.NewWorkflow(apiKey)

		if EA, err := Entry(text, "", w, w, w, w, nil, nil); err != nil {
			t.Fatalf("Entry error: %v", err)
		} else {
			if EA.DOIOrg.DOI != expected {
				t.Fatalf("Found wrong DOI: expected=%v actual=%v", expected, EA.DOIOrg.DOI)
			}
			if !EA.DOIOrg.Found {
				t.Fatalf("Should have found existing DOI")
			}
		}
	} else {
		t.Skip("no SHIRTY_API_KEY not provided")
	}
}
