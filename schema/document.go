// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package schema

func BibliographyPageJSONSchema() map[string]any {
	return map[string]any{
		"name":   "bibliography_page",
		"strict": true,
		"schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"contains_bibliography": map[string]string{
					"type": "boolean",
				},
			},
			"required":             []string{"contains_bibliography"},
			"additionalProperties": false,
		},
	}
}
