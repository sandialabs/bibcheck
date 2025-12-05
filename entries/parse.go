// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package entries

type Software struct {
	Name        string   `json:"name"`
	Developers  []string `json:"developers"`
	HomepageUrl string   `json:"homepage_url"`
}

type Online struct {
	Title   string   `json:"title"`
	Authors []string `json:"authors"`
	URL     string   `json:"url"`
}

type Authors struct {
	Authors    []string `json:"authors"`
	Incomplete bool     `json:"has_et_al"`
}

type Parser interface {
	ParseDOI(entry string) (string, error)
	ParseArxiv(entry string) (string, error)
	ParseURL(entry string) (string, error)
	ParseOSTI(entry string) (string, error)
	ParseOnline(entry string) (*Online, error)

	ParseAuthors(entry string) (*Authors, error)
	ParseTitle(entry string) (string, error)
	ParsePub(entry string) (string, error)
}
