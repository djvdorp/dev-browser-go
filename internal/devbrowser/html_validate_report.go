package devbrowser

import "time"

type HTMLValidateMeta struct {
	URL     string `json:"url"`
	Page    string `json:"page"`
	Profile string `json:"profile"`
	TS      string `json:"ts"`
}

type HTMLValidateReport struct {
	Meta     HTMLValidateMeta      `json:"meta"`
	Findings []HTMLValidateFinding `json:"findings"`
}

func NewHTMLValidateReport(url, page, profile string, ts time.Time, findings []HTMLValidateFinding) HTMLValidateReport {
	if ts.IsZero() {
		ts = time.Now()
	}
	return HTMLValidateReport{
		Meta:     HTMLValidateMeta{URL: url, Page: page, Profile: profile, TS: ts.UTC().Format(time.RFC3339Nano)},
		Findings: findings,
	}
}
