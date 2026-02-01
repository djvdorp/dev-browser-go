package devbrowser

import "sort"

func SortNetworkEntries(entries []NetworkEntry) {
	sort.Slice(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.Started == b.Started {
			if a.URL == b.URL {
				return a.Method < b.Method
			}
			return a.URL < b.URL
		}
		return a.Started < b.Started
	})
}

func SortConsoleEntries(entries []ConsoleEntry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].TimeMS == entries[j].TimeMS {
			return entries[i].ID < entries[j].ID
		}
		return entries[i].TimeMS < entries[j].TimeMS
	})
}

func SortHTMLValidateFindings(findings []HTMLValidateFinding) {
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].RuleID == findings[j].RuleID {
			if findings[i].Location == findings[j].Location {
				return findings[i].Message < findings[j].Message
			}
			return findings[i].Location < findings[j].Location
		}
		return findings[i].RuleID < findings[j].RuleID
	})
}
