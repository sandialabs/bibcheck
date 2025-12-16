package elsevier

import (
	"testing"
)

func TestQuery(t *testing.T) {

	q := Query{
		Authors: []string{
			"Carl Pearson",
		},
	}

	got := q.toString()

	if got != "aut(Carl Pearson)" {
		t.Errorf("toString mismatch")
	}

}
