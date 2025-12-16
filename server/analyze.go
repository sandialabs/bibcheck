// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package server

import (
	"html/template"
	"net/http"

	"github.com/labstack/echo/v4"
)

func handleAnalyzeGet(c echo.Context) error {
	docID := c.Param("id")

	docMu.RLock()
	doc, exists := documents[docID]
	docMu.RUnlock()

	if !exists {
		return c.String(http.StatusNotFound, "Document not found")
	}

	tmpl := template.Must(template.ParseFiles("server/templates/analyze.html"))
	return tmpl.Execute(c.Response(), map[string]interface{}{
		"DocID":    doc.ID,
		"Filename": doc.Filename,
	})
}
