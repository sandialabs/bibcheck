// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package entries

type CompareResult struct {
	Explanation  string `json:"explanation"`
	IsEquivalent bool   `json:"is_equivalent"`
}

type Comparer interface {
	Compare(e1, e2 string) (*CompareResult, error)
}
