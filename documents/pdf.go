// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package documents

import (
	"bytes"
	"fmt"

	pdfapi "github.com/pdfcpu/pdfcpu/pkg/api"
)

func PDFPageCount(pdf []byte) (int, error) {
	return pdfapi.PageCount(bytes.NewReader(pdf), nil)
}

func PDFSlicePages(pdf []byte, startPage, endPage int) ([]byte, error) {
	if startPage < 1 {
		return nil, fmt.Errorf("invalid start page %d", startPage)
	}
	if endPage < startPage {
		return nil, fmt.Errorf("invalid page range %d-%d", startPage, endPage)
	}

	var buf bytes.Buffer
	selection := []string{fmt.Sprintf("%d-%d", startPage, endPage)}
	if err := pdfapi.Trim(bytes.NewReader(pdf), &buf, selection, nil); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
