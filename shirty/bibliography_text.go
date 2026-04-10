// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package shirty

import (
	"github.com/sandialabs/bibcheck/bibliography"
)

func bibliographyText(text string) string {
	return bibliography.ReduceText(text)
}
