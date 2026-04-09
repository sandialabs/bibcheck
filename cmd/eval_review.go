// Copyright 2026 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/sandialabs/bibcheck/eval"
)

type reviewItem struct {
	paper       eval.RunPaper
	result      *eval.PaperResult
	entry       eval.EntryResult
	annotations *eval.AnnotationFile
}

func runEvalReviewCommand(runID string) error {
	workspaceRoot, err := filepath.Abs(evalWorkspace)
	if err != nil {
		return fmt.Errorf("resolve eval workspace %q: %w", evalWorkspace, err)
	}

	workspace := eval.NewWorkspace(workspaceRoot)
	run, err := workspace.LoadRun(runID)
	if err != nil {
		return err
	}

	items, err := collectReviewItems(workspace, run)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		fmt.Println("No review items matched the current filters.")
		return nil
	}

	reader := bufio.NewReader(os.Stdin)
	savedPapers := map[string]bool{}

	for idx, item := range items {
		if err := presentReviewItem(idx+1, len(items), item); err != nil {
			return err
		}

		labelInput, err := promptLine(reader, "Label [tp/fp/fn/tn/skip/quit]")
		if err != nil {
			return err
		}
		switch strings.ToLower(strings.TrimSpace(labelInput)) {
		case "quit", "q":
			break
		case "skip", "s":
			continue
		}
		if strings.EqualFold(strings.TrimSpace(labelInput), "quit") || strings.EqualFold(strings.TrimSpace(labelInput), "q") {
			break
		}

		existing, hasExisting := findAnnotation(item.annotations, item.entry.EntryNumber)
		label, keepExisting, err := parseReviewLabel(labelInput, existing, hasExisting)
		if err != nil {
			fmt.Printf("Invalid label: %v\n\n", err)
			continue
		}
		if keepExisting {
			fmt.Print("Keeping existing annotation.\n\n")
			continue
		}

		notePrompt := "Note [blank=keep existing, -=clear]"
		if !hasExisting {
			notePrompt = "Note [optional, -=clear]"
		}
		noteInput, err := promptLine(reader, notePrompt)
		if err != nil {
			return err
		}

		canonicalPrompt := "Canonical reference [blank=keep existing, -=clear]"
		if !hasExisting {
			canonicalPrompt = "Canonical reference [optional, -=clear]"
		}
		canonicalInput, err := promptLine(reader, canonicalPrompt)
		if err != nil {
			return err
		}

		annotation := eval.EntryAnnotation{
			EntryNumber: item.entry.EntryNumber,
			Label:       label,
			Reviewer:    chooseReviewer(existing.Reviewer),
		}
		now := time.Now().UTC()
		annotation.Timestamp = &now
		annotation.Note = chooseOptionalField(noteInput, existing.Note, hasExisting)
		annotation.CanonicalReferenceText = chooseOptionalField(canonicalInput, existing.CanonicalReferenceText, hasExisting)

		if item.annotations.Entries == nil {
			item.annotations.Entries = map[string]eval.EntryAnnotation{}
		}
		item.annotations.FormatVersion = eval.FormatVersion
		item.annotations.PaperID = item.paper.PaperID
		item.annotations.UpdatedAt = now
		item.annotations.Entries[strconv.Itoa(item.entry.EntryNumber)] = annotation

		if err := workspace.SaveAnnotations(item.annotations); err != nil {
			return err
		}
		savedPapers[item.paper.PaperID] = true
		fmt.Printf("Saved %s entry %d to %s\n\n", item.paper.PaperID, item.entry.EntryNumber, workspace.AnnotationPath(item.paper.PaperID))
	}

	fmt.Printf("Updated annotations for %d papers\n", len(savedPapers))
	return nil
}

