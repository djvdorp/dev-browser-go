package main

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// newTestRoot creates a minimal root command suitable for unit-testing subcommands
// without requiring a live daemon.
func newTestRoot() *cobra.Command {
	globalOpts = &globalOptions{}
	root := &cobra.Command{
		Use:           "dev-browser-go",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	bindGlobalFlags(root)
	root.PersistentPreRunE = func(cmd *cobra.Command, _ []string) error {
		return applyGlobalOptions(cmd)
	}
	return root
}

// noopRunE replaces a command's RunE with a no-op so only argument / pre-run
// validation is exercised without needing a live browser daemon.
func withNoopRunE(cmd *cobra.Command) *cobra.Command {
	cmd.RunE = func(_ *cobra.Command, _ []string) error { return nil }
	return cmd
}

// --- save-html tests ---------------------------------------------------------

func TestSaveHTMLNoPathSucceeds(t *testing.T) {
	root := newTestRoot()
	root.AddCommand(withNoopRunE(newSaveHTMLCmd()))
	root.SetArgs([]string{"save-html"})
	if err := root.Execute(); err != nil {
		t.Fatalf("expected no error when --path is omitted, got: %v", err)
	}
}

func TestSaveHTMLWithPathSucceeds(t *testing.T) {
	root := newTestRoot()
	root.AddCommand(withNoopRunE(newSaveHTMLCmd()))
	root.SetArgs([]string{"save-html", "--path", "out.html"})
	if err := root.Execute(); err != nil {
		t.Fatalf("expected no error with --path flag, got: %v", err)
	}
}

// --- js-eval tests -----------------------------------------------------------

func TestJSEvalPositionalExpr(t *testing.T) {
	root := newTestRoot()
	root.AddCommand(withNoopRunE(newJSEvalCmd()))
	root.SetArgs([]string{"js-eval", "document.title"})
	if err := root.Execute(); err != nil {
		t.Fatalf("expected no error for positional expression, got: %v", err)
	}
}

func TestJSEvalFlagExpr(t *testing.T) {
	root := newTestRoot()
	root.AddCommand(withNoopRunE(newJSEvalCmd()))
	root.SetArgs([]string{"js-eval", "--expr", "document.title"})
	if err := root.Execute(); err != nil {
		t.Fatalf("expected no error for --expr flag, got: %v", err)
	}
}

func TestJSEvalBothPositionalAndFlagFails(t *testing.T) {
	root := newTestRoot()
	root.AddCommand(withNoopRunE(newJSEvalCmd()))
	root.SetArgs([]string{"js-eval", "document.title", "--expr", "window.location"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when both positional and --expr are provided")
	}
	if !strings.Contains(err.Error(), "not both") {
		t.Fatalf("expected conflict error mentioning 'not both', got: %v", err)
	}
}

func TestJSEvalNoExprFails(t *testing.T) {
	root := newTestRoot()
	root.AddCommand(withNoopRunE(newJSEvalCmd()))
	root.SetArgs([]string{"js-eval"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when no expression is provided")
	}
	if !strings.Contains(err.Error(), "expression required") {
		t.Fatalf("expected 'expression required' error, got: %v", err)
	}
}

func TestJSEvalInvalidFormatFails(t *testing.T) {
	root := newTestRoot()
	root.AddCommand(withNoopRunE(newJSEvalCmd()))
	root.SetArgs([]string{"js-eval", "--expr", "1+1", "--format", "invalid"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for invalid --format")
	}
	if !strings.Contains(err.Error(), "--format") {
		t.Fatalf("expected --format error, got: %v", err)
	}
}

// --- console tests -----------------------------------------------------------

func TestConsolePositionalArgFails(t *testing.T) {
	root := newTestRoot()
	root.AddCommand(withNoopRunE(newConsoleCmd()))
	root.SetArgs([]string{"console", "document.title"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error when positional arg passed to console")
	}
	if !strings.Contains(err.Error(), "js-eval") {
		t.Fatalf("expected error message to reference js-eval, got: %v", err)
	}
}
