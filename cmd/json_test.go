package cmd

import (
	"testing"
	"time"

	"chores/internal/chore"
)

func TestWeekJSON(t *testing.T) {
	start := time.Date(2026, 11, 9, 0, 0, 0, 0, time.UTC)
	w := chore.Week{Start: start, Timeless: false}
	w.Days[0] = []string{"Make bed", "Pay bills"}

	got := newWeekJSON(w)
	if got.Start != "2026-11-09" || got.Timeless {
		t.Errorf("header: got start %q timeless %v", got.Start, got.Timeless)
	}
	if len(got.Days) != 7 {
		t.Fatalf("got %d days, want 7", len(got.Days))
	}
	if d := got.Days[0]; d.Date != "2026-11-09" || d.Name != "Monday" || len(d.Tasks) != 2 {
		t.Errorf("monday: got %+v", d)
	}
	// A free day is an empty list, not null; JSON consumers should not need a
	// nil check.
	if d := got.Days[6]; d.Date != "2026-11-15" || d.Name != "Sunday" || d.Tasks == nil || len(d.Tasks) != 0 {
		t.Errorf("sunday: got %+v", d)
	}
}
