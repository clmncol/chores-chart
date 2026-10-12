package chore

import (
	"slices"
	"strings"
	"testing"
	"time"
)

func load(t *testing.T, src string) Plan {
	t.Helper()
	p, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func tasksOn(p Plan, date string) []string {
	d, err := time.Parse("2006-01-02", date)
	if err != nil {
		panic(err)
	}
	return p.Week(d).Days[(int(d.Weekday())+6)%7]
}

func TestSchedule(t *testing.T) {
	p := load(t, `
Make bed
Vacuum | sat
Pay bills | 1st
Dust | 22nd
Clean gutters | apr 5, oct 5
Clean windows | de 9
Replace HVAC filters | 1st every 3 months
Replace faucet | 2026-11-23
`)

	cases := []struct {
		date string
		want []string
	}{
		{"2026-10-14", []string{"Make bed"}},                                      // Wednesday
		{"2026-10-17", []string{"Make bed", "Vacuum"}},                            // Saturday
		{"2026-10-22", []string{"Make bed", "Dust"}},                              // 22nd, an ordinal with a suffix
		{"2026-11-01", []string{"Make bed", "Pay bills"}},                         // 1st, Nov is not a quarter month
		{"2026-10-01", []string{"Make bed", "Pay bills", "Replace HVAC filters"}}, // Oct is a quarter month
		{"2026-04-05", []string{"Make bed", "Clean gutters"}},                     // apr 5
		{"2026-12-09", []string{"Make bed", "Clean windows"}},                     // de 9, a unique month prefix
		{"2026-11-23", []string{"Make bed", "Replace faucet"}},                    // one-off
		{"2026-11-24", []string{"Make bed"}},                                      // the day after
	}
	for _, tc := range cases {
		if got := tasksOn(p, tc.date); !slices.Equal(got, tc.want) {
			t.Errorf("%s: got %q, want %q", tc.date, got, tc.want)
		}
	}
}

func TestOrderAndComments(t *testing.T) {
	p := load(t, "# header\n\nZebra\n   \nApple\n# trailing\n")

	// Order follows the file, and comments/blank lines are skipped.
	if got, want := tasksOn(p, "2026-10-14"), []string{"Zebra", "Apple"}; !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestParseErrors(t *testing.T) {
	for _, src := range []string{
		"Vacuum | someday",
		"Vacuum | t",
		"Vacuum | m 5",
		"Vacuum | a 1",
		"Vacuum | 1s",
		"Vacuum | 1stst",
		"Vacuum | 1st every 2 weeks",
		" | sat",
	} {
		if _, err := Parse(strings.NewReader(src)); err == nil {
			t.Errorf("Parse(%q): want error, got nil", src)
		}
	}
}
