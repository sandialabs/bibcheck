// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func handleGetAnalysisStatus(c echo.Context) error {

	docID := c.Param("id")

	docMu.RLock()
	doc, exists := documents[docID]
	docMu.RUnlock()

	if !exists {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Document not found"})
	}

	doc.mu.RLock()
	defer doc.mu.RUnlock()

	completed := 0
	for _, entry := range doc.Entries {
		if entry.AnalysisStatus == "completed" {
			completed++
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"total":     len(doc.Entries),
		"completed": completed,
		"entries":   doc.Entries,
	})

}
