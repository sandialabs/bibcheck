// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package schema

func ParseAuthorsJSONSchema() map[string]any {
	return map[string]any{
		"name":   "authors",
		"strict": true,
		"schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"authors": map[string]any{
					"type": "array",
					"items": map[string]string{
						"type": "string",
					},
				},
				"has_et_al": map[string]string{
					"type": "boolean",
				},
			},
			"required":             []string{"authors", "has_et_al"},
			"additionalProperties": false,
		},
	}
}

func ParseTitleJSONSchema() map[string]any {
	return map[string]any{
		"name":   "title",
		"strict": true,
		"schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"title": map[string]string{
					"type": "string",
				},
			},
			"required":             []string{"title"},
			"additionalProperties": false,
		},
	}
}

func ParsePubJSONSchema() map[string]any {
	return ParseTitleJSONSchema()
}

func SummaryJSONSchema() map[string]any {
	return map[string]any{
		"name":   "compare",
		"strict": true,
		"schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"explanation": map[string]string{
					"type": "string",
				},
				"possible_mismatch": map[string]string{
					"type": "boolean",
				},
			},
			"required":             []string{"possible_mismatch", "explanation"},
			"additionalProperties": false,
		},
	}
}
