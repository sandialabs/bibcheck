// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package cmd

import (
	"log"

	"github.com/cwpearson/bibliography-checker/server"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve [shirty API key]",
	Short: "Run web UI server",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		shirtyApiKey = args[0]

		s := server.NewServer(shirtyApiKey)
		if err := s.Run(); err != nil {
			log.Fatal(err)
		}
	},
}
