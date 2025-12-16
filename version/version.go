// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package version

var gitSha string
var buildDate string

func GitSha() string {
	if gitSha == "" {
		return "[git SHA not provided]"
	} else {
		return gitSha
	}
}

func BuildDate() string {
	if buildDate == "" {
		return "[build date not provided]"
	} else {
		return buildDate
	}
}

func String() string {
	return "0.4.0"
}
