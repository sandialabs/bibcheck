// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package shirty

import (
	"encoding/json"
	"fmt"

	"github.com/sandialabs/bibcheck/entries"
	"github.com/sandialabs/bibcheck/openai"
	"github.com/sandialabs/bibcheck/schema"
)

func NewParseOnlineRF() *openai.ResponseFormat {
	return openai.NewResponseFormat(schema.WebsiteJSONSchema())
}

func (w *Workflow) ParseOnline(text string) (*entries.Online, error) {
	model := "openai/RedHatAI/Llama-3.3-70B-Instruct-quantized.w8a8"

	req := &openai.ChatRequest{
		Model: model,
		Messages: []openai.Message{
			openai.MakeSystemMessage(`Extract the title, authors, and URL of the online resource from this bibliography entry.
- Produce JSON
- If the bibliography entry does not appear to be an online resource (e.g., no URL), produce an empty string for all values
- If the title or authors are missing, produce an empty string for the corresponding value
`),
			openai.MakeUserMessage(text),
		},
		ResponseFormat: NewParseOnlineRF(),
	}

	content, err := w.oaiClient.ChatGetChoiceZero(req)
	if err != nil {
		return nil, fmt.Errorf("chat choice 0 error: %w", err)
	}

	online := entries.Online{}
	if err := json.Unmarshal(content, &online); err != nil {
		return nil, fmt.Errorf("couldn't unmarshal structured JSON response: %w", err)
	}

	return &online, nil
}
