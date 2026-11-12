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

func renderCmd() *cobra.Command {
	var (
		input  string
		output string
		date   string
	)

	c := &cobra.Command{
		Use:   "render",
		Short: "Render the chore chart PDF",
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

			f, path, err := create(output)
			if err != nil {
				return err
			}
			// Close as a best effort if we return early; the happy path below
			// closes explicitly so its error is reported.
			defer func() { _ = f.Close() }()

			week := plan.Week(ref)
			if err := chart.Render(week, f); err != nil {
				return err
			}
			if err := f.Close(); err != nil {
				return err
			}

			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s for %s – %s\n", path,
				week.Start.Format("Mon Jan 2"), week.Start.AddDate(0, 0, 6).Format("Mon Jan 2"))
			return err
		},
	}

	c.Flags().StringVarP(&input, "input", "i", "chores.txt", "chore list to render")
	c.Flags().StringVarP(&output, "output", "o", ".", "PDF output path; a directory writes "+defaultName+" inside it")
	c.Flags().StringVarP(&date, "date", "d", "", "reference date YYYY-MM-DD (default today)")
	return c
}

// defaultName is written when the output path names a directory.
const defaultName = "chores.pdf"

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
// separator) gets the default file name appended.
func create(path string) (*os.File, string, error) {
	if isDir(path) || strings.HasSuffix(path, string(os.PathSeparator)) {
		path = filepath.Join(path, defaultName)
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
