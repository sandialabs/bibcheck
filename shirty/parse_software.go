// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package shirty

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sandialabs/bibcheck/entries"
	"github.com/sandialabs/bibcheck/openai"
	"github.com/sandialabs/bibcheck/schema"
)

func NewParseSoftwareRF() *openai.ResponseFormat {
	return openai.NewResponseFormat(schema.SoftwareJSONSchema())
}

func (w *Workflow) ParseSoftware(text string) (*entries.Software, error) {
	model := "openai/RedHatAI/Llama-3.3-70B-Instruct-quantized.w8a8"

	req := &openai.ChatRequest{
		Model: model,
		Messages: []openai.Message{
			openai.MakeSystemMessage(`Extract the name, developers, and homepage URL of the software package referenced in this bibliography entry.
If specific information is not provided, leave the field empty.
Produce JSON.`),
			openai.MakeUserMessage(text),
		},
		ResponseFormat: NewParseSoftwareRF(),
	}

	content, err := w.oaiClient.ChatGetChoiceZero(req)
	if err != nil {
		return nil, fmt.Errorf("chat choice 0 error: %w", err)
	}

	s := entries.Software{}
	if err := json.Unmarshal(content, &s); err != nil {
		return nil, fmt.Errorf("couldn't unmarshal structured JSON response: %w", err)
	}

	s.HomepageUrl = strings.TrimSpace(s.HomepageUrl)

	return &s, nil
}
