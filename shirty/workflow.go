// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package shirty

import (
	"time"

	"github.com/sandialabs/bibcheck/openai"
)

// DefaultModel is the model used for requests unless overridden via WithModel.
const DefaultModel = "openai/RedHatAI/Llama-3.3-70B-Instruct-quantized.w8a8"

type Workflow struct {
	apiKey    string
	baseUrl   string
	model     string
	oaiClient *openai.Client
}

type WorkflowOpt func(*Workflow)

func NewWorkflow(apiKey string, options ...WorkflowOpt) *Workflow {
	c := &Workflow{
		apiKey:  apiKey,
		baseUrl: "https://shirty.sandia.gov/api/v1",
		model:   DefaultModel,
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

// WithModel overrides the default model used for requests. An empty string
// leaves the default in place.
func WithModel(model string) WorkflowOpt {
	return func(w *Workflow) {
		if model != "" {
			w.model = model
		}
	}
}

func WithAuditEnabled(enabled bool) WorkflowOpt {
	return func(w *Workflow) {
		openai.WithAuditEnabled(enabled)(w.oaiClient)
	}
}

func (w *Workflow) OpenAIClient() *openai.Client {
	return w.oaiClient
}
