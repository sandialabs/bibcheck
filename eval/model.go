// Copyright 2026 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	DefaultWorkspaceDir = "evaldata"
	CorpusFileName      = "corpus.json"
	FormatVersion       = 1
	RunsDirName         = "runs"
	ResultsDirName      = "results"
	ReportsDirName      = "reports"
	AnnotationsDirName  = "annotations"
)

type Workspace struct {
	Root string
}

type RunStatus string

const (
	RunStatusPending RunStatus = "pending"
	RunStatusRunning RunStatus = "running"
	RunStatusDone    RunStatus = "done"
	RunStatusError   RunStatus = "error"
)

type Corpus struct {
	FormatVersion int        `json:"format_version"`
	GeneratedAt   time.Time  `json:"generated_at"`
	CorpusRoot    string     `json:"corpus_root"`
	Venues        []Venue    `json:"venues"`
	Papers        []PaperRef `json:"papers"`
}

type Venue struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	RelativePath string     `json:"relative_path"`
	Papers       []PaperRef `json:"papers"`
}

type PaperRef struct {
	ID           string `json:"id"`
	VenueID      string `json:"venue_id"`
	RelativePath string `json:"relative_path"`
	FileName     string `json:"file_name"`
}

type Run struct {
	FormatVersion int              `json:"format_version"`
	RunID         string           `json:"run_id"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
	GitSHA        string           `json:"git_sha"`
	Pipeline      string           `json:"pipeline"`
	CorpusRoot    string           `json:"corpus_root"`
	Providers     RunProviders     `json:"providers"`
	StatusSummary RunStatusSummary `json:"status_summary"`
	Papers        []RunPaper       `json:"papers"`
}

type RunProviders struct {
	ShirtyConfigured     bool `json:"shirty_configured"`
	OpenRouterConfigured bool `json:"openrouter_configured"`
	ElsevierConfigured   bool `json:"elsevier_configured"`
}

type RunStatusSummary struct {
	Pending int `json:"pending"`
	Running int `json:"running"`
	Done    int `json:"done"`
	Error   int `json:"error"`
}

type RunPaper struct {
	PaperID      string     `json:"paper_id"`
	VenueID      string     `json:"venue_id"`
	RelativePath string     `json:"relative_path"`
	Status       RunStatus  `json:"status"`
	ResultPath   string     `json:"result_path"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	Error        string     `json:"error,omitempty"`
}

type PaperResult struct {
	FormatVersion int           `json:"format_version"`
	RunID         string        `json:"run_id"`
	PaperID       string        `json:"paper_id"`
	VenueID       string        `json:"venue_id"`
	PDFPath       string        `json:"pdf_path"`
	CompletedAt   time.Time     `json:"completed_at"`
	TotalEntries  int           `json:"total_entries"`
	Entries       []EntryResult `json:"entries"`
}

type EntryResult struct {
	PaperID            string         `json:"paper_id"`
	VenueID            string         `json:"venue_id"`
	EntryNumber        int            `json:"entry_number"`
	ExtractedEntryText string         `json:"extracted_entry_text"`
	FinalDecision      string         `json:"final_decision"`
	PrimarySource      string         `json:"primary_matched_source,omitempty"`
	Sources            []SourceResult `json:"sources"`
	SummaryState       string         `json:"summary_state"`
	SummaryComment     string         `json:"summary_comment,omitempty"`
}

type SourceResult struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

type AnnotationLabel string

const (
	AnnotationTP AnnotationLabel = "tp"
	AnnotationFP AnnotationLabel = "fp"
	AnnotationFN AnnotationLabel = "fn"
	AnnotationTN AnnotationLabel = "tn"
)

type AnnotationFile struct {
	FormatVersion int                        `json:"format_version"`
	PaperID       string                     `json:"paper_id"`
	UpdatedAt     time.Time                  `json:"updated_at"`
	Entries       map[string]EntryAnnotation `json:"entries"`
}

type EntryAnnotation struct {
	EntryNumber            int             `json:"entry_number"`
	Label                  AnnotationLabel `json:"label"`
	Note                   string          `json:"note,omitempty"`
	Reviewer               string          `json:"reviewer,omitempty"`
	Timestamp              *time.Time      `json:"timestamp,omitempty"`
	CanonicalReferenceText string          `json:"canonical_reference_text,omitempty"`
}

