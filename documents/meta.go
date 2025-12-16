// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package documents

type Metadata struct {
	Title           string   `json:"title"`
	Authors         []string `json:"authors"`
	ContributingOrg string   `json:"contributing_org"`
}

type MetaExtractor interface {
	PDFMetadata(content []byte) (*Metadata, error)
	HTMLMetadata(content []byte) (*Metadata, error)
}
