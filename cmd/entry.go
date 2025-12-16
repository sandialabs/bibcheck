// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package cmd

import (
	"fmt"
	"log"
	"strconv"

	"github.com/cwpearson/bibliography-checker/analyze"
	"github.com/cwpearson/bibliography-checker/openrouter"
	"github.com/cwpearson/bibliography-checker/shirty"
	"github.com/spf13/cobra"
)

var entryCmd = &cobra.Command{
	Use:   "entry [file.pdf] [id]",
	Short: "Extract a bibliography entry",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {

		filePath := args[0]
		id, err := strconv.ParseInt(args[1], 10, 32)
		if err != nil {
			log.Fatalf("expected id %s to be int", args[1])
		}

		if shirtyApiKey != "" && shirtyBaseUrl != "" {
			shirtyClient := shirty.NewWorkflow(
				shirtyApiKey,
				shirty.WithBaseUrl(shirtyBaseUrl),
			)

			text, err := shirtyClient.Textract(filePath)
			if err != nil {
				log.Fatalf("textract error: %v", err)
			}

			entryText, err := shirtyClient.EntryFromText(text.Text, int(id))
			if err != nil {
				log.Fatalf("openai error: %v", err)
			}
			fmt.Println(entryText)

		} else if openrouterApiKey != "" && openrouterBaseUrl != "" {

			encoded, err := analyze.Encode(filePath)
			if err != nil {
				log.Fatalf("encode error: %v", err)
			}

			client := openrouter.NewClient(
				openrouterApiKey,
			)

			entryText, err := client.EntryFromRaw(encoded, int(id))
			if err != nil {
				log.Fatalf("analyze error: %v", err)
			}

			fmt.Println(entryText)
		} else {
			log.Fatalf("requires shirty or openrouter API config")
		}
	},
}
