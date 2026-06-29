// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package version

var gitSha string
var gitRefName string

func GitSha() string {
	if gitSha == "" {
		return "[git SHA not provided]"
	} else {
		return gitSha
	}
}

func GitRefName() string {
	if gitRefName == "" {
		return "[git ref not provided]"
	} else {
		return gitRefName
	}
}

func String() string {
	return GitRefName() + " (" + shortGitSha() + ")"
}

func shortGitSha() string {
	if gitSha == "" {
		return GitSha()
	}
	sha := gitSha
	if len(sha) <= 7 {
		return sha
	}
	return sha[:7]
}
