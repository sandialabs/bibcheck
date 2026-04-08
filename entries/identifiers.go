// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package entries

import (
	"regexp"
	"strings"
)

var (
	doiRe = regexp.MustCompile(`(?i)(?:https?://(?:dx\.)?doi\.org/|doi:\s*)?(10\.\d{4,9}/[-._;()/:A-Z0-9]+)`)

	arxivURLRe = regexp.MustCompile(`(?i)https?://arxiv\.org/(?:abs|pdf)/([^\s?#]+)`)
	arxivIDRe  = regexp.MustCompile(`(?i)\barxiv:\s*([A-Z\-]+/\d{7}|\d{4}\.\d{4,5}(?:v\d+)?)\b`)

	ostiURLRe = regexp.MustCompile(`(?i)https?://(?:www\.)?osti\.gov/bib(?:lio|lo)/(\d+)`)
	ostiIDRe  = regexp.MustCompile(`(?i)\bOSTI(?:\s+(?:ID|identifier))?\s*[:#]?\s*(\d{4,})\b`)
)

func ExtractDOI(text string) string {
	matches := doiRe.FindStringSubmatch(text)
	if len(matches) < 2 {
		return ""
	}
	return trimIdentifierSuffix(matches[1])
}

func ExtractArxiv(text string) string {
	if matches := arxivURLRe.FindStringSubmatch(text); len(matches) >= 2 {
		id := trimIdentifierSuffix(matches[1])
		id = strings.TrimSuffix(id, ".pdf")
		if id != "" {
			return "https://arxiv.org/abs/" + id
		}
	}

	matches := arxivIDRe.FindStringSubmatch(text)
	if len(matches) < 2 {
		return ""
	}
	return "https://arxiv.org/abs/" + trimIdentifierSuffix(matches[1])
}

func ExtractOSTI(text string) string {
	if matches := ostiURLRe.FindStringSubmatch(text); len(matches) >= 2 {
		return trimIdentifierSuffix(matches[1])
	}

	matches := ostiIDRe.FindStringSubmatch(text)
	if len(matches) < 2 {
		return ""
	}
	return trimIdentifierSuffix(matches[1])
}

func trimIdentifierSuffix(s string) string {
	s = strings.TrimSpace(s)
	for s != "" {
		last := s[len(s)-1]
		switch last {
		case '.', ',', ';', ':', '!', '?', '"', '\'':
			s = s[:len(s)-1]
		case ')':
			if strings.Count(s, "(") < strings.Count(s, ")") {
				s = s[:len(s)-1]
			} else {
				return s
			}
		case ']':
			if strings.Count(s, "[") < strings.Count(s, "]") {
				s = s[:len(s)-1]
			} else {
				return s
			}
		case '}':
			if strings.Count(s, "{") < strings.Count(s, "}") {
				s = s[:len(s)-1]
			} else {
				return s
			}
		default:
			return s
		}
	}
	return s
}
