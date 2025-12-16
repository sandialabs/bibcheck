// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package cmd

import (
	"fmt"
	"log"

	"github.com/cwpearson/bibliography-checker/shirty"
	"github.com/spf13/cobra"
)

var textractCmd = &cobra.Command{
	Use:   "textract [file.pdf]",
	Short: "Extract text from a file using shirty.sandia.gov",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		if shirtyApiKey == "" {
			log.Fatal("please provide --shirty-api-key")
		}
		if shirtyBaseUrl == "" {
			log.Fatal("pelase provide --shirty-base-url")
		}

		filePath := args[0]
		client := shirty.NewWorkflow(shirtyApiKey,
			shirty.WithBaseUrl(shirtyBaseUrl))
		resp, err := client.Textract(filePath)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(resp.Text)
	},
}
