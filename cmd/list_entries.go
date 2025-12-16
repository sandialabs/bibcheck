// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package cmd

import (
	"fmt"
	"log"

	"github.com/sandialabs/bibcheck/shirty"
	"github.com/spf13/cobra"
)

var listEntriesCmd = &cobra.Command{
	Use:   "list-entries [file.pdf]",
	Short: "List bibliography entries",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		filePath := args[0]

		if shirtyApiKey != "" && shirtyBaseUrl != "" {

			shirtyWorkflow := shirty.NewWorkflow(
				shirtyApiKey,
				shirty.WithBaseUrl(shirtyBaseUrl),
			)

			text, err := shirtyWorkflow.Textract(filePath)
			if err != nil {
				log.Fatalf("textract error: %v", err)
			}

			format, err := shirtyWorkflow.BibIdFormat(text.Text)
			if err != nil {
				log.Fatalf("error getting bib id format: %v", err)
			}
			switch format {
			case shirty.BibIdFormatNumeric:

				numEntries, err := shirtyWorkflow.NumBibEntries(text.Text)
				if err != nil {
					log.Fatalf("error getting number of entries: %v", err)
				}
				for e := range numEntries {
					fmt.Println(e + 1)
				}

			case shirty.BibIdFormatAlphanumeric:
				log.Fatal("unsupported format:", format)
			default:
				log.Fatal("unexpected format:", format)
			}

		} else if openrouterApiKey != "" && openrouterBaseUrl != "" {

		} else {
			log.Fatal("requires openrouter or shirty API config")
		}
	},
}