type SummaryReport struct {
	FormatVersion      int                `json:"format_version"`
	RunID              string             `json:"run_id"`
	GeneratedAt        time.Time          `json:"generated_at"`
	EntryCounts        DecisionCounts     `json:"entry_counts"`
	SummaryStateCounts SummaryStateCounts `json:"summary_state_counts"`
	ReviewCoverage     ReviewCoverage     `json:"review_coverage"`
	ConfusionMatrix    ConfusionMatrix    `json:"confusion_matrix"`
	Metrics            Metrics            `json:"metrics"`
	Papers             []PaperSummary     `json:"papers"`
	Venues             []VenueSummary     `json:"venues"`
}

type DecisionCounts struct {
	MatchFound int `json:"match_found"`
	NoMatch    int `json:"no_match"`
	Error      int `json:"error"`
}

type SummaryStateCounts struct {
	OK      int `json:"ok"`
	Review  int `json:"review"`
	Error   int `json:"error"`
	Unknown int `json:"unknown"`
}

type ReviewCoverage struct {
	ReviewedEntries   int     `json:"reviewed_entries"`
	UnreviewedEntries int     `json:"unreviewed_entries"`
	ReviewedFraction  float64 `json:"reviewed_fraction"`
}

type ConfusionMatrix struct {
	TP int `json:"tp"`
	FP int `json:"fp"`
	FN int `json:"fn"`
	TN int `json:"tn"`
}

type Metrics struct {
	Precision *float64 `json:"precision"`
	Recall    *float64 `json:"recall"`
	F1        *float64 `json:"f1"`
}

type PaperSummary struct {
	PaperID        string             `json:"paper_id"`
	VenueID        string             `json:"venue_id"`
	TotalEntries   int                `json:"total_entries"`
	EntryCounts    DecisionCounts     `json:"entry_counts"`
	ReviewCoverage ReviewCoverage     `json:"review_coverage"`
	SummaryStates  SummaryStateCounts `json:"summary_state_counts"`
	ErrorRate      float64            `json:"error_rate"`
	MatchFoundRate float64            `json:"match_found_rate"`
	NoMatchRate    float64            `json:"no_match_rate"`
}

type VenueSummary struct {
	VenueID        string             `json:"venue_id"`
	PaperCount     int                `json:"paper_count"`
	TotalEntries   int                `json:"total_entries"`
	EntryCounts    DecisionCounts     `json:"entry_counts"`
	ReviewCoverage ReviewCoverage     `json:"review_coverage"`
	SummaryStates  SummaryStateCounts `json:"summary_state_counts"`
	ErrorRate      float64            `json:"error_rate"`
	MatchFoundRate float64            `json:"match_found_rate"`
	NoMatchRate    float64            `json:"no_match_rate"`
}

func NewWorkspace(root string) Workspace {
	return Workspace{Root: root}
}

func (w Workspace) CorpusPath() string {
	return filepath.Join(w.Root, CorpusFileName)
}

func (w Workspace) RunsDir() string {
	return filepath.Join(w.Root, RunsDirName)
}

func (w Workspace) RunDir(runID string) string {
	return filepath.Join(w.RunsDir(), runID)
}

func (w Workspace) RunPath(runID string) string {
	return filepath.Join(w.RunDir(runID), "run.json")
}

func (w Workspace) RunResultsDir(runID string) string {
	return filepath.Join(w.RunDir(runID), ResultsDirName)
}

func (w Workspace) PaperResultPath(runID, paperID string) string {
	return filepath.Join(w.RunResultsDir(runID), paperID+".json")
}

func (w Workspace) ReportsDir(runID string) string {
	return filepath.Join(w.RunDir(runID), ReportsDirName)
}

func (w Workspace) ReportPath(runID string) string {
	return filepath.Join(w.ReportsDir(runID), "summary.json")
}

func (w Workspace) AnnotationsDir() string {
	return filepath.Join(w.Root, AnnotationsDirName)
}

func (w Workspace) AnnotationPath(paperID string) string {
	return filepath.Join(w.AnnotationsDir(), paperID+".json")
}

