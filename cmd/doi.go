// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package cmd

import (
	"fmt"
	"log"

	"github.com/sandialabs/bibcheck/lookup"
	"github.com/spf13/cobra"
)

var doiCmd = &cobra.Command{
	Use:   "doi [DOI]",
	Short: "Ask doi.org to resolve a DOI",

	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		exists, err := lookup.CheckDOI(args[0])
		if err != nil {
			log.Fatalf("retrieve DOI error: %v", err)
		}

		fmt.Println(exists)

	},
}
