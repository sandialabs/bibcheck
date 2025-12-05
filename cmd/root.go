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

		var elsevierClient *elsevier.Client
		if elsevierApiKey != "" {
			elsevierClient = elsevier.NewClient(elsevierApiKey)
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

		// default to using openrouter, if available
		if openrouterClient != nil {
			comp = openrouterClient
			class = openrouterClient
			entryParser = openrouterClient
			docRawExtract = openrouterClient
			docMeta = openrouterClient
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

		cfg := &analyze.EntryConfig{
			ElsevierClient: elsevierClient,
		}

		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)
		t.AppendHeader(table.Row{"ORIG", "onl.", "Xref", "Els.", "Arxiv", "doi", "OSTI"})

		for i := entryStart; i < entryStart+entryCount; i++ {
			var ea *analyze.EntryAnalysis
			if docRawExtract != nil {
				ea, err = analyze.EntryFromBase64(pdfEncoded, i, pipeline, comp, class, docRawExtract, docMeta, entryParser, cfg)
			} else if docTextExtract != nil {
				ea, err = analyze.EntryFromText(pdfText, i, pipeline, comp, class, docTextExtract, docMeta, entryParser, cfg)
			} else {
				log.Fatalf("requires something that can extract a bib entry from a pdf")
			}

			if err != nil {
				log.Printf("entry analysis error: %v", err)
				continue
			}
			analyze.Print(ea)

			// add original entry to row
			WrapSoftLimit := 40
			row := []any{
				text.WrapSoft(ea.Text, WrapSoftLimit),
			}

			red := func(err error) string {
				return text.WrapSoft(
					text.FgRed.Sprintf("%v", err),
					WrapSoftLimit,
				)
			}

			if ea.Online.Metadata != nil {
				row = append(row, text.WrapSoft(ea.Online.Metadata.ToString(), WrapSoftLimit))
			} else if ea.Online.Error != nil {
				row = append(row, red(ea.Online.Error))
			} else {
				row = append(row, "")
			}
			if ea.Crossref.Work != nil {
				row = append(row, text.WrapSoft(ea.Crossref.Work.ToString(), WrapSoftLimit))
			} else if ea.Crossref.Error != nil {
				row = append(row, red(ea.Crossref.Error))
			} else {
				row = append(row, "")
			}
			if ea.Elsevier.Result != nil {
				row = append(row, text.WrapSoft(ea.Elsevier.Result.ToString(), WrapSoftLimit))
			} else if ea.Elsevier.Error != nil {
				row = append(row, red(ea.Elsevier.Error))
			} else {
				row = append(row, "")
			}
			if ea.Arxiv.Entry != nil {
				row = append(row, text.WrapSoft(ea.Arxiv.Entry.ToString(), WrapSoftLimit))
			} else if ea.Arxiv.Error != nil {
				row = append(row, red(ea.Arxiv.Error))
			} else {
				row = append(row, "")
			}
			if ea.DOIOrg.Found {
				row = append(row, text.WrapSoft("exists", WrapSoftLimit))
			} else if ea.DOIOrg.Error != nil {
				row = append(row, red(ea.DOIOrg.Error))
			} else {
				row = append(row, "")
			}
			if ea.OSTI.Record != nil {
				row = append(row, text.WrapSoft(ea.OSTI.Record.ToString(), WrapSoftLimit))
			} else if ea.OSTI.Error != nil {
				row = append(row, red(ea.OSTI.Error))
			} else {
				row = append(row, "")
			}
			t.AppendRow(row)
		}

		t.Render()
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
