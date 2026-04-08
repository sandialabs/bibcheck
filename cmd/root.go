// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package cmd

import (
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
	pipeline       string
)

const (
	FlagCarelessHideOK string = "careless-hide-ok"
	FlagEntry          string = "entry"
	FlagPipeline       string = "pipeline"
)

var rootCmd = &cobra.Command{
	Use:   "bibcheck <pdf-file>",
	Short: "Check bibliography entries in a PDF file",
	Long: `bibliograph-checker ` + version.String() + ` (` + version.GitSha() + `)
A tool that analyzes bibliography entries in PDF files and verifies their existence.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
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
					log.Fatalf("pdf encode error: %v", err)
				}
				fmt.Println("Counting bibliography entries...")
				entryCount, err = openrouterClient.NumEntries(pdfEncoded)
				if err != nil {
					log.Fatalf("Bibliography size error: %v\n", err)
				}
				fmt.Printf("Found %d bibliographic entries\n", entryCount)
			} else if shirtyProvider != nil {
				textractResp, err := shirtyProvider.Textract(pdfPath)
				pdfText = textractResp.Text
				if err != nil {
					log.Fatalf("textract error: %v", err)
				}
				entryCount, err = shirtyProvider.NumBibEntries(pdfText)
				if err != nil {
					log.Fatalf("bibliography size error: %v\n", err)
				}

			} else {
				log.Fatalf("need shirty or openrouter config")
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
					log.Fatalf("textract error: %v", err)
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
				log.Fatalf("requires something that can extract a bib entry from a pdf")
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
		fmt.Fprint(os.Stdout, renderDocument(doc, views, carelessHideOK, singleEntry))
	},
}

func init() {
	// don't include the `completion` subcommand
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.PersistentFlags().String("elsevier-api-key", "", "Elsevier API key")
	rootCmd.PersistentFlags().String("openrouter-api-key", "", "OpenRouter API key")
	rootCmd.PersistentFlags().String("openrouter-base-url", config.DefaultOpenRouterBaseURL, "Openrouter-compatible API url")
	rootCmd.PersistentFlags().String("shirty-api-key", "", "shirty.sandia.gov API key")
	rootCmd.PersistentFlags().String("shirty-base-url", config.DefaultShirtyBaseURL, "Shirty base URL")
	if err := config.BindFlags(rootCmd.PersistentFlags()); err != nil {
		panic(err)
	}
	rootCmd.Flags().BoolVar(&carelessHideOK, FlagCarelessHideOK, false, "Hide entries whose summary explicitly says they look okay")
	rootCmd.Flags().Int(FlagEntry, -1, "Analyze a single entry")
	rootCmd.Flags().StringVar(&pipeline, FlagPipeline, "auto", "Analysis pipeline to use")

	rootCmd.AddCommand(bibCmd)
	rootCmd.AddCommand(doiCmd)
	rootCmd.AddCommand(entryCmd)
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
