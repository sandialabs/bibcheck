// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"

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
				pdfText = textractResp.Text
				if err != nil {
					log.Fatalf("textract error: %v", err)
				}
			}

		}

		cfg := &lookup.EntryConfig{
			ElsevierClient: elsevierClient,
		}

		var summarizer *summary.ShirtySummarizer

		if shirtyApiKey != "" {
			summarizer = summary.NewShirtySummarizer(
				shirty.NewWorkflow(shirtyApiKey),
			)
		}

		t := table.NewWriter()
		t.SetOutputMirror(os.Stdout)

		header := table.Row{"#", "ORIG", "onl.", "Xref", "Els.", "Arx.", "DOI", "OSTI"}
		if summarizer != nil {
			header = append(header, "ANALYSIS")
		}

		t.AppendHeader(header)

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

			// add original entry to row
			WrapSoftLimit := 40
			row := []any{
				i,
				text.WrapSoft(lr.Text, WrapSoftLimit),
			}

			red := func(err error) string {
				return text.WrapSoft(
					text.FgRed.Sprintf("%v", err),
					WrapSoftLimit,
				)
			}

			yellow := func(s string) string {
				return text.WrapSoft(
					text.FgYellow.Sprintf("%s", s),
					WrapSoftLimit,
				)
			}

			green := func(s string) string {
				return text.WrapSoft(
					text.FgGreen.Sprintf("%s", s),
					WrapSoftLimit,
				)
			}

			if lr.Online.Metadata != nil {
				row = append(row, text.WrapSoft(lr.Online.Metadata.ToString(), WrapSoftLimit))
			} else if lr.Online.Error != nil {
				row = append(row, red(lr.Online.Error))
			} else {
				row = append(row, "")
			}
			if lr.Crossref.Work != nil {
				row = append(row, text.WrapSoft(lr.Crossref.Work.ToString(), WrapSoftLimit))
			} else if lr.Crossref.Error != nil {
				row = append(row, red(lr.Crossref.Error))
			} else {
				row = append(row, "")
			}
			if lr.Elsevier.Result != nil {
				row = append(row, text.WrapSoft(lr.Elsevier.Result.ToString(), WrapSoftLimit))
			} else if lr.Elsevier.Error != nil {
				row = append(row, red(lr.Elsevier.Error))
			} else {
				row = append(row, "")
			}
			if lr.Arxiv.Entry != nil {
				row = append(row, text.WrapSoft(lr.Arxiv.Entry.ToString(), WrapSoftLimit))
			} else if lr.Arxiv.Error != nil {
				row = append(row, red(lr.Arxiv.Error))
			} else {
				row = append(row, "")
			}
			if lr.DOIOrg.Found {
				row = append(row, text.WrapSoft("exists", WrapSoftLimit))
			} else if lr.DOIOrg.Error != nil {
				row = append(row, red(lr.DOIOrg.Error))
			} else {
				row = append(row, "")
			}
			if lr.OSTI.Record != nil {
				row = append(row, text.WrapSoft(lr.OSTI.Record.ToString(), WrapSoftLimit))
			} else if lr.OSTI.Error != nil {
				row = append(row, red(lr.OSTI.Error))
			} else {
				row = append(row, "")
			}

			if summarizer != nil {
				mismatch, comment, err := summarizer.Summarize(lr)
				log.Println(mismatch, comment, err)
				if err != nil {
					log.Printf("summarizer error: %v", err)
					row = append(row, red(err))
				} else if mismatch {
					row = append(row, yellow(comment))
				} else {
					row = append(row, green("OK"))
				}
			}

			t.AppendRow(row)
			t.AppendSeparator()
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
