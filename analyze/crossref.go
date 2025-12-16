// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package analyze

import (
	"fmt"

	"github.com/cwpearson/bibliography-checker/crossref"
)

type CrossrefResult struct {
	Comment string
	Entry   *crossref.CrossrefWork
}

const (
	MatchThreshold float64 = 85 // determine empirically
)

// .Entry is nil if there is no match
func CrossrefBibliography(entry string) (*CrossrefResult, error) {

	// search for 2 results
	fmt.Println("query crossref.org...")
	crossrefResp, err := crossref.QueryBibliographic(entry, 2)
	if err != nil {
		return nil, fmt.Errorf("crossref API error: %w", err)
	}

	if len(crossrefResp.Message.Items) == 0 {
		return &CrossrefResult{
			Comment: "no matches found",
		}, nil
	}

	best := &crossrefResp.Message.Items[0]

	if best.Score < MatchThreshold {
		return &CrossrefResult{
			Comment: fmt.Sprintf("best match score %f was less than threshold %f", best.Score, MatchThreshold),
		}, nil
	}

	if len(crossrefResp.Message.Items) > 1 {
		secondMatch := &crossrefResp.Message.Items[1]

		// Check if there's a tie (scores are too close)
		scoreDiff := best.Score - secondMatch.Score
		scoreThreshold := 0.01 // Adjust this threshold as needed

		if scoreDiff < scoreThreshold {
			return &CrossrefResult{
				Comment: "no conclusive match",
			}, nil
		}
	}

	return &CrossrefResult{
		Comment: "",
		Entry:   best,
	}, nil
}
