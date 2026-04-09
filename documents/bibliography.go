// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package documents

import "fmt"

type Bibliography struct {
	PDF       []byte
	Text      string
	StartPage int
	EndPage   int
}

func (b *Bibliography) Content() (string, error) {
	if b == nil {
		return "", fmt.Errorf("missing bibliography")
	}
	return b.Text, nil
}
