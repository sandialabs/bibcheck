// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package cmd

import (
	"fmt"
	"log"

	"github.com/sandialabs/bibcheck/config"
	"github.com/sandialabs/bibcheck/shirty"
	"github.com/spf13/cobra"
)

var textractCmd = &cobra.Command{
	Use:   "textract [file.pdf]",
	Short: "Extract text from a file using shirty.sandia.gov",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		settings := config.Runtime()

		if settings.ShirtyAPIKey == "" {
			log.Fatal("please provide SHIRTY_API_KEY or --shirty-api-key")
		}
		if settings.ShirtyBaseURL == "" {
			log.Fatal("please provide SHIRTY_BASE_URL or --shirty-base-url")
		}

		filePath := args[0]
		client := shirty.NewWorkflow(settings.ShirtyAPIKey,
			shirty.WithBaseUrl(settings.ShirtyBaseURL))
		resp, err := client.Textract(filePath)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println(resp.Text)
	},
}
