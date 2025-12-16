// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package analyze

import (
	"encoding/base64"
	"fmt"
	"os"
)

func Encode(path string) (string, error) {
	// Read PDF file
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read PDF error: %w", err)
	}

	// Encode to base64
	return base64.StdEncoding.EncodeToString(data), nil
}
