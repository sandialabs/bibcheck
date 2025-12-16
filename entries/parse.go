// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package entries

type Software struct {
	Name        string   `json:"name"`
	Developers  []string `json:"developers"`
	HomepageUrl string   `json:"homepage_url"`
}

type Website struct {
	Title   string   `json:"title"`
	Authors []string `json:"authors"`
	URL     string   `json:"url"`
}

type Parser interface {
	ParseDOI(entry string) (string, error)
	ParseArxiv(entry string) (string, error)
	ParseURL(entry string) (string, error)
	ParseOSTI(entry string) (string, error)
	ParseWebsite(entry string) (*Website, error)
	ParseSoftware(entry string) (*Software, error)
}
