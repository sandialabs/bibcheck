// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package lookup

import (
	"errors"
	"log"

	"github.com/sandialabs/bibcheck/doi"
)

// returns whether the id was found on DOI.org
func CheckDOI(id string) (bool, error) {
	log.Println("checking doi", id, "...")
	_, err := doi.ResolveDOI(id)

	if err != nil {
		if errors.Is(err, doi.DoesNotExistError) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
