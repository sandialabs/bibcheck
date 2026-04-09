// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/sandialabs/bibcheck/config"
	"github.com/sandialabs/bibcheck/version"
)

var (
	carelessHideOK bool
	format         outputFormat
	pipeline       string
)

const (
	FlagCarelessHideOK string = "careless-hide-ok"
	FlagEntry          string = "entry"
	FlagFormat         string = "format"
	FlagPipeline       string = "pipeline"
)

type outputFormat string

const (
	outputFormatJSON outputFormat = "json"
	outputFormatText outputFormat = "text"
)

var rootCmd = &cobra.Command{
	Use:   "bibcheck <pdf-file>",
	Short: "Check bibliography entries in a PDF file",
	Long: `bibliograph-checker ` + version.String() + ` (` + version.GitSha() + `)
A tool that analyzes bibliography entries in PDF files and verifies their existence.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validateOutputFormat(format); err != nil {
			return err
		}

		pdfPath := args[0]
		settings := config.Runtime()
		entryStart := 1
		var entryCount int
		analyzer, err := newAnalyzer(settings)
		if err != nil {
			return err
		}
		var prepared *preparedDocument

		if cmd.Flags().Changed(FlagEntry) {
			entryStart, _ = cmd.Flags().GetInt(FlagEntry)
			entryCount = 1
			prepared, err = analyzer.prepareDocument(pdfPath, false)
		} else {
			prepared, err = analyzer.prepareDocument(pdfPath, true)
			entryCount = prepared.entryCount
			fmt.Printf("Found %d bibliographic entries\n", entryCount)
		}
		if err != nil {
			return err
		}

		views := []entryView{}
		singleEntry := cmd.Flags().Changed(FlagEntry)

		for i := entryStart; i < entryStart+entryCount; i++ {
			view, err := analyzer.analyzeEntry(prepared, i)
			if err != nil {
				log.Printf("entry analysis error: %v", err)
				continue
			}
			views = append(views, view)
		}

		doc := buildDocumentView(views, carelessHideOK)
		switch format {
		case outputFormatText:
			fmt.Fprint(os.Stdout, renderDocument(doc, views, carelessHideOK, singleEntry))
		case outputFormatJSON:
			rendered, err := renderJSONDocument(doc, views, carelessHideOK, singleEntry)
			if err != nil {
				return fmt.Errorf("render json output: %w", err)
			}
			fmt.Fprint(os.Stdout, rendered)
		default:
			return fmt.Errorf("unsupported output format %q", format)
		}
		return nil
	},
}

func init() {
	// don't include the `completion` subcommand
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.PersistentFlags().String("elsevier-api-key", "", "Elsevier API key")
	rootCmd.PersistentFlags().String("openai-audit-dir", "", "Directory for OpenAI API audit logs")
	rootCmd.PersistentFlags().Bool("openai-audit-enabled", true, "Enable OpenAI API audit logging")
	rootCmd.PersistentFlags().String("openrouter-api-key", "", "OpenRouter API key")
	rootCmd.PersistentFlags().String("openrouter-base-url", config.DefaultOpenRouterBaseURL, "Openrouter-compatible API url")
	rootCmd.PersistentFlags().String("shirty-api-key", "", "shirty.sandia.gov API key")
	rootCmd.PersistentFlags().String("shirty-base-url", config.DefaultShirtyBaseURL, "Shirty base URL")
	if err := config.BindFlags(rootCmd.PersistentFlags()); err != nil {
		panic(err)
	}
	rootCmd.Flags().BoolVar(&carelessHideOK, FlagCarelessHideOK, false, "Hide entries whose summary explicitly says they look okay")
	rootCmd.Flags().Int(FlagEntry, -1, "Analyze a single entry")
	rootCmd.Flags().Var(newOutputFormatValue(&format), FlagFormat, "Output format: text or json")
	rootCmd.Flags().StringVar(&pipeline, FlagPipeline, "auto", "Analysis pipeline to use")

	rootCmd.AddCommand(bibCmd)
	rootCmd.AddCommand(doiCmd)
	rootCmd.AddCommand(entryCmd)
	rootCmd.AddCommand(evalCmd)
	rootCmd.AddCommand(listEntriesCmd)
	rootCmd.AddCommand(textractCmd)
	rootCmd.AddCommand(serveCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateOutputFormat(format outputFormat) error {
	switch format {
	case outputFormatText, outputFormatJSON:
		return nil
	default:
		return fmt.Errorf("invalid --format %q (supported: text, json)", format)
	}
}

type outputFormatValue struct {
	target *outputFormat
}

func newOutputFormatValue(target *outputFormat) *outputFormatValue {
	*target = outputFormatText
	return &outputFormatValue{target: target}
}

func (v *outputFormatValue) String() string {
	if v == nil || v.target == nil {
		return string(outputFormatText)
	}
	return string(*v.target)
}

func (v *outputFormatValue) Set(value string) error {
	format := outputFormat(value)
	if err := validateOutputFormat(format); err != nil {
		return err
	}
	*v.target = format
	return nil
}

func (v *outputFormatValue) Type() string {
	return "outputFormat"
}
