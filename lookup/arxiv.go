// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package lookup

import (
	"errors"
	"fmt"

	"github.com/sandialabs/bibcheck/arxiv"
)

var defaultArxivClient = arxiv.NewClient()

// returns nil if not found
func GetArxivMetadata(id, rawEntry string) (*arxiv.Entry, error) {
	return getArxivMetadata(defaultArxivClient, id, rawEntry)
}

func getArxivMetadata(arxivClient *arxiv.Client, id, rawEntry string) (*arxiv.Entry, error) {
	rec, err := arxivClient.GetByID(id)

	if errors.Is(err, arxiv.ErrDoesNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("arxiv client error: %w", err)
	}

	return rec, nil
}
