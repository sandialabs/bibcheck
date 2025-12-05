// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package documents

import (
	"fmt"
	"strings"
)

type Metadata struct {
	Title           string   `json:"title"`
	Authors         []string `json:"authors"`
	PublicationDate string   `json:"publication_date"`
}

type MetaExtractor interface {
	PDFMetadata(content []byte) (*Metadata, error)
	HTMLMetadata(content []byte) (*Metadata, error)
}

func (m *Metadata) ToString() string {

	s := ""
	if len(m.Authors) > 0 {
		s += strings.Join(m.Authors, ", ") + "."
	}
	if m.Title != "" {
		if s != "" {
			s += " "
		}
		s += m.Title
	}
	if m.PublicationDate != "" {
		if s != "" {
			s += " "
		}
		s += fmt.Sprintf("(%s)", m.PublicationDate)
	}

	return s
}
