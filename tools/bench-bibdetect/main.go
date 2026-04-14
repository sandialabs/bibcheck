// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/sandialabs/bibcheck/config"
	"github.com/sandialabs/bibcheck/openrouter"
)

type manifest struct {
	Runs []runConfig `json:"runs"`
}

type runConfig struct {
	Name             string `json:"name"`
	Model            string `json:"model"`
	PromptFile       string `json:"prompt_file"`
	PDFEngine        string `json:"pdf_engine"`
	ReasoningEnabled *bool  `json:"reasoning_enabled"`
}

type truthDocument struct {
	StartPage int `json:"start_page"`
	EndPage   int `json:"end_page"`
}

type datasetDocument struct {
	Name  string
	PDF   string
	Truth truthDocument
}

type documentMetrics struct {
	Document        string
	PageCount       int
	TP              int
	TN              int
	FP              int
	FN              int
	ExactRangeMatch bool
	PredictedStart  int
	PredictedEnd    int
	ExpectedStart   int
	ExpectedEnd     int
	Latency         time.Duration
	Cost            float64
	Err             error
}

type runMetrics struct {
	Name           string
	Model          string
	PromptFile     string
	Documents      int
	Failures       int
	Pages          int
	TP             int
	TN             int
	FP             int
	FN             int
	ExactRangeHits int
	Latency        time.Duration
	Cost           float64
	DocResults     []documentMetrics
}

func main() {
	manifestPath := flag.String("manifest", "", "Path to benchmark manifest JSON")
	datasetDir := flag.String("dataset", "", "Directory with same-basename PDF and truth JSON files")
	openRouterAPIKey := flag.String("openrouter-api-key", "", "OpenRouter API key")
	flag.Parse()

	if *manifestPath == "" || *datasetDir == "" {
		fmt.Fprintln(os.Stderr, "usage: bench-bibdetect --manifest <path> --dataset <dir> [--openrouter-api-key ...] [--openrouter-base-url ...]")
		os.Exit(2)
	}

	settings := config.Runtime()
	apiKey := chooseFirstNonEmpty(*openRouterAPIKey, settings.OpenRouterAPIKey)
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "missing OpenRouter API key")
		os.Exit(1)
	}

	m, err := loadManifest(*manifestPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load manifest: %v\n", err)
		os.Exit(1)
	}

	docs, err := discoverDataset(*datasetDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load dataset: %v\n", err)
		os.Exit(1)
	}

	client := openrouter.NewClient(apiKey)
	results, err := runBenchmark(client, m, docs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "run benchmark: %v\n", err)
		os.Exit(1)
	}

	renderResults(os.Stdout, results)
}

func chooseFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func loadManifest(path string) (*manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var m manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest JSON: %w", err)
	}
	if len(m.Runs) == 0 {
		return nil, errors.New("manifest must define at least one run")
	}

	for i := range m.Runs {
		run := &m.Runs[i]
		if run.Name == "" {
			return nil, fmt.Errorf("run %d missing name", i)
		}
		if run.Model == "" {
			return nil, fmt.Errorf("run %q missing model", run.Name)
		}
		if run.PromptFile == "" {
			return nil, fmt.Errorf("run %q missing prompt_file", run.Name)
		}
		if _, err := parsePDFEngine(run.PDFEngine); err != nil {
			return nil, fmt.Errorf("run %q invalid pdf_engine: %w", run.Name, err)
		}
	}

	return &m, nil
}

func discoverDataset(dir string) ([]datasetDocument, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	docs := make([]datasetDocument, 0)
	for _, entry := range entries {
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".pdf" {
			continue
		}

		base := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		pdfPath := filepath.Join(dir, entry.Name())
		jsonPath := filepath.Join(dir, base+".json")

		truth, err := loadTruthDocument(jsonPath)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", jsonPath, err)
		}

		docs = append(docs, datasetDocument{
			Name:  base,
			PDF:   pdfPath,
			Truth: truth,
		})
	}

	if len(docs) == 0 {
		return nil, fmt.Errorf("no PDF fixtures found in %s", dir)
	}

	sort.Slice(docs, func(i, j int) bool {
		return docs[i].Name < docs[j].Name
	})

	return docs, nil
}

