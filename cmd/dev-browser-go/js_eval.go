package main

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"
)

func newJSEvalCmd() *cobra.Command {
	var pageName string
	var expression string
	var format string
	var selector string
	var ariaRole string
	var ariaName string
	var nth int
	var timeout int

	cmd := &cobra.Command{
		Use:   "js-eval [expression]",
		Short: "Evaluate JavaScript in page context",
		Args:  cobra.MaximumNArgs(1),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			exprFlagSet := cmd.Flags().Changed("expr")
			hasPositional := len(args) > 0
			if hasPositional && exprFlagSet {
				return errors.New("provide expression as positional argument OR via --expr flag, not both")
			}
			if hasPositional {
				expression = args[0]
			}
			if strings.TrimSpace(expression) == "" {
				return errors.New(`expression required; e.g.: js-eval "document.title" or js-eval --expr "document.title"`)
			}
			if format != "auto" && format != "json" && format != "string" && format != "number" && format != "boolean" {
				return errors.New("--format must be auto, json, string, number, or boolean")
			}
			if timeout < 0 {
				return errors.New("--timeout-ms must be >= 0")
			}
			return nil
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			payload := map[string]interface{}{
				"expression": expression,
				"format":     format,
				"nth":        nth,
				"timeout_ms": timeout,
			}
			if strings.TrimSpace(selector) != "" {
				payload["selector"] = selector
			}
			if strings.TrimSpace(ariaRole) != "" {
				payload["aria_role"] = ariaRole
			}
			if strings.TrimSpace(ariaName) != "" {
				payload["aria_name"] = ariaName
			}
			return runWithPage(pageName, "js_eval", payload)
		},
	}

	cmd.Flags().StringVar(&pageName, "page", "main", "Page name")
	cmd.Flags().StringVar(&expression, "expr", "", "JavaScript expression to evaluate")
	cmd.Flags().StringVar(&format, "format", "auto", "Output format (auto|json|string|number|boolean)")
	cmd.Flags().StringVar(&selector, "selector", "", "CSS selector to scope evaluation")
	cmd.Flags().StringVar(&ariaRole, "aria-role", "", "ARIA role to scope evaluation")
	cmd.Flags().StringVar(&ariaName, "aria-name", "", "ARIA name to scope evaluation")
	cmd.Flags().IntVar(&nth, "nth", 1, "Nth match (1-based)")
	cmd.Flags().IntVar(&timeout, "timeout-ms", 5_000, "Timeout for element wait")

	return cmd
}
