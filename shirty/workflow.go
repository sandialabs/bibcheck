// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package shirty

import "github.com/cwpearson/bibliography-checker/openai"

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