func loadTruthDocument(path string) (truthDocument, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return truthDocument{}, fmt.Errorf("missing truth file for fixture")
		}
		return truthDocument{}, err
	}

	var truth truthDocument
	if err := json.Unmarshal(data, &truth); err != nil {
		return truthDocument{}, fmt.Errorf("parse truth JSON: %w", err)
	}
	if truth.StartPage < 1 {
		return truthDocument{}, fmt.Errorf("invalid start_page %d", truth.StartPage)
	}
	if truth.EndPage < truth.StartPage {
		return truthDocument{}, fmt.Errorf("invalid page range %d-%d", truth.StartPage, truth.EndPage)
	}
	return truth, nil
}

func runBenchmark(client *openrouter.Client, m *manifest, docs []datasetDocument) ([]runMetrics, error) {
	results := make([]runMetrics, 0, len(m.Runs))
	for _, run := range m.Runs {
		prompt, err := os.ReadFile(run.PromptFile)
		if err != nil {
			return nil, fmt.Errorf("read prompt for run %q: %w", run.Name, err)
		}

		rm := runMetrics{
			Name:       run.Name,
			Model:      run.Model,
			PromptFile: run.PromptFile,
		}

		cfg := openrouter.BibliographyPageDetectorConfig{
			Model:            run.Model,
			Prompt:           string(prompt),
			ReasoningEnabled: run.ReasoningEnabled,
		}
		engine, err := parsePDFEngine(run.PDFEngine)
		if err != nil {
			return nil, fmt.Errorf("parse pdf_engine for run %q: %w", run.Name, err)
		}
		cfg.PDFEngine = engine

		for _, doc := range docs {
			rm.Documents++
			result := benchmarkDocument(client, cfg, doc)
			rm.DocResults = append(rm.DocResults, result)
			if result.Err != nil {
				rm.Failures++
				continue
			}

			rm.Pages += result.PageCount
			rm.TP += result.TP
			rm.TN += result.TN
			rm.FP += result.FP
			rm.FN += result.FN
			if result.ExactRangeMatch {
				rm.ExactRangeHits++
			}
			rm.Latency += result.Latency
			rm.Cost += result.Cost
		}

		results = append(results, rm)
	}

	return results, nil
}

func parsePDFEngine(value string) (*openrouter.PDFEngine, error) {
	if value == "" {
		return nil, nil
	}

	engine := openrouter.PDFEngine(value)
	switch engine {
	case openrouter.PDFEngineCloudflareAI, openrouter.PDFEngineMistralOCR, openrouter.PDFEngineNative:
		return &engine, nil
	default:
		return nil, fmt.Errorf("unsupported value %q", value)
	}
}

func benchmarkDocument(client *openrouter.Client, cfg openrouter.BibliographyPageDetectorConfig, doc datasetDocument) documentMetrics {
	pdf, err := os.ReadFile(doc.PDF)
	if err != nil {
		return documentMetrics{Document: doc.Name, Err: fmt.Errorf("read pdf: %w", err)}
	}

	predictions, err := client.DetectBibliographyPages(pdf, cfg)
	if err != nil {
		return documentMetrics{Document: doc.Name, Err: err}
	}

	pageCount := len(predictions)
	if doc.Truth.EndPage > pageCount {
		return documentMetrics{
			Document: doc.Name,
			Err:      fmt.Errorf("truth end_page %d exceeds page count %d", doc.Truth.EndPage, pageCount),
		}
	}

	expected := makeExpectedFlags(pageCount, doc.Truth)
	predicted := make([]bool, pageCount)
	latency := time.Duration(0)
	cost := 0.0
	for i, prediction := range predictions {
		predicted[i] = prediction.ContainsBibliography
		latency += prediction.Latency
		if prediction.Usage != nil && prediction.Usage.Cost != nil {
			cost += *prediction.Usage.Cost
		}
	}

	tp, tn, fp, fn := compareFlags(predicted, expected)
	predictedStart, predictedEnd, ok := pageRange(predicted)
	if !ok {
		predictedStart = 0
		predictedEnd = 0
	}

	return documentMetrics{
		Document:        doc.Name,
		PageCount:       pageCount,
		TP:              tp,
		TN:              tn,
		FP:              fp,
		FN:              fn,
		ExactRangeMatch: predictedStart == doc.Truth.StartPage && predictedEnd == doc.Truth.EndPage,
		PredictedStart:  predictedStart,
		PredictedEnd:    predictedEnd,
		ExpectedStart:   doc.Truth.StartPage,
		ExpectedEnd:     doc.Truth.EndPage,
		Latency:         latency,
		Cost:            cost,
	}
}

