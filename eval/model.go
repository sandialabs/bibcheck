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
)

type Workspace struct {
	Root string
}

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

func NewWorkspace(root string) Workspace {
	return Workspace{Root: root}
}

func (w Workspace) CorpusPath() string {
	return filepath.Join(w.Root, CorpusFileName)
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
	payload, err := json.MarshalIndent(corpus, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal corpus: %w", err)
	}
	if err := os.WriteFile(w.CorpusPath(), append(payload, '\n'), 0o644); err != nil {
		return fmt.Errorf("write corpus file %q: %w", w.CorpusPath(), err)
	}
	return nil
}
