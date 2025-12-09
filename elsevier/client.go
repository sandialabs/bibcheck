// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package elsevier

import (
	"time"
)

type Client struct {
	apiKey  string
	baseUrl string
	timeout time.Duration
}

type ClientOpt func(*Client)

func NewClient(apiKey string, options ...ClientOpt) *Client {
	c := &Client{
		apiKey:  apiKey,
		baseUrl: "https://api.elsevier.com",
		timeout: 10 * time.Second,
	}
	for _, o := range options {
		o(c)
	}
	return c
}

func WithTimeout(t time.Duration) ClientOpt {
	return func(c *Client) {
		c.timeout = t
	}
}
