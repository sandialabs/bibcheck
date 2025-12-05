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
	return fmt.Sprintf("%s. %s. %s", m.Title, strings.Join(m.Authors, ", "), m.PublicationDate)
}
