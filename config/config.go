// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package config

import "github.com/sandialabs/bibcheck/version"

func UserAgent() string {
	return "bibcheck / " + version.String() + " github.com/sandialabs/bibcheck"
}

func UserEmail() string {
	return "cwpears@sandia.gov"
}
