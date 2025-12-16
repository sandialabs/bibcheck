// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package analyze

import (
	"errors"
	"fmt"
	"strings"

	"github.com/sandialabs/bibcheck/osti"
)

// returns nil if no match is found on OSTI
func GetOSTIRecord(id, rawEntry string) (*osti.Record, error) {

	id = strings.TrimPrefix(id, "https://www.osti.gov/biblio/")
	id = strings.TrimPrefix(id, "http://www.osti.gov/biblio/")
	id = strings.TrimPrefix(id, "www.osti.gov/biblio/")
	id = strings.TrimPrefix(id, "osti.gov/biblio/")

	ostiClient := osti.NewClient()

	rec, err := ostiClient.GetRecord(id)
	if errors.Is(err, osti.ErrDoesNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("osti client error: %w", err)
	}

	return rec, err
}
