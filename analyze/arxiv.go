// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package analyze

import (
	"errors"
	"fmt"

	"github.com/cwpearson/bibliography-checker/arxiv"
)

// returns nil if not found
func GetArxivMetadata(id, rawEntry string) (*arxiv.Entry, error) {

	arxivClient := arxiv.NewClient()

	rec, err := arxivClient.GetByID(id)

	if errors.Is(err, arxiv.ErrDoesNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("arxiv client error: %w", err)
	}

	return rec, nil
}
