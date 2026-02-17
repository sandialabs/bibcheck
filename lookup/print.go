// Copyright 2025 National Technology and Engineering Solutions of Sandia
// SPDX-License-Identifier: BSD-3-Clause
package lookup

import "fmt"

func Print(r *Result) {

	green := "\033[32m"
	yellow := "\033[33m"
	red := "\033[31m"
	reset := "\033[0m"

	if r.Arxiv.Status == SearchStatusDone {
		if r.Arxiv.Entry != nil {
			fmt.Println(green + "Arxiv: compare retrieved metadata:" + reset)
			fmt.Println(green + r.Arxiv.Entry.ToString() + reset)
		} else {
			fmt.Println(red + "Arxiv: ID NOT FOUND" + reset)
		}
	} else if r.Arxiv.Error != nil {
		fmt.Println(yellow + fmt.Sprintf("Arxiv: search error: %v", r.Arxiv.Error) + reset)
	}

	if r.Crossref.Status == SearchStatusDone {
		if r.Crossref.Work != nil {
			fmt.Println(green + "Crossref: compare retrieved metadata" + reset)
			fmt.Println(green + r.Crossref.Work.ToString() + reset)
		} else {
			fmt.Println(red + "Crossref: NO SEARCH RESULTS" + reset)
			fmt.Println(red + "          " + r.Crossref.Comment + reset)
		}
	} else if r.Crossref.Error != nil {
		fmt.Println(yellow + fmt.Sprintf("Crossref: search error: %v", r.Crossref.Error) + reset)
	}

	if r.DOIOrg.Status == SearchStatusDone {
		if r.DOIOrg.Found {
			fmt.Println(green + "DOI: EXISTS (content match not verified)" + reset)
		} else {
			fmt.Println(red + "DOI: NOT FOUND" + reset)
		}
	} else if r.DOIOrg.Error != nil {
		fmt.Println(yellow + fmt.Sprintf("DOI:  search error: %v", r.DOIOrg.Error) + reset)
	}

	if r.OSTI.Status == SearchStatusDone {
		if r.OSTI.Record != nil {
			fmt.Println(green + "OSTI: compare retrieved metadata:" + reset)
			fmt.Println(green + r.OSTI.Record.ToString() + reset)
		} else {
			fmt.Println(red + "OSTI: NOT FOUND" + reset)
		}
	} else if r.OSTI.Error != nil {
		fmt.Println(yellow + fmt.Sprintf("OSTI: search error: %v", r.OSTI.Error) + reset)
	}

	if r.Online.Status == SearchStatusDone {
		if r.Online.Metadata != nil {
			fmt.Println(green + "Online: ✓ LOOKS OKAY" + reset)
			fmt.Println(green + "     " + r.Online.Metadata.ToString() + reset)
		} else {
			fmt.Println(red + "Online: NOT FOUND" + reset)
		}
	} else if r.Online.Error != nil {
		fmt.Println(yellow + fmt.Sprintf("Online: search error: %v", r.Online.Error) + reset)
	}

	if r.Web.Status == SearchStatusDone {
		if r.Web.Exists {
			fmt.Println(green + "Web Search: ✓ LOOKS OKAY" + reset)
			fmt.Println(green + "            " + r.Web.Comment + reset)
		} else {
			fmt.Println(red + "Web Search: NOT FOUND" + reset)
			fmt.Println(red + "            " + r.Web.Comment + reset)
		}
	} else if r.Web.Error != nil {
		fmt.Println(yellow + fmt.Sprintf("Web: search error: %v", r.Web.Error) + reset)
	}

}
