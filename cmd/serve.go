// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package cmd

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

var (
	serveAddr string
	serveDir  string
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Serve the static web UI",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintf(cmd.ErrOrStderr(), "serving %s at http://%s\n", serveDir, serveAddr)
		return http.ListenAndServe(serveAddr, http.FileServer(http.Dir(serveDir)))
	},
}

func init() {
	serveCmd.Flags().StringVar(&serveAddr, "addr", "localhost:8080", "Address for the static web UI")
	serveCmd.Flags().StringVar(&serveDir, "web-dir", "web/static", "Directory containing the static web UI")
}
