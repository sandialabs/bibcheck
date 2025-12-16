// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package lookup

import "fmt"

func Print(ea *EntryAnalysis) {

	green := "\033[32m"
	yellow := "\033[33m"
	red := "\033[31m"
	reset := "\033[0m"

	if ea.Arxiv.Status == SearchStatusDone {
		if ea.Arxiv.Entry != nil {
			fmt.Println(green + "Arxiv: compare retrieved metadata:" + reset)
			fmt.Println(green + ea.Arxiv.Entry.ToString() + reset)
		} else {
			fmt.Println(red + "Arxiv: ID NOT FOUND" + reset)
		}
	} else if ea.Arxiv.Error != nil {
		fmt.Println(yellow + fmt.Sprintf("Arxiv: search error: %v", ea.Arxiv.Error) + reset)
	}

	if ea.Crossref.Status == SearchStatusDone {
		if ea.Crossref.Work != nil {
			fmt.Println(green + "Crossref: compare retrieved metadata" + reset)
			fmt.Println(green + ea.Crossref.Work.ToString() + reset)
		} else {
			fmt.Println(red + "Crossref: NO SEARCH RESULTS" + reset)
			fmt.Println(red + "          " + ea.Crossref.Comment + reset)
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
		if ea.OSTI.Record != nil {
			fmt.Println(green + "OSTI: compare retrieved metadata:" + reset)
			fmt.Println(green + ea.OSTI.Record.ToString() + reset)
		} else {
			fmt.Println(red + "OSTI: NOT FOUND" + reset)
		}
	} else if ea.OSTI.Error != nil {
		fmt.Println(yellow + fmt.Sprintf("OSTI: search error: %v", ea.OSTI.Error) + reset)
	}

	if ea.Online.Status == SearchStatusDone {
		if ea.Online.Metadata != nil {
			fmt.Println(green + "Online: ✓ LOOKS OKAY" + reset)
			fmt.Println(green + "     " + ea.Online.Metadata.ToString() + reset)
		} else {
			fmt.Println(red + "Online: NOT FOUND" + reset)
		}
	} else if ea.Online.Error != nil {
		fmt.Println(yellow + fmt.Sprintf("Online: search error: %v", ea.Online.Error) + reset)
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
