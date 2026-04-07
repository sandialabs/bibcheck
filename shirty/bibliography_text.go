// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package shirty

import (
	"log"
	"regexp"
	"strings"
)

var bibliographyHeading = regexp.MustCompile(`(?i)^(?:[0-9]+(?:\.[0-9]+)*\.?\s+|[ivxlcdm]+\.?\s+)?(?:references|bibliography|works cited|literature cited|cited references)\s*$`)

func bibliographyText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	lines := strings.SplitAfter(text, "\n")
	offset := 0

	for _, line := range lines {
		if bibliographyHeading.MatchString(strings.TrimSpace(line)) {
			sliced := strings.TrimSpace(text[offset:])
			log.Printf("bibliography heading detected: %q; reduced text from %d to %d bytes", strings.TrimSpace(line), len(text), len(sliced))
			return sliced
		}
		offset += len(line)
	}

	trimmed := strings.TrimSpace(text)
	log.Printf("bibliography heading not detected; using full text (%d bytes)", len(trimmed))
	return trimmed
}
