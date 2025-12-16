// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package documents

type EntryFromRawExtractor interface {
	// Retrieve bib entry `id` from `b64` base-64 encoded PDF file
	EntryFromRaw(b64 string, id int) (string, error)
}

type EntryFromTextExtractor interface {
	// Retrieve bib entry `id` from `text` file text
	EntryFromText(text string, id int) (string, error)
}
