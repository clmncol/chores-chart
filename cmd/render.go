package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"chores/internal/chart"
	"chores/internal/chore"
)

// Output formats; the resolved week is the same for all of them.
const (
	formatPDF  = "pdf"
	formatJSON = "json"
)

// Default file names, written when the output path names a directory.
const (
	pdfName  = "chores.pdf"
	jsonName = "chores.json"
)

func renderCmd() *cobra.Command {
	var (
		input  string
		output string
		date   string
		format string
	)

	c := &cobra.Command{
		Use:   "render",
		Short: "Render the chore chart",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			plan, err := chore.Load(input)
			if err != nil {
				return err
			}
			for _, w := range plan.Lint() {
				fmt.Fprintln(cmd.ErrOrStderr(), "warning:", w)
			}
			ref, err := referenceDate(date)
			if err != nil {
				return err
			}

			week := plan.Week(ref)
			switch format {
			case formatPDF:
				return renderPDF(cmd, week, output)
			case formatJSON:
				return renderJSON(cmd, week, output)
			default:
				return fmt.Errorf("invalid --format %q (want %s or %s)", format, formatPDF, formatJSON)
			}
		},
	}

	c.Flags().StringVarP(&input, "input", "i", "chores.txt", "chore list to render")
	c.Flags().StringVarP(&output, "output", "o", ".", "output path; a directory writes "+pdfName+" or "+jsonName+" inside it")
	c.Flags().StringVarP(&date, "date", "d", "", "reference date YYYY-MM-DD (default today)")
	c.Flags().StringVarP(&format, "format", "f", formatPDF, "output format: "+formatPDF+" or "+formatJSON)
	return c
}

func renderPDF(cmd *cobra.Command, week chore.Week, output string) error {
	f, path, err := create(output, pdfName)
	if err != nil {
		return err
	}
	// Close as a best effort if we return early; the happy path below closes
	// explicitly so its error is reported.
	defer func() { _ = f.Close() }()

	if err := chart.Render(week, f); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	_, err = fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s for %s – %s\n", path,
		week.Start.Format("Mon Jan 2"), week.Start.AddDate(0, 0, 6).Format("Mon Jan 2"))
	return err
}

// renderJSON writes to stdout, the expected target for a programmatic consumer,
// unless the caller asked for a file with -o.
func renderJSON(cmd *cobra.Command, week chore.Week, output string) error {
	if !cmd.Flags().Changed("output") {
		return writeJSON(cmd.OutOrStdout(), week)
	}

	f, _, err := create(output, jsonName)
	if err != nil {
		return err
	}
	if err := writeJSON(f, week); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func referenceDate(s string) (time.Time, error) {
	if s == "" {
		return time.Now(), nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid --date %q (want YYYY-MM-DD)", s)
	}
	return t, nil
}

// create opens the output file, creating parent directories, and returns the
// path actually written. A directory (existing, or written with a trailing
// separator) gets name appended.
func create(path, name string) (*os.File, string, error) {
	if isDir(path) || strings.HasSuffix(path, string(os.PathSeparator)) {
		path = filepath.Join(path, name)
	}
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, "", err
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, "", err
	}
	return f, path, nil
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
