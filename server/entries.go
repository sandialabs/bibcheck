// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func handleGetBibEntries(c echo.Context) error {

	docID := c.Param("id")

	docMu.RLock()
	doc, exists := documents[docID]
	docMu.RUnlock()

	if !exists {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Document not found"})
	}

	doc.mu.RLock()
	entries := doc.Entries
	doc.mu.RUnlock()

	return c.JSON(http.StatusOK, entries)

}
