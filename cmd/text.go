// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	analysisrunner "github.com/sandialabs/bibcheck/analysis"
	"github.com/sandialabs/bibcheck/bibliography"
	"github.com/sandialabs/bibcheck/config"
	"github.com/sandialabs/bibcheck/crossref"
	"github.com/sandialabs/bibcheck/documents"
	"github.com/sandialabs/bibcheck/elsevier"
	"github.com/sandialabs/bibcheck/entries"
	"github.com/sandialabs/bibcheck/lookup"
	"github.com/sandialabs/bibcheck/openrouter"
	"github.com/sandialabs/bibcheck/shirty"
)

var (
	textCarelessHideOK bool
	textFormat         outputFormat
	textWorkers        int
)

var textCmd = &cobra.Command{
	Use:   "text [file|-]",
	Short: "Check bibliography entries provided as plain text",
	Long: `Check a bibliography passed as plain text instead of a PDF, read from a file
or from stdin when no file (or "-") is given. Entries are split on bracket
markers like [1] or [Smith97], falling back to blank-line or per-line
splitting, without any LLM. Entries containing arXiv, DOI, or OSTI
identifiers are verified against public APIs and require no API keys;
Shirty/OpenRouter/Elsevier configuration is used when present to check
entries without identifiers.`,
	Args:          cobra.MaximumNArgs(1),
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validateOutputFormat(textFormat); err != nil {
			return err
		}

		input, err := readTextInput(cmd, args)
		if err != nil {
			return err
		}

		split := bibliography.SplitEntries(input)
		if len(split) == 0 {
			return fmt.Errorf("no bibliography entries found in input")
		}

		// set up optional clients depending on config; unlike the PDF flow,
		// no provider is required
		settings := config.Runtime()
		var openrouterClient *openrouter.Client
		var shirtyProvider *shirty.Workflow
		if settings.OpenRouterAPIKey != "" && settings.OpenRouterBaseURL != "" {
			openrouterClient = openrouter.NewClient(
				settings.OpenRouterAPIKey,
				openrouter.WithBaseURL(settings.OpenRouterBaseURL),
			)
		}
		if settings.ShirtyAPIKey != "" && settings.ShirtyBaseURL != "" {
			shirtyProvider = shirty.NewWorkflow(
				settings.ShirtyAPIKey,
				settings.ShirtyBaseURL,
				shirty.WithModel(settings.ShirtyModel),
			)
		}

		var elsevierClient *elsevier.Client
		if settings.ElsevierAPIKey != "" {
			elsevierClient = elsevier.NewClient(settings.ElsevierAPIKey)
		}

		var class entries.Classifier
		var entryParser entries.Parser
		var docMeta documents.MetaExtractor
		var summarizer summarizer
		if openrouterClient != nil {
			class = openrouterClient
			entryParser = openrouterClient
			docMeta = openrouterClient
			summarizer = openrouterClient
		}
		if shirtyProvider != nil {
			class = shirtyProvider
			entryParser = shirtyProvider
			docMeta = shirtyProvider
			summarizer = shirtyProvider
		}

		byID := make(map[int]string, len(split))
		entryIDs := make([]int, 0, len(split))
		for _, entry := range split {
			byID[entry.ID] = entry.Text
			entryIDs = append(entryIDs, entry.ID)
		}
		if cmd.Flags().Changed(FlagEntry) {
			id, _ := cmd.Flags().GetInt(FlagEntry)
			if _, ok := byID[id]; !ok {
				return fmt.Errorf("entry %d not found in input (%d entries)", id, len(split))
			}
			entryIDs = []int{id}
		}

		cfg := &lookup.EntryConfig{
			ElsevierClient: elsevierClient,
			CrossrefClient: crossref.NewClient(),
		}

		run, err := analysisrunner.Run(cmd.Context(), analysisrunner.Config{
			EntryIDs: entryIDs,
			Workers:  textWorkers,
			Extract: func(id int) (string, error) {
				return byID[id], nil
			},
			Lookup: func(text string) (*lookup.Result, error) {
				return lookup.Entry(text, "", class, docMeta, entryParser, cfg)
			},
			Summarize: func(result *lookup.Result) (analysisrunner.Summary, error) {
				if summarizer == nil {
					return analysisrunner.Summary{}, nil
				}
				mismatch, comment, err := summarizer.Summarize(result)
				return analysisrunner.Summary{Mismatch: mismatch, Comment: comment}, err
			},
		})
		if err != nil {
			return err
		}

		return renderRun(run, textFormat, textCarelessHideOK, cmd.Flags().Changed(FlagEntry))
	},
}

// readTextInput reads the bibliography text from the file argument, or from
// stdin when no argument (or "-") is given.
func readTextInput(cmd *cobra.Command, args []string) (string, error) {
	if len(args) == 0 || args[0] == "-" {
		data, err := io.ReadAll(cmd.InOrStdin())
		if err != nil {
			return "", fmt.Errorf("read stdin error: %w", err)
		}
		return string(data), nil
	}
	data, err := os.ReadFile(args[0])
	if err != nil {
		return "", fmt.Errorf("read input file error: %w", err)
	}
	return string(data), nil
}

func init() {
	textCmd.Flags().BoolVar(&textCarelessHideOK, FlagCarelessHideOK, false, "Hide entries whose summary explicitly says they look okay")
	textCmd.Flags().Int(FlagEntry, -1, "Analyze a single entry")
	textCmd.Flags().Var(newOutputFormatValue(&textFormat), FlagFormat, "Output format: text or json")
	textCmd.Flags().IntVar(&textWorkers, FlagWorkers, analysisrunner.DefaultWorkers, "Number of bibliography workers")
}
