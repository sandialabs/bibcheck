// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package cmd

import (
	"fmt"
	"log"

	"github.com/sandialabs/bibcheck/shirty"
	"github.com/spf13/cobra"
)

var bibCmd = &cobra.Command{
	Use:   "bib [file.pdf]",
	Short: "Extract bibliography",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		filePath := args[0]

		if shirtyApiKey != "" && shirtyBaseUrl != "" {
			shirtyClient := shirty.NewWorkflow(
				shirtyApiKey,
				shirty.WithBaseUrl(shirtyBaseUrl),
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

		} else if openrouterApiKey != "" && openrouterBaseUrl != "" {
			log.Fatalf("for with openrouter not implemented")

		} else {
			log.Fatalf("requires shirty or openrouter API config")
		}
	},
}
