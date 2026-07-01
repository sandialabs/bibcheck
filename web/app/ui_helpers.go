// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
//go:build js && wasm

package main

import (
	"fmt"
	"strings"

	"github.com/sandialabs/bibcheck/web/workflow"
)

func providerText(provider workflow.ProviderKind) string {
	if provider == "" {
		return "selected provider"
	}
	return string(provider)
}

func progressText(state workflow.State) string {
	if state.Total == 0 {
		return ""
	}
	return fmt.Sprintf("%d of %d analyzed", state.Completed, state.Total)
}

func maxProgress(state workflow.State) int {
	if state.Total < 1 {
		return 1
	}
	return state.Total
}

func valueProgress(state workflow.State) int {
	if state.Completed < 0 {
		return 0
	}
	return state.Completed
}

func entryStatus(entry workflow.EntryState) string {
	if entry.TextStatus == "error" || entry.AnalysisStatus == "error" {
		return "error"
	}
	if entry.AnalysisStatus == "completed" {
		return "completed"
	}
	if entry.TextStatus == "active" || entry.AnalysisStatus == "active" {
		return "active"
	}
	return "pending"
}

func statusCopy(status string) string {
	switch status {
	case "active":
		return "Working..."
	case "completed":
		return ""
	case "error":
		return "Error"
	default:
		return "Pending"
	}
}

func statusLabel(status string) string {
	return strings.ReplaceAll(nonEmpty(status, "unknown"), "-", " ")
}

func statusClass(status string) string {
	if status == "" {
		return "unknown"
	}
	return status
}

func panelStatusClass(status string) string {
	return "state-" + statusClass(status)
}

func lookupStatusClass(status string) string {
	return "lookup-" + statusClass(status)
}

func summaryStatusClass(status string) string {
	return "summary-" + statusClass(status)
}

func lookupFallback(entry workflow.EntryState) string {
	switch entry.AnalysisStatus {
	case "active":
		return "Looking up metadata."
	case "completed":
		return "No relevant lookup results."
	case "error":
		return nonEmpty(entry.Error, "Lookup did not complete.")
	default:
		return statusCopy(entry.AnalysisStatus)
	}
}

func summaryTitle(status string) string {
	switch status {
	case "ok":
		return "Looks okay"
	case "review":
		return "Review suggested"
	case "error":
		return "Error"
	case "active":
		return "Analyzing"
	case "pending":
		return "Pending"
	default:
		return "Unknown"
	}
}

func summaryFallback(entry workflow.EntryState, status string) string {
	switch status {
	case "active":
		return "Building the aggregate summary."
	case "error":
		return nonEmpty(entry.Error, "Analysis did not complete.")
	case "ok":
		return "No issues found."
	case "review":
		return "No matching metadata found."
	default:
		return statusCopy(entry.AnalysisStatus)
	}
}

func nonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
