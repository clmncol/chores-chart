package cmd

import (
	"encoding/json"
	"io"
	"time"

	"chores/internal/chore"
)

// weekJSON is the machine-readable form of a resolved week. Dates are always
// present here, even for a timeless list, so a consumer reads a concrete
// resolution instead of a template.
type weekJSON struct {
	Start    string    `json:"start"`
	Timeless bool      `json:"timeless"`
	Days     []dayJSON `json:"days"`
}

type dayJSON struct {
	Date  string   `json:"date"`
	Name  string   `json:"name"`
	Tasks []string `json:"tasks"`
}

func newWeekJSON(w chore.Week) weekJSON {
	out := weekJSON{Start: w.Start.Format(time.DateOnly), Timeless: w.Timeless}
	for i, tasks := range w.Days {
		date := w.Start.AddDate(0, 0, i)
		if tasks == nil {
			tasks = []string{} // a free day is an empty list, never null
		}
		out.Days = append(out.Days, dayJSON{
			Date:  date.Format(time.DateOnly),
			Name:  date.Format("Monday"),
			Tasks: tasks,
		})
	}
	return out
}

func writeJSON(w io.Writer, week chore.Week) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(newWeekJSON(week))
}
