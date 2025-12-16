package lookup

import (
	"fmt"
	"os"
	"testing"

	"github.com/sandialabs/bibcheck/elsevier"
	"github.com/sandialabs/bibcheck/shirty"
)

func shirtyWorkflowFromEnv() *shirty.Workflow {
	if apiKey, ok := os.LookupEnv("SHIRTY_API_KEY"); ok {
		return shirty.NewWorkflow(apiKey)
	}
	return nil
}

func elsevierClientFromEnv() *elsevier.Client {
	if apiKey, ok := os.LookupEnv("ELSEVIER_API_KEY"); ok {
		return elsevier.NewClient(apiKey)
	}
	return nil
}

func Test_DOIExists_1(t *testing.T) {
	if w := shirtyWorkflowFromEnv(); w != nil {
		text := `Brice Goglin, Emmanuel Jeannot, Farouk Mansouri, and Guillaume
Mercier. 2018. Hardware Topology Management in MPI Applications
through Hierarchical Communicators. Parallel Comput. 76 (2018),
70–90. https://doi.org/10.1016/j.parco.2018.05.006`
		expected := "10.1016/j.parco.2018.05.006"

		if EA, err := Entry(text, "", w, w, w, &EntryConfig{
			ElsevierClient: elsevierClientFromEnv(),
		}); err != nil {
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
		t.Skip("SHIRTY_API_KEY not provided")
	}
}

func Test_ArxivExists_1(t *testing.T) {

	if w := shirtyWorkflowFromEnv(); w != nil {
		text := `Sivasankaran Rajamanickam, Seher Acer, Luc Berger-Vergiat, Vinh Dang, Nathan Ellingwood, Evan Harvey, Brian
Kelley, Christian R Trott, Jeremiah Wilke, and Ichitaro Yamazaki. 2021. Kokkos kernels: Performance portable
sparse/dense linear algebra and graph kernels. arXiv preprint arXiv:2103.11991 -, - (2021), 1–12`
		expected := "https://arxiv.org/abs/2103.11991"

		if EA, err := Entry(text, "", w, w, w, &EntryConfig{
			ElsevierClient: elsevierClientFromEnv(),
		}); err != nil {
			t.Fatalf("Entry error: %v", err)
		} else {
			if EA.Arxiv.ID != expected {
				t.Fatalf("Found wrong Arxiv ID: expected=%v actual=%v", expected, EA.Arxiv.ID)
			}
		}
	} else {
		t.Skip("SHIRTY_API_KEY not provided")
	}

}

// Test_NotArxiv_1 makes sure we don't find an arxiv entry when there isn't one
func Test_NotArxiv_1(t *testing.T) {

	if w := shirtyWorkflowFromEnv(); w != nil {
		text := `Brice Goglin, Emmanuel Jeannot, Farouk Mansouri, and Guillaume
Mercier. 2018. Hardware Topology Management in MPI Applications
through Hierarchical Communicators. Parallel Comput. 76 (2018),
70–90. https://doi.org/10.1016/j.parco.2018.05.006`

		if EA, err := Entry(text, "", w, w, w, &EntryConfig{
			ElsevierClient: elsevierClientFromEnv(),
		}); err != nil {
			t.Fatalf("Entry error: %v", err)
		} else if EA.Arxiv.Entry != nil {
			t.Fatalf("made up an Arxiv entry")
		}
	} else {
		t.Skip("SHIRTY_API_KEY not provided")
	}

}

func Test_Elsevier_1(t *testing.T) {

	w := shirtyWorkflowFromEnv()
	e := elsevierClientFromEnv()

	if w == nil {
		t.Skip("nil shirty workflow")
	}
	if e == nil {
		t.Skip("nil elsevier client")
	}

	text := `Sergio Sarmiento-Rosales, Víctor Adrían Sosa Hernández, Raúl Monroy,
Evolutionary Neural Architecture Search for Super-Resolution: Benchmarking SynFlow and model-based predictors,
Swarm and Evolutionary Computation,
Volume 100,
2026,`

	EA, err := Entry(text, "", w, w, w, &EntryConfig{
		ElsevierClient: elsevierClientFromEnv(),
	})
	if err != nil {
		t.Fatalf("Entry error: %v", err)
	}

	if EA.Arxiv.Entry != nil {
		t.Fatalf("made up an Arxiv entry")
	}
	if EA.DOIOrg.ID != "" {
		t.Fatalf("made up a DOI")
	}
	if EA.OSTI.Record != nil {
		t.Fatalf("made up an OSTI ID")
	}
	if EA.Elsevier.Error != nil {
		t.Fatalf("Elsevier error: %v", EA.Elsevier.Error)
	}
	if EA.Elsevier.Result == nil {
		t.Fatalf("Elsevier search failed")
	}

	if EA.Elsevier.Result.DOI != "10.1016/j.swevo.2025.102236" {
		t.Fatalf("Elsevier search returned wrong result")
	}

	fmt.Println(EA.Elsevier.Result)
}

func Test_Crossref_1(t *testing.T) {

	w := shirtyWorkflowFromEnv()

	if w == nil {
		t.Skip("nil shirty workflow")
	}

	text := `Sergio Sarmiento-Rosales, Víctor Adrían Sosa Hernández, Raúl Monroy,
Evolutionary Neural Architecture Search for Super-Resolution: Benchmarking SynFlow and model-based predictors,
Swarm and Evolutionary Computation,
Volume 100,
2026,`

	EA, err := Entry(text, "", w, w, w, nil)
	if err != nil {
		t.Fatalf("Entry error: %v", err)
	}

	if EA.Arxiv.Entry != nil {
		t.Fatalf("made up an Arxiv entry")
	}
	if EA.DOIOrg.ID != "" {
		t.Fatalf("made up a DOI")
	}
	if EA.OSTI.Record != nil {
		t.Fatalf("made up an OSTI ID")
	}
	if EA.Crossref.Error != nil {
		t.Fatalf("Crossref error: %v", EA.Crossref.Error)
	}
	if EA.Crossref.Work == nil {
		t.Fatalf("Crossref search failed")
	}

	if EA.Crossref.Work.DOI != "10.1016/j.swevo.2025.102236" {
		t.Fatalf("Crossref search returned wrong result")
	}

	fmt.Println(EA.Crossref.Work)
}

func Test_Online_1(t *testing.T) {

	if w := shirtyWorkflowFromEnv(); w != nil {
		text := `2023. Frontier User Guide. https://docs.olcf.ornl.gov/systems/frontier_
user_guide.html`

		EA, err := Entry(text, "", w, w, w, nil)
		if err != nil {
			t.Fatalf("Entry error: %v", err)
		}

		if EA.Arxiv.Entry != nil {
			t.Fatalf("made up an Arxiv entry")
		}
		if EA.DOIOrg.ID != "" {
			t.Fatalf("made up a DOI")
		}
		if EA.OSTI.Record != nil {
			t.Fatalf("made up an OSTI ID")
		}
		if EA.Crossref.Work != nil {
			t.Fatalf("extraneous crossref result")
		}
		if EA.Elsevier.Result != nil {
			t.Fatalf("extraneous elsevier result")
		}

		if EA.Online.Error != nil {
			t.Fatalf("online error: %v", EA.Online.Error)
		}
		if EA.Online.Metadata.Title != "Frontier User Guide" {
			t.Fatalf("online retrieved wrong title")
		}

	} else {
		t.Skip("SHIRTY_API_KEY not provided")
	}

}
