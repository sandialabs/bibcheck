package doi

import (
	"fmt"
	"testing"

	"github.com/sandialabs/bibcheck/internal/testutil"
)

func TestDoi(t *testing.T) {
	testutil.SkipIfTCPUnavailable(t, "doi.org:443")

	record, err := ResolveDOI("https://doi.org/10.1016/j.parco.2018.05.006")
	if err != nil {
		t.Fatalf("ResolveDOI error: %v", err)
	}

	fmt.Print(record)
}
