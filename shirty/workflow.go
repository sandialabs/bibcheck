// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package shirty

import (
	"fmt"
	"time"

	"github.com/sandialabs/bibcheck/openai"
)

type Workflow struct {
	apiKey    string
	baseUrl   string
	oaiClient *openai.Client
}

type WorkflowOpt func(*Workflow)

func NewWorkflow(apiKey string, options ...WorkflowOpt) *Workflow {
	c := &Workflow{
		apiKey:  apiKey,
		baseUrl: "https://shirty.sandia.gov/api/v1",
		oaiClient: openai.NewClient(
			apiKey,
			openai.WithBaseUrl("https://shirty.sandia.gov/api/v1"),
			openai.WithTimeout(60*time.Second),
		),
	}
	for _, o := range options {
		o(c)
	}
	return c
}

func WithBaseUrl(baseUrl string) WorkflowOpt {
	return func(w *Workflow) {
		w.baseUrl = baseUrl
		openai.WithBaseUrl(baseUrl)(w.oaiClient)
	}
}

func (w *Workflow) ChatGetChoiceZero(req *openai.ChatRequest) ([]byte, error) {
	resp, err := w.oaiClient.Chat(req)
	if err != nil {
		return nil, fmt.Errorf("openai error: %w", err)
	}
	if len(resp.Choices) != 1 {
		return nil, fmt.Errorf("expected 1 choice in openai response")
	}
	return []byte(resp.Choices[0].Message.Content), nil
}
