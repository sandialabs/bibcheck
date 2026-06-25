// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package shirty

import (
	"time"

	"github.com/sandialabs/bibcheck/openai"
)

type Workflow struct {
	apiKey    string
	oaiClient *openai.Client
}

type WorkflowOpt func(*Workflow)

func NewWorkflow(apiKey, baseUrl string, options ...WorkflowOpt) *Workflow {
	c := &Workflow{
		apiKey: apiKey,
		oaiClient: openai.NewClient(
			apiKey,
			openai.WithBaseUrl(baseUrl),
			openai.WithTimeout(60*time.Second),
		),
	}
	for _, o := range options {
		o(c)
	}
	return c
}

func WithAuditEnabled(enabled bool) WorkflowOpt {
	return func(w *Workflow) {
		openai.WithAuditEnabled(enabled)(w.oaiClient)
	}
}

func (w *Workflow) OpenAIClient() *openai.Client {
	return w.oaiClient
}
