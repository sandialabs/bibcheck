package elsevier

import (
	"os"
	"testing"
	"time"

	"github.com/sandialabs/bibcheck/internal/testutil"
)

func TestSearch(t *testing.T) {

	apiKey, ok := os.LookupEnv("ELSEVIER_API_KEY")

	if !ok {
		t.Skipf("ELSEVIER_API_KEY not provided")
	}
	testutil.SkipIfTCPUnavailable(t, "api.elsevier.com:443")

	client := NewClient(apiKey, WithTimeout(10*time.Second))

	_, err := client.Search(&SearchQuery{
		Authors: "IDO, NOTEXIST",
	})
	if err != nil {
		t.Errorf("elsevier client error: %v", err)
	}
}
