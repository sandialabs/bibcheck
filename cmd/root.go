// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	analysisrunner "github.com/sandialabs/bibcheck/analysis"
	"github.com/sandialabs/bibcheck/config"
	"github.com/sandialabs/bibcheck/crossref"
	"github.com/sandialabs/bibcheck/documents"
	"github.com/sandialabs/bibcheck/elsevier"
	"github.com/sandialabs/bibcheck/entries"
	"github.com/sandialabs/bibcheck/lookup"
	"github.com/sandialabs/bibcheck/openrouter"
	"github.com/sandialabs/bibcheck/shirty"
	"github.com/sandialabs/bibcheck/version"
)

var (
	carelessHideOK bool
	format         outputFormat
	pipeline       string
	workers        int
)

const (
	FlagCarelessHideOK string = "careless-hide-ok"
	FlagEntry          string = "entry"
	FlagFormat         string = "format"
	FlagPipeline       string = "pipeline"
	FlagWorkers        string = "workers"
)

type outputFormat string

type summarizer interface {
	Summarize(*lookup.Result) (bool, string, error)
}

const (
	outputFormatJSON outputFormat = "json"
	outputFormatText outputFormat = "text"
)

var rootCmd = &cobra.Command{
	Use:   "bibcheck <pdf-file>",
	Short: "Check bibliography entries in a PDF file",
	Long: `bibliograph-checker ` + version.String() + ` (` + version.GitSha() + `)
A tool that analyzes bibliography entries in PDF files and verifies their existence.`,
	Args:          cobra.ExactArgs(1),
	SilenceErrors: true,
	SilenceUsage:  true,
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
				settings.ShirtyBaseURL,
				shirty.WithModel(settings.ShirtyModel),
			)
		}

		var elsevierClient *elsevier.Client
		if settings.ElsevierAPIKey != "" {
			elsevierClient = elsevier.NewClient(settings.ElsevierAPIKey)
		}

		var bibliography *documents.Bibliography
		var err error

		if shirtyProvider != nil {
			bibliography, err = shirtyProvider.PrepareBibliography(pdfPath)
			if err != nil {
				return fmt.Errorf("prepare bibliography error: %w", err)
			}
		} else if openrouterClient != nil {
			bibliography, err = openrouterClient.PrepareBibliography(pdfPath)
			if err != nil {
				return fmt.Errorf("prepare bibliography error: %w", err)
			}
		}

		if cmd.Flags().Changed(FlagEntry) {
			entryStart, _ = cmd.Flags().GetInt(FlagEntry)
			entryCount = 1
		} else {

			// Get citation counts
			if bibliography != nil && openrouterClient != nil && shirtyProvider == nil {
				fmt.Println("Counting bibliography entries...")
				entryCount, err = openrouterClient.NumBibliographyEntries(bibliography)
				if err != nil {
					return fmt.Errorf("bibliography size error: %w", err)
				}
				fmt.Printf("Found %d bibliographic entries\n", entryCount)
			} else if bibliography != nil {
				entryCount, err = shirtyProvider.NumBibEntries(bibliography)
				if err != nil {
					return fmt.Errorf("bibliography size error: %w", err)
				}

			} else {
				return fmt.Errorf("need shirty or openrouter config")
			}
		}

		var class entries.Classifier
		var entryParser entries.Parser
		var docBibliographyExtract documents.EntryFromBibliographyExtractor
		var docMeta documents.MetaExtractor

		// default to using openrouter, if available
		if openrouterClient != nil {
			class = openrouterClient
			entryParser = openrouterClient
			docBibliographyExtract = openrouterClient
			docMeta = openrouterClient
		}

		// use shirty where possible
		if shirtyProvider != nil {
			class = shirtyProvider
			entryParser = shirtyProvider
			docBibliographyExtract = shirtyProvider
			docMeta = shirtyProvider
		}

		cfg := &lookup.EntryConfig{
			ElsevierClient: elsevierClient,
			CrossrefClient: crossref.NewClient(),
		}

		var summarizer summarizer
		if openrouterClient != nil {
			summarizer = openrouterClient
		}
		if shirtyProvider != nil {
			summarizer = shirtyProvider
		}

		entryIDs := make([]int, entryCount)
		for i := range entryIDs {
			entryIDs[i] = entryStart + i
		}
		if docBibliographyExtract == nil || bibliography == nil {
			return fmt.Errorf("requires something that can extract a bib entry from a pdf")
		}

		run, err := analysisrunner.Run(cmd.Context(), analysisrunner.Config{
			EntryIDs: entryIDs,
			Workers:  workers,
			Extract: func(id int) (string, error) {
				return docBibliographyExtract.EntryFromBibliography(bibliography, id)
			},
			Lookup: func(text string) (*lookup.Result, error) {
				return lookup.Entry(text, pipeline, class, docMeta, entryParser, cfg)
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

		views := []entryView{}
		singleEntry := cmd.Flags().Changed(FlagEntry)
		for _, entry := range run.Entries {
			if entry.Result == nil {
				if entry.ExtractionError != nil {
					log.Printf("entry %d extraction error: %v", entry.ID, entry.ExtractionError)
				} else if entry.LookupError != nil {
					log.Printf("entry %d analysis error: %v", entry.ID, entry.LookupError)
				}
				continue
			}
			outcome := summaryOutcome{
				mismatch: entry.Summary.Mismatch,
				comment:  entry.Summary.Comment,
				err:      entry.SummaryError,
			}
			views = append(views, buildEntryView(entry.ID, entry.Result, outcome))
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
	rootCmd.PersistentFlags().String("shirty-model", config.DefaultShirtyModel, "Default Shirty model")
	if err := config.BindFlags(rootCmd.PersistentFlags()); err != nil {
		panic(err)
	}
	rootCmd.Flags().BoolVar(&carelessHideOK, FlagCarelessHideOK, false, "Hide entries whose summary explicitly says they look okay")
	rootCmd.Flags().Int(FlagEntry, -1, "Analyze a single entry")
	rootCmd.Flags().Var(newOutputFormatValue(&format), FlagFormat, "Output format: text or json")
	rootCmd.Flags().StringVar(&pipeline, FlagPipeline, "auto", "Analysis pipeline to use")
	rootCmd.Flags().IntVar(&workers, FlagWorkers, analysisrunner.DefaultWorkers, "Number of bibliography workers")

	rootCmd.AddCommand(bibCmd)
	rootCmd.AddCommand(doiCmd)
	rootCmd.AddCommand(entryCmd)
	rootCmd.AddCommand(listEntriesCmd)
	rootCmd.AddCommand(textractCmd)
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