func (w Workspace) Ensure() error {
	if err := os.MkdirAll(w.Root, 0o755); err != nil {
		return fmt.Errorf("create eval workspace %q: %w", w.Root, err)
	}
	return nil
}

func (w Workspace) SaveCorpus(corpus *Corpus) error {
	if err := w.Ensure(); err != nil {
		return err
	}
	if err := writeJSONFile(w.CorpusPath(), corpus); err != nil {
		return fmt.Errorf("write corpus file %q: %w", w.CorpusPath(), err)
	}
	return nil
}

func (w Workspace) LoadCorpus() (*Corpus, error) {
	var corpus Corpus
	if err := readJSONFile(w.CorpusPath(), &corpus); err != nil {
		return nil, fmt.Errorf("read corpus file %q: %w", w.CorpusPath(), err)
	}
	return &corpus, nil
}

func (w Workspace) EnsureRun(runID string) error {
	if err := os.MkdirAll(w.RunResultsDir(runID), 0o755); err != nil {
		return fmt.Errorf("create run directory %q: %w", w.RunDir(runID), err)
	}
	return nil
}

func (w Workspace) SaveRun(run *Run) error {
	if err := w.EnsureRun(run.RunID); err != nil {
		return err
	}
	if err := writeJSONFile(w.RunPath(run.RunID), run); err != nil {
		return fmt.Errorf("write run file %q: %w", w.RunPath(run.RunID), err)
	}
	return nil
}

func (w Workspace) LoadRun(runID string) (*Run, error) {
	var run Run
	if err := readJSONFile(w.RunPath(runID), &run); err != nil {
		return nil, fmt.Errorf("read run file %q: %w", w.RunPath(runID), err)
	}
	return &run, nil
}

func (w Workspace) SavePaperResult(result *PaperResult) error {
	if err := w.EnsureRun(result.RunID); err != nil {
		return err
	}
	path := w.PaperResultPath(result.RunID, result.PaperID)
	if err := writeJSONFile(path, result); err != nil {
		return fmt.Errorf("write paper result %q: %w", path, err)
	}
	return nil
}

func (w Workspace) LoadPaperResult(runID, paperID string) (*PaperResult, error) {
	var result PaperResult
	path := w.PaperResultPath(runID, paperID)
	if err := readJSONFile(path, &result); err != nil {
		return nil, fmt.Errorf("read paper result %q: %w", path, err)
	}
	return &result, nil
}

func (w Workspace) SaveSummaryReport(report *SummaryReport) error {
	if err := os.MkdirAll(w.ReportsDir(report.RunID), 0o755); err != nil {
		return fmt.Errorf("create reports directory %q: %w", w.ReportsDir(report.RunID), err)
	}
	if err := writeJSONFile(w.ReportPath(report.RunID), report); err != nil {
		return fmt.Errorf("write report file %q: %w", w.ReportPath(report.RunID), err)
	}
	return nil
}

func (w Workspace) SaveAnnotations(file *AnnotationFile) error {
	if err := os.MkdirAll(filepath.Dir(w.AnnotationPath(file.PaperID)), 0o755); err != nil {
		return fmt.Errorf("create annotations directory %q: %w", filepath.Dir(w.AnnotationPath(file.PaperID)), err)
	}
	if err := writeJSONFile(w.AnnotationPath(file.PaperID), file); err != nil {
		return fmt.Errorf("write annotation file %q: %w", w.AnnotationPath(file.PaperID), err)
	}
	return nil
}

func (w Workspace) LoadAnnotations(paperID string) (*AnnotationFile, error) {
	var file AnnotationFile
	path := w.AnnotationPath(paperID)
	if err := readJSONFile(path, &file); err != nil {
		if os.IsNotExist(err) {
			return &AnnotationFile{
				FormatVersion: FormatVersion,
				PaperID:       paperID,
				Entries:       map[string]EntryAnnotation{},
			}, nil
		}
		return nil, fmt.Errorf("read annotation file %q: %w", path, err)
	}
	if file.Entries == nil {
		file.Entries = map[string]EntryAnnotation{}
	}
	return &file, nil
}

func writeJSONFile(path string, value any) error {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, append(payload, '\n'), 0o644); err != nil {
		return err
	}
	return nil
}

func readJSONFile(path string, dest any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return err
	}
	return nil
}
