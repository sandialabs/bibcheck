// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestReadTextInputStdin(t *testing.T) {
	for _, args := range [][]string{{}, {"-"}} {
		cmd := &cobra.Command{}
		cmd.SetIn(strings.NewReader("[1] Some entry"))
		got, err := readTextInput(cmd, args)
		if err != nil {
			t.Fatalf("args %v: unexpected error: %v", args, err)
		}
		if got != "[1] Some entry" {
			t.Fatalf("args %v: expected stdin content, got %q", args, got)
		}
	}
}

func TestReadTextInputFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bib.txt")
	if err := os.WriteFile(path, []byte("[1] File entry"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := readTextInput(&cobra.Command{}, []string{path})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "[1] File entry" {
		t.Fatalf("expected file content, got %q", got)
	}
}

func TestReadTextInputMissingFile(t *testing.T) {
	if _, err := readTextInput(&cobra.Command{}, []string{filepath.Join(t.TempDir(), "nope.txt")}); err == nil {
		t.Fatal("expected error for missing file")
	}
}
