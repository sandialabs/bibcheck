// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package lookup

import (
	"context"
	"fmt"
	"log"

	"github.com/sandialabs/bibcheck/crossref"
)

const (
	CrossrefMatchThreshold float64 = 99 // determined empirically
)

func crossrefQueryBibliographic(client *crossref.Client, entry string) (*crossref.CrossrefWork, string, error) {
	if client == nil {
		return nil, "", fmt.Errorf("crossref client is required")
	}

	// search for 2 results
	log.Print("query crossref.org...")
	crossrefResp, err := client.QueryBibliographic(context.Background(), entry, 2)
	if err != nil {
		return nil, "", fmt.Errorf("crossref API error: %w", err)
	}

	if len(crossrefResp.Message.Items) == 0 {
		return nil, "no matches found", nil
	}

	best := &crossrefResp.Message.Items[0]

	if best.Score < CrossrefMatchThreshold {
		return nil, fmt.Sprintf("score %f < threshold %f", best.Score, CrossrefMatchThreshold), nil
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

	log.Println("crossref.org best score:", best.Score)
	return best, "", nil
}
