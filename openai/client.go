// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package openai

import (
	"net/http"
	"time"
)

type Client struct {
	apiKey     string
	baseUrl    string
	httpClient *http.Client
}

type ClientOpt func(*Client)

func NewClient(apiKey string, options ...ClientOpt) *Client {
	c := &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
	for _, o := range options {
		o(c)
	}
	return c
}

func WithBaseUrl(baseUrl string) ClientOpt {
	return func(c *Client) {
		c.baseUrl = baseUrl
	}
}

func WithTimeout(t time.Duration) ClientOpt {
	return func(c *Client) {
		c.httpClient.Timeout = t
	}
}
