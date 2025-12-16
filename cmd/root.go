// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/sandialabs/bibcheck/analyze"
	"github.com/sandialabs/bibcheck/documents"
	"github.com/sandialabs/bibcheck/entries"
	"github.com/sandialabs/bibcheck/openrouter"
	"github.com/sandialabs/bibcheck/search"
	"github.com/sandialabs/bibcheck/shirty"
	"github.com/sandialabs/bibcheck/version"
	"github.com/spf13/cobra"
)

var (
	pipeline string
)

const (
	FlagEntry    string = "entry"
	FlagPipeline string = "pipeline"
)

var rootCmd = &cobra.Command{
	Use:   "bibcheck <pdf-file>",
	Short: "Check bibliography entries in a PDF file",
	Long: `bibliograph-checker ` + version.String() + ` (` + version.GitSha() + `)
A tool that analyzes bibliography entries in PDF files and verifies their existence.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		pdfPath := args[0]
		entryStart := 1
		var entryCount int

		// set up clients depending on config
		var openrouterClient *openrouter.Client
		var shirtyProvider *shirty.Workflow
		if openrouterApiKey != "" && openrouterBaseUrl != "" {
			openrouterClient = openrouter.NewClient(openrouterApiKey)
		}
		if shirtyApiKey != "" && shirtyBaseUrl != "" {
			shirtyProvider = shirty.NewWorkflow(
				shirtyApiKey,
				shirty.WithBaseUrl(shirtyBaseUrl))
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
				pdfEncoded, err = analyze.Encode(pdfPath)
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

		var comp entries.Comparer
		var class entries.Classifier
		var entryParser entries.Parser
		var docRawExtract documents.EntryFromRawExtractor
		var docTextExtract documents.EntryFromTextExtractor
		var docMeta documents.MetaExtractor
		var searcher search.Searcher

		// default to using openrouter, if available
		if openrouterClient != nil {
			comp = openrouterClient
			class = openrouterClient
			entryParser = openrouterClient
			docRawExtract = openrouterClient
			docMeta = openrouterClient
			searcher = openrouterClient
		}

		// use shirty where possible
		if shirtyProvider != nil {
			comp = shirtyProvider
			class = shirtyProvider
			entryParser = shirtyProvider
			docTextExtract = shirtyProvider
			docMeta = shirtyProvider

			if pdfText == "" {
				textractResp, err := shirtyProvider.Textract(pdfPath)
				pdfText = textractResp.Text
				if err != nil {
					log.Fatalf("textract error: %v", err)
				}
			}

		}

		for i := entryStart; i < entryStart+entryCount; i++ {
			var ea *analyze.EntryAnalysis
			if docRawExtract != nil {
				ea, err = analyze.EntryFromBase64(pdfEncoded, i, pipeline, comp, class, docRawExtract, docMeta, entryParser, searcher)
			} else if docTextExtract != nil {
				ea, err = analyze.EntryFromText(pdfText, i, pipeline, comp, class, docTextExtract, docMeta, entryParser, searcher)
			} else {
				log.Fatalf("requires something that can extract a bib entry from a pdf")
			}

			if err != nil {
				log.Printf("entry analysis error: %v", err)
				continue
			}
			analyze.Print(ea)
		}

	},
}

func init() {
	// don't include the `completion` subcommand
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.PersistentFlags().StringVar(&openrouterApiKey, "openrouter-api-key", "", "OpenRouter API key")
	rootCmd.PersistentFlags().StringVar(&openrouterBaseUrl,
		"openrouter-base-url",
		"https://openrouter.ai/api/v1",
		"Openrouter-compatible API url",
	)
	rootCmd.PersistentFlags().StringVar(&shirtyApiKey, "shirty-api-key", "", "shirty.sandia.gov API key")
	rootCmd.PersistentFlags().StringVar(&shirtyBaseUrl,
		"shirty-base-url",
		"https://shirty.sandia.gov/api/v1",
		"Shirty base URL",
	)
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
