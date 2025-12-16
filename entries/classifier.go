// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package entries

const (
	KindBook                  string = "book"
	KindScientificPublication string = "scientific_publication"
	KindSoftwarePackage       string = "software_package"
	KindUnknown               string = "unknown"
	KindWebsite               string = "website"
)

type Classifier interface {
	Classify(text string) (string, error)
}
