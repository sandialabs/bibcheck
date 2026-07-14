// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sandialabs/bibcheck/internal/server"
)

func main() {
	options := server.Options{}

	flag.StringVar(&options.Addr, "addr", server.DefaultAddr, "Address for the static web UI")
	flag.StringVar(&options.WebDir, "web-dir", server.DefaultWebDir, "Directory containing the static web UI")
	flag.Int64Var(&options.FetchMaxBytes, "fetch-max-bytes", server.DefaultFetchMaxBytes, "Maximum bytes to read from /api/fetch upstream responses")
	flag.Parse()

	if flag.NArg() != 0 {
		fmt.Fprintf(os.Stderr, "unexpected arguments: %v\n", flag.Args())
		os.Exit(2)
	}

	if err := server.Run(options); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
