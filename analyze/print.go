// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package analyze

import "fmt"

func Print(ea *EntryAnalysis) {

	green := "\033[32m"
	yellow := "\033[33m"
	red := "\033[31m"
	reset := "\033[0m"

	if ea.Arxiv.Status == SearchStatusDone {
		if ea.Arxiv.Found {
			fmt.Println(green + "Arxiv: compare retrieved metadata:" + reset)
			fmt.Println(green + ea.Arxiv.Result + reset)
		} else {
			fmt.Println(red + "Arxiv: ID NOT FOUND" + reset)
		}
	} else if ea.Arxiv.Error != nil {
		fmt.Println(yellow + fmt.Sprintf("Arxiv: search error: %v", ea.Arxiv.Error) + reset)
	}

	if ea.Crossref.Status == SearchStatusDone {
		if ea.Crossref.Found {
			fmt.Println(green + "Crossref: compare retrieved metadata" + reset)
			fmt.Println(green + ea.Crossref.Result + reset)
		} else {
			fmt.Println(red + "Crossref: NO SEARCH RESULTS" + reset)
			fmt.Println(red + "          " + ea.Crossref.Result + reset)
		}
	} else if ea.Crossref.Error != nil {
		fmt.Println(yellow + fmt.Sprintf("Crossref: search error: %v", ea.Crossref.Error) + reset)
	}

	if ea.DOIOrg.Status == SearchStatusDone {
		if ea.DOIOrg.Found {
			fmt.Println(green + "DOI: EXISTS (content match not verified)" + reset)
		} else {
			fmt.Println(red + "DOI: NOT FOUND" + reset)
		}
	} else if ea.DOIOrg.Error != nil {
		fmt.Println(yellow + fmt.Sprintf("DOI:  search error: %v", ea.DOIOrg.Error) + reset)
	}

	if ea.OSTI.Status == SearchStatusDone {
		if ea.OSTI.Found {
			fmt.Println(green + "OSTI: compare retrieved metadata:" + reset)
			fmt.Println(green + ea.OSTI.Result + reset)
		} else {
			fmt.Println(red + "OSTI: NOT FOUND" + reset)
		}
	} else if ea.OSTI.Error != nil {
		fmt.Println(yellow + fmt.Sprintf("OSTI: search error: %v", ea.OSTI.Error) + reset)
	}

	if ea.URL.Status == SearchStatusDone {
		if ea.URL.Exists {
			fmt.Println(green + "URL: ✓ LOOKS OKAY" + reset)
			fmt.Println(green + "     " + ea.URL.Comment + reset)
		} else {
			fmt.Println(red + "URL: NO MATCH" + reset)
			fmt.Println(red + "     " + ea.URL.Comment + reset)
		}
	} else if ea.URL.Error != nil {
		fmt.Println(yellow + fmt.Sprintf("Web: search error: %v", ea.URL.Error) + reset)
	}

	if ea.Web.Status == SearchStatusDone {
		if ea.Web.Exists {
			fmt.Println(green + "Web Search: ✓ LOOKS OKAY" + reset)
			fmt.Println(green + "            " + ea.Web.Comment + reset)
		} else {
			fmt.Println(red + "Web Search: NOT FOUND" + reset)
			fmt.Println(red + "            " + ea.Web.Comment + reset)
		}
	} else if ea.Web.Error != nil {
		fmt.Println(yellow + fmt.Sprintf("Web: search error: %v", ea.Web.Error) + reset)
	}

}
