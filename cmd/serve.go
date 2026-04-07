// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package cmd

import (
	"log"

	"github.com/sandialabs/bibcheck/config"
	"github.com/sandialabs/bibcheck/server"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve [shirty API key]",
	Short: "Run web UI server",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		apiKey := config.Runtime().ShirtyAPIKey
		if len(args) == 1 {
			apiKey = args[0]
		}
		if apiKey == "" {
			log.Fatal("please provide shirty API key via SHIRTY_API_KEY, --shirty-api-key, or serve argument")
		}

		s := server.NewServer(apiKey)
		if err := s.Run(); err != nil {
			log.Fatal(err)
		}
	},
}
