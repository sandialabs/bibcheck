// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package config

import "github.com/cwpearson/bibliography-checker/version"

func UserAgent() string {
	return "bibliography-checker / " + version.String() + " github.com/cwpearson/bibliography-checker"
}

func UserEmail() string {
	return "cwpears@sandia.gov"
}
