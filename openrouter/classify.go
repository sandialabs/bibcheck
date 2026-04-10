// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package openrouter

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/sandialabs/bibcheck/entries"
	"github.com/sandialabs/bibcheck/schema"
)

func NewClassifyEntryResponseFormat() *ResponseFormat {
	return &ResponseFormat{
		Type:       "json_schema",
		JSONSchema: schema.ClassifyEntryJSONSchema(),
	}
}

func (c *Client) Classify(text string) (string, error) {
	baseURL := c.baseUrl
	model := "google/gemini-2.5-flash"

	temperature := new(int)
	*temperature = 0

	req := ChatRequest{
		Model: model,
		Messages: []Message{
			systemString(`Determine what kind of bibliography entry the user provides:
- ` + entries.KindScientificPublication + `
- ` + entries.KindSoftwarePackage + `
- ` + entries.KindWebsite + `
- ` + entries.KindUnknown + `
"` + entries.KindWebsite + `" should be used for anything with a URL that does not fit another category.
Produce JSON.`),
			userString(text),
		},
		ResponseFormat: NewClassifyEntryResponseFormat(),
		Provider: Provider{
			RequireParameters: true,
			Sort:              "price",
		},
		Temperature: temperature,
	}

	resp, err := c.ChatCompletion(req, baseURL)
	if err != nil {
		return entries.KindUnknown, fmt.Errorf("chat completion error: %w", err)
	}

	if len(resp.Choices) != 1 {
		return entries.KindUnknown, fmt.Errorf("expected one choice in response")
	}

	content := resp.Choices[0].Message.Content

	if cstring, ok := content.(string); ok {
		s := struct {
			Kind string `json:"kind"`
		}{}
		if err := json.Unmarshal([]byte(cstring), &s); err != nil {
			log.Println(cstring)
			return entries.KindUnknown, fmt.Errorf("couldn't unmarshal structured JSON response: %w", err)
		}

		return s.Kind, nil
	}

	return entries.KindUnknown, fmt.Errorf("content was not string")
}
