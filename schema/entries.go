// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package schema

import "github.com/sandialabs/bibcheck/entries"

func ClassifyEntryJSONSchema() map[string]any {
	return map[string]any{
		"name":   "entry_exists",
		"strict": true,
		"schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"kind": map[string]any{
					"type": "string",
					"enum": []string{
						entries.KindScientificPublication,
						entries.KindSoftwarePackage,
						entries.KindWebsite,
						entries.KindUnknown,
					},
				},
			},
			"required":             []string{"kind"},
			"additionalProperties": false,
		},
	}
}

func DocumentMetadataJSONSchema() map[string]any {
	return map[string]any{
		"name":   "metadata",
		"strict": true,
		"schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"title": map[string]string{
					"type": "string",
				},
				"authors": map[string]any{
					"type": "array",
					"items": map[string]string{
						"type": "string",
					},
				},
				"publication_date": map[string]string{
					"type": "string",
				},
			},
			"required":             []string{"title", "authors", "publication_date"},
			"additionalProperties": false,
		},
	}
}

func ParseURLJSONSchema() map[string]any {
	return map[string]any{
		"name":   "url",
		"strict": true,
		"schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"url": map[string]string{
					"type": "string",
				},
			},
			"required":             []string{"url"},
			"additionalProperties": false,
		},
	}
}

func WebsiteJSONSchema() map[string]any {
	return map[string]any{
		"name":   "website",
		"strict": true,
		"schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"title": map[string]string{
					"type": "string",
				},
				"authors": map[string]any{
					"type": "array",
					"items": map[string]string{
						"type": "string",
					},
				},
				"url": map[string]string{
					"type": "string",
				},
			},
			"required":             []string{"title", "authors", "url"},
			"additionalProperties": false,
		},
	}
}

func SoftwareJSONSchema() map[string]any {
	return map[string]any{
		"name":   "software",
		"strict": true,
		"schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]string{
					"type": "string",
				},
				"developers": map[string]any{
					"type": "array",
					"items": map[string]string{
						"type": "string",
					},
				},
				"homepage_url": map[string]string{
					"type": "string",
				},
			},
			"required":             []string{"name", "developers", "homepage_url"},
			"additionalProperties": false,
		},
	}
}
