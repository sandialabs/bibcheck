// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package schema

func BibIDFormatJSONSchema(numericValue, alphanumericValue string) map[string]any {
	return map[string]any{
		"name":   "bib_id_format",
		"strict": true,
		"schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"id_format": map[string]any{
					"type": "string",
					"enum": []string{numericValue, alphanumericValue},
				},
			},
			"required":             []string{"id_format"},
			"additionalProperties": false,
		},
	}
}

func ExtractBibJSONSchema() map[string]any {
	return map[string]any{
		"name":       "bibliography",
		"properties": map[string]any{},
		"strict":     true,
		"schema": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"entry_id": map[string]string{
						"type": "string",
					},
					"entry_text": map[string]string{
						"type": "string",
					},
				},
				"required":             []string{"entry_id", "entry_text"},
				"additionalProperties": false,
			},
		},
	}
}

func NumEntriesJSONSchema(name, numberType string) map[string]any {
	return map[string]any{
		"name":   name,
		"strict": true,
		"schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"num_entries": map[string]string{
					"type": numberType,
				},
			},
			"required":             []string{"num_entries"},
			"additionalProperties": false,
		},
	}
}

func BibliographyEntryJSONSchema() map[string]any {
	return map[string]any{
		"name":   "bib_entry",
		"strict": true,
		"schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"entry_id": map[string]string{
					"type": "string",
				},
				"entry_text": map[string]string{
					"type": "string",
				},
			},
			"required":             []string{"entry_id", "entry_text"},
			"additionalProperties": false,
		},
	}
}

func BibliographyEntryLookupJSONSchema() map[string]any {
	return map[string]any{
		"name":   "bib_entry",
		"strict": true,
		"schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"entry_exists": map[string]string{
					"type": "boolean",
				},
				"bibliography_entry": map[string]string{
					"type": "string",
				},
			},
			"required":             []string{"entry_exists"},
			"additionalProperties": false,
		},
	}
}
