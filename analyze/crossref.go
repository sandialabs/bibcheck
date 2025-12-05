// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package analyze

import (
	"fmt"

	"github.com/sandialabs/bibcheck/crossref"
)

const (
	CrossrefMatchThreshold float64 = 85 // determined empirically
)

// CrossrefQueryBibliographic
func CrossrefQueryBibliographic(entry string) (*crossref.CrossrefWork, string, error) {

	// search for 2 results
	fmt.Println("query crossref.org...")
	crossrefResp, err := crossref.QueryBibliographic(entry, 2)
	if err != nil {
		return nil, "", fmt.Errorf("crossref API error: %w", err)
	}

	if len(crossrefResp.Message.Items) == 0 {
		return nil, "no matches found", nil
	}

	best := &crossrefResp.Message.Items[0]

	if best.Score < CrossrefMatchThreshold {
		return nil, fmt.Sprintf("best match score %f was less than threshold %f", best.Score, CrossrefMatchThreshold), nil
	}

	if len(crossrefResp.Message.Items) > 1 {
		secondMatch := &crossrefResp.Message.Items[1]

		// Check if there's a tie (scores are too close)
		scoreDiff := best.Score - secondMatch.Score
		scoreThreshold := 0.01 // empirically determined

		if scoreDiff < scoreThreshold {
			return nil, "no single conclusive match", nil
		}
	}

	return best, "", nil
}
