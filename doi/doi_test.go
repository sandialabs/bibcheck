package doi

import (
	"fmt"
	"testing"
)

func TestDoi(t *testing.T) {

	record, err := ResolveDOI("https://doi.org/10.1016/j.parco.2018.05.006")
	if err != nil {
		t.Fatalf("ResolveDOI error: %v", err)
	}

	fmt.Print(record)
}