func makeExpectedFlags(pageCount int, truth truthDocument) []bool {
	flags := make([]bool, pageCount)
	for i := truth.StartPage; i <= truth.EndPage; i++ {
		flags[i-1] = true
	}
	return flags
}

func compareFlags(predicted, expected []bool) (tp, tn, fp, fn int) {
	for i := range predicted {
		switch {
		case predicted[i] && expected[i]:
			tp++
		case predicted[i] && !expected[i]:
			fp++
		case !predicted[i] && expected[i]:
			fn++
		default:
			tn++
		}
	}
	return tp, tn, fp, fn
}

func pageRange(flags []bool) (startPage, endPage int, ok bool) {
	for i, flag := range flags {
		if !flag {
			continue
		}
		if !ok {
			startPage = i + 1
			ok = true
		}
		endPage = i + 1
	}
	return startPage, endPage, ok
}

func renderResults(out io.Writer, results []runMetrics) {
	for i, run := range results {
		if i > 0 {
			fmt.Fprintln(out)
		}

		fmt.Fprintf(out, "Run: %s\n", run.Name)
		fmt.Fprintf(out, "Model: %s\n", run.Model)
		fmt.Fprintf(out, "Prompt: %s\n", run.PromptFile)

		tw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "Metric\tValue")
		fmt.Fprintf(tw, "Documents\t%d\n", run.Documents)
		fmt.Fprintf(tw, "Failures\t%d\n", run.Failures)
		fmt.Fprintf(tw, "Pages\t%d\n", run.Pages)
		fmt.Fprintf(tw, "Page Accuracy\t%.4f\n", ratio(run.TP+run.TN, run.Pages))
		fmt.Fprintf(tw, "Precision\t%.4f\n", ratio(run.TP, run.TP+run.FP))
		fmt.Fprintf(tw, "Recall\t%.4f\n", ratio(run.TP, run.TP+run.FN))
		fmt.Fprintf(tw, "F1\t%.4f\n", f1(run.TP, run.FP, run.FN))
		fmt.Fprintf(tw, "Exact Range Matches\t%d/%d\n", run.ExactRangeHits, run.Documents-run.Failures)
		fmt.Fprintf(tw, "Total Latency\t%s\n", run.Latency.Round(time.Millisecond))
		fmt.Fprintf(tw, "Avg Latency / Document\t%s\n", averageDuration(run.Latency, run.Documents-run.Failures))
		fmt.Fprintf(tw, "Avg Latency / Page\t%s\n", averageDuration(run.Latency, run.Pages))
		fmt.Fprintf(tw, "Total Cost\t$%.6f\n", run.Cost)
		fmt.Fprintf(tw, "Avg Cost / Document\t$%.6f\n", averageFloat(run.Cost, run.Documents-run.Failures))
		fmt.Fprintf(tw, "Avg Cost / Page\t$%.6f\n", averageFloat(run.Cost, run.Pages))
		_ = tw.Flush()

		for _, doc := range run.DocResults {
			switch {
			case doc.Err != nil:
				fmt.Fprintf(out, "  FAIL %s: %v\n", doc.Document, doc.Err)
			case !doc.ExactRangeMatch:
				fmt.Fprintf(out, "  MISMATCH %s: expected %d-%d predicted %d-%d\n",
					doc.Document, doc.ExpectedStart, doc.ExpectedEnd, doc.PredictedStart, doc.PredictedEnd)
			}
		}
	}
}

func ratio(num, den int) float64 {
	if den == 0 {
		return 0
	}
	return float64(num) / float64(den)
}

func f1(tp, fp, fn int) float64 {
	precision := ratio(tp, tp+fp)
	recall := ratio(tp, tp+fn)
	if precision == 0 || recall == 0 {
		return 0
	}
	return 2 * precision * recall / (precision + recall)
}

func averageFloat(total float64, count int) float64 {
	if count <= 0 {
		return 0
	}
	return total / float64(count)
}

func averageDuration(total time.Duration, count int) time.Duration {
	if count <= 0 {
		return 0
	}
	avg := float64(total) / float64(count)
	return time.Duration(math.Round(avg)).Round(time.Millisecond)
}
