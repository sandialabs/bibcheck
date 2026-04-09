// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package cmd

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/sandialabs/bibcheck/config"
	"github.com/sandialabs/bibcheck/documents"
	"github.com/sandialabs/bibcheck/elsevier"
	"github.com/sandialabs/bibcheck/entries"
	"github.com/sandialabs/bibcheck/lookup"
	"github.com/sandialabs/bibcheck/openrouter"
	"github.com/sandialabs/bibcheck/shirty"
	"github.com/sandialabs/bibcheck/summary"
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

		// set up clients depending on config
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
				shirty.WithBaseUrl(settings.ShirtyBaseURL))
		}

		var elsevierClient *elsevier.Client
		if settings.ElsevierAPIKey != "" {
			elsevierClient = elsevier.NewClient(settings.ElsevierAPIKey)
		}

		// different representations of the file
		var pdfEncoded string
		var pdfText string
		var err error

		if cmd.Flags().Changed(FlagEntry) {
			entryStart, _ = cmd.Flags().GetInt(FlagEntry)
			entryCount = 1
		} else {

			// Get citation counts
			if openrouterClient != nil {
				pdfEncoded, err = lookup.Encode(pdfPath)
				if err != nil {
					return fmt.Errorf("pdf encode error: %w", err)
				}
				fmt.Println("Counting bibliography entries...")
				entryCount, err = openrouterClient.NumEntries(pdfEncoded)
				if err != nil {
					return fmt.Errorf("bibliography size error: %w", err)
				}
				fmt.Printf("Found %d bibliographic entries\n", entryCount)
			} else if shirtyProvider != nil {
				textractResp, err := shirtyProvider.Textract(pdfPath)
				pdfText = textractResp.Text
				if err != nil {
					return fmt.Errorf("textract error: %w", err)
				}
				entryCount, err = shirtyProvider.NumBibEntries(pdfText)
				if err != nil {
					return fmt.Errorf("bibliography size error: %w", err)
				}

			} else {
				return errors.New("need shirty or openrouter config")
			}
		}

		var class entries.Classifier
		var entryParser entries.Parser
		var docRawExtract documents.EntryFromRawExtractor
		var docTextExtract documents.EntryFromTextExtractor
		var docMeta documents.MetaExtractor

		// default to using openrouter, if available
		if openrouterClient != nil {
			class = openrouterClient
			entryParser = openrouterClient
			docRawExtract = openrouterClient
			docMeta = openrouterClient
		}

		// use shirty where possible
		if shirtyProvider != nil {
			class = shirtyProvider
			entryParser = shirtyProvider
			docTextExtract = shirtyProvider
			docMeta = shirtyProvider

			if pdfText == "" {
				textractResp, err := shirtyProvider.Textract(pdfPath)
				if err != nil {
					return fmt.Errorf("textract error: %w", err)
				}
				pdfText = textractResp.Text
			}

		}

		cfg := &lookup.EntryConfig{
			ElsevierClient: elsevierClient,
		}

		var summarizer *summary.ShirtySummarizer

		if settings.ShirtyAPIKey != "" {
			summarizer = summary.NewShirtySummarizer(
				shirty.NewWorkflow(
					settings.ShirtyAPIKey,
					shirty.WithBaseUrl(settings.ShirtyBaseURL),
				),
			)
		}

		views := []entryView{}
		singleEntry := cmd.Flags().Changed(FlagEntry)

		for i := entryStart; i < entryStart+entryCount; i++ {
			var lr *lookup.Result
			if docRawExtract != nil {
				lr, err = lookup.EntryFromBase64(pdfEncoded, i, pipeline, class, docRawExtract, docMeta, entryParser, cfg)
			} else if docTextExtract != nil {
				lr, err = lookup.EntryFromText(pdfText, i, pipeline, class, docTextExtract, docMeta, entryParser, cfg)
			} else {
				return errors.New("requires something that can extract a bib entry from a pdf")
			}

			if err != nil {
				log.Printf("entry analysis error: %v", err)
				continue
			}

			outcome := summaryOutcome{}
			if summarizer != nil {
				mismatch, comment, err := summarizer.Summarize(lr)
				outcome.mismatch = mismatch
				outcome.comment = comment
				outcome.err = err
				if err != nil {
					log.Printf("summarizer error: %v", err)
				}
			}

			views = append(views, buildEntryView(i, lr, outcome))
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