func collectReviewItems(workspace eval.Workspace, run *eval.Run) ([]reviewItem, error) {
	items := []reviewItem{}
	for _, paper := range run.Papers {
		if paper.Status != eval.RunStatusDone {
			continue
		}
		if !matchesEvalFilters(paper.VenueID, paper.PaperID) {
			continue
		}

		result, err := workspace.LoadPaperResult(run.RunID, paper.PaperID)
		if err != nil {
			return nil, err
		}
		annotations, err := workspace.LoadAnnotations(paper.PaperID)
		if err != nil {
			return nil, err
		}

		for _, entry := range result.Entries {
			if !matchesDecisionFilter(entry.FinalDecision) {
				continue
			}
			_, reviewed := findAnnotation(annotations, entry.EntryNumber)
			if evalUnreviewed && reviewed {
				continue
			}
			items = append(items, reviewItem{
				paper:       paper,
				result:      result,
				entry:       entry,
				annotations: annotations,
			})
		}
	}
	return items, nil
}

func presentReviewItem(index, total int, item reviewItem) error {
	fmt.Printf("Review %d/%d\n", index, total)
	fmt.Printf("Paper: %s\n", item.paper.PaperID)
	fmt.Printf("Venue: %s\n", item.paper.VenueID)
	fmt.Printf("Entry: %d\n", item.entry.EntryNumber)
	fmt.Printf("Decision: %s\n", item.entry.FinalDecision)
	if item.entry.PrimarySource != "" {
		fmt.Printf("Primary Source: %s\n", item.entry.PrimarySource)
	}
	fmt.Printf("Summary State: %s\n", item.entry.SummaryState)
	if item.entry.SummaryComment != "" {
		fmt.Printf("Summary Comment: %s\n", item.entry.SummaryComment)
	}
	fmt.Println("Entry Text:")
	fmt.Println(indentBlock(item.entry.ExtractedEntryText))
	if len(item.entry.Sources) > 0 {
		fmt.Println("Sources:")
		for _, source := range item.entry.Sources {
			line := fmt.Sprintf("- %s: %s", source.Name, source.Status)
			if source.Detail != "" {
				line += " (" + source.Detail + ")"
			}
			fmt.Println(line)
		}
	}

	if annotation, ok := findAnnotation(item.annotations, item.entry.EntryNumber); ok {
		fmt.Printf("Existing Annotation: %s", annotation.Label)
		if annotation.Reviewer != "" {
			fmt.Printf(" by %s", annotation.Reviewer)
		}
		fmt.Println()
		if annotation.Note != "" {
			fmt.Printf("Existing Note: %s\n", annotation.Note)
		}
		if annotation.CanonicalReferenceText != "" {
			fmt.Println("Existing Canonical Reference:")
			fmt.Println(indentBlock(annotation.CanonicalReferenceText))
		}
	} else {
		fmt.Println("Existing Annotation: none")
	}
	fmt.Println()
	return nil
}

func promptLine(reader *bufio.Reader, prompt string) (string, error) {
	fmt.Printf("%s: ", prompt)
	line, err := reader.ReadString('\n')
	if err != nil {
		if err.Error() == "EOF" {
			return strings.TrimSpace(line), nil
		}
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func parseReviewLabel(input string, existing eval.EntryAnnotation, hasExisting bool) (eval.AnnotationLabel, bool, error) {
	value := strings.ToLower(strings.TrimSpace(input))
	if value == "" {
		if hasExisting {
			return existing.Label, true, nil
		}
		return "", false, fmt.Errorf("blank label without existing annotation")
	}
	switch eval.AnnotationLabel(value) {
	case eval.AnnotationTP, eval.AnnotationFP, eval.AnnotationFN, eval.AnnotationTN:
		return eval.AnnotationLabel(value), false, nil
	default:
		return "", false, fmt.Errorf("expected tp, fp, fn, tn, skip, or quit")
	}
}

func chooseReviewer(existing string) string {
	if strings.TrimSpace(evalReviewer) != "" {
		return strings.TrimSpace(evalReviewer)
	}
	return existing
}

func chooseOptionalField(input, existing string, hasExisting bool) string {
	switch strings.TrimSpace(input) {
	case "-":
		return ""
	case "":
		if hasExisting {
			return existing
		}
		return ""
	default:
		return strings.TrimSpace(input)
	}
}

func matchesDecisionFilter(decision string) bool {
	if len(evalDecisionFilter) == 0 {
		return true
	}
	return containsString(evalDecisionFilter, decision)
}

func indentBlock(text string) string {
	if text == "" {
		return "  "
	}
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = "  " + line
	}
	return strings.Join(lines, "\n")
}
