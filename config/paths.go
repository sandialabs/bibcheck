// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func StateHome() (string, error) {
	if dir := strings.TrimSpace(os.Getenv("XDG_STATE_HOME")); dir != "" {
		return dir, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home: %w", err)
	}
	if home == "" {
		return "", fmt.Errorf("resolve user home: empty home directory")
	}

	return filepath.Join(home, ".local", "state"), nil
}

func OpenAIAuditDir(settings Settings) (string, error) {
	if dir := strings.TrimSpace(settings.OpenAIAuditDir); dir != "" {
		return dir, nil
	}

	stateHome, err := StateHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(stateHome, "bibcheck", "openai-audit"), nil
}
