// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package openai

import "time"

type Client struct {
	apiKey  string
	baseUrl string
	timeout time.Duration
}

type ClientOpt func(*Client)

func NewClient(apiKey string, options ...ClientOpt) *Client {
	c := &Client{
		apiKey:  apiKey,
		timeout: 30 * time.Second,
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
		c.timeout = t
	}
}
