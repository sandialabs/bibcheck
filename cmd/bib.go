// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package cmd

import (
	"fmt"
	"log"

	"github.com/sandialabs/bibcheck/config"
	"github.com/sandialabs/bibcheck/lookup"
	"github.com/sandialabs/bibcheck/openrouter"
	"github.com/sandialabs/bibcheck/shirty"
	"github.com/spf13/cobra"
)

var bibCmd = &cobra.Command{
	Use:   "bib [file.pdf]",
	Short: "Extract bibliography",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		settings := config.Runtime()

		filePath := args[0]

		if settings.ShirtyAPIKey != "" && settings.ShirtyBaseURL != "" {
			shirtyClient := shirty.NewWorkflow(
				settings.ShirtyAPIKey,
				shirty.WithBaseUrl(settings.ShirtyBaseURL),
			)

			text, err := shirtyClient.Textract(filePath)
			if err != nil {
				log.Fatalf("textract error: %v", err)
			}

			entries, err := shirtyClient.ExtractBib(text.Text)
			if err != nil {
				log.Fatalf("openai error: %v", err)
			}
			for _, entry := range entries {
				fmt.Printf("[%s] %s\n", entry.EntryId, entry.EntryText)
			}

		} else if settings.OpenRouterAPIKey != "" && settings.OpenRouterBaseURL != "" {
			encoded, err := lookup.Encode(filePath)
			if err != nil {
				log.Fatalf("encode error: %v", err)
			}

			client := openrouter.NewClient(
				settings.OpenRouterAPIKey,
				openrouter.WithBaseURL(settings.OpenRouterBaseURL),
			)

			entries, err := client.ExtractBib(encoded)
			if err != nil {
				log.Fatalf("analyze error: %v", err)
			}
			for _, entry := range entries {
				fmt.Printf("[%s] %s\n", entry.EntryId, entry.EntryText)
			}

		} else {
			log.Fatalf("requires shirty or openrouter API config")
		}
	},
}
