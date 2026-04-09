// Copyright 2026 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package cmd

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/sandialabs/bibcheck/eval"
)

var (
	evalWorkspace string
)

const (
	FlagEvalWorkspace string = "workspace"
)

var evalCmd = &cobra.Command{
	Use:   "eval",
	Short: "Corpus evaluation workflows",
}

var evalDiscoverCmd = &cobra.Command{
	Use:   "discover <corpus-root>",
	Short: "Discover a corpus and write eval workspace metadata",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		corpusRoot := args[0]

		workspaceRoot, err := filepath.Abs(evalWorkspace)
		if err != nil {
			return fmt.Errorf("resolve eval workspace %q: %w", evalWorkspace, err)
		}

		corpus, err := eval.DiscoverCorpus(corpusRoot)
		if err != nil {
			return err
		}

		workspace := eval.NewWorkspace(workspaceRoot)
		if err := workspace.SaveCorpus(corpus); err != nil {
			return err
		}

		fmt.Printf("Discovered %d venues and %d papers\n", len(corpus.Venues), len(corpus.Papers))
		fmt.Printf("Wrote %s\n", workspace.CorpusPath())
		return nil
	},
}

var evalRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run corpus evaluation",
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("not implemented")
	},
}

var evalReviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Review stored corpus results",
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("not implemented")
	},
}

var evalReportCmd = &cobra.Command{
	Use:   "report",
	Short: "Report corpus metrics",
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("not implemented")
	},
}

func init() {
	evalCmd.PersistentFlags().StringVar(&evalWorkspace, FlagEvalWorkspace, eval.DefaultWorkspaceDir, "Eval workspace directory")

	evalCmd.AddCommand(evalDiscoverCmd)
	evalCmd.AddCommand(evalRunCmd)
	evalCmd.AddCommand(evalReviewCmd)
	evalCmd.AddCommand(evalReportCmd)
}
