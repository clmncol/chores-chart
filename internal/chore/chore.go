// Package chore parses a line-based chore list and resolves it into a weekly
// schedule. It knows nothing about how that schedule is presented.
//
// The list is one chore per line:
//
//	<task> | <when>
//
// where <when> is a comma-separated list of dates. With no "|" the chore is
// every day. Blank lines and lines starting with "#" are ignored, and the
// order of lines is the order chores appear on the chart.
package chore

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

// Week is a resolved Monday-Sunday schedule: the tasks due on each day, with
// day 0 = Monday.
type Week struct {
	Start time.Time
	Days  [7][]string
	// Timeless is true when the plan has no chore pinned to a date, so every
	// week resolves the same and the chart need not name its week.
	Timeless bool
}

// Plan is a parsed chore list, in file order.
type Plan struct {
	chores []chore
}

type chore struct {
	task   string
	raw    string  // the line as written, for diagnostics
	tokens []token // empty means every day
}

// Load parses a chore list from path.
func Load(path string) (Plan, error) {
	f, err := os.Open(path)
	if err != nil {
		return Plan{}, err
	}
	defer func() { _ = f.Close() }()

	p, err := Parse(f)
	if err != nil {
		return Plan{}, fmt.Errorf("%s: %w", path, err)
	}
	return p, nil
}

// Parse reads a chore list.
func Parse(r io.Reader) (Plan, error) {
	var p Plan
	sc := bufio.NewScanner(r)
	for line := 1; sc.Scan(); line++ {
		s := strings.TrimSpace(sc.Text())
		if s == "" || strings.HasPrefix(s, "#") {
			continue
		}

		task, when := s, ""
		if i := strings.Index(s, "|"); i >= 0 {
			task, when = strings.TrimSpace(s[:i]), strings.TrimSpace(s[i+1:])
		}
		if task == "" {
			return Plan{}, fmt.Errorf("line %d: missing task before \"|\"", line)
		}

		c := chore{task: task, raw: s}
		if when != "" {
			for _, part := range strings.Split(when, ",") {
				tok, err := parseToken(part)
				if err != nil {
					return Plan{}, fmt.Errorf("line %d: %v", line, err)
				}
				c.tokens = append(c.tokens, tok)
			}
		}
		p.chores = append(p.chores, c)
	}
	if err := sc.Err(); err != nil {
		return Plan{}, err
	}
	return p, nil
}

// Week resolves the plan into the week containing ref, Monday first.
func (p Plan) Week(ref time.Time) Week {
	start := mondayOf(ref)
	w := Week{Start: start, Timeless: p.timeless()}
	for i := range w.Days {
		date := start.AddDate(0, 0, i)
		for _, c := range p.chores {
			if c.due(date) {
				w.Days[i] = append(w.Days[i], c.task)
			}
		}
	}
	return w
}

func (c chore) due(date time.Time) bool {
	if len(c.tokens) == 0 {
		return true
	}
	for _, t := range c.tokens {
		if t.match(date) {
			return true
		}
	}
	return false
}

// timeless reports whether every week resolves identically, which holds when no
// chore is pinned to a date: only daily and weekday chores are present.
func (p Plan) timeless() bool {
	for _, c := range p.chores {
		for _, t := range c.tokens {
			if t.kind != kindDaily && t.kind != kindWeekly {
				return false
			}
		}
	}
	return true
}

// Lint reports chores that can never fall on some of the dates they name, such
// as a chore on the 31st in months with 30 days. The list still renders in the
// months where the date does occur; warnings are advisory.
func (p Plan) Lint() []string {
	var out []string
	for _, c := range p.chores {
		var missing []time.Month
		for _, t := range c.tokens {
			missing = append(missing, t.absentMonths()...)
		}
		if len(missing) > 0 {
			out = append(out, fmt.Sprintf("%s: never occurs in %s", c.raw, joinMonths(missing)))
		}
	}
	return out
}

// absentMonths lists the months a monthly or yearly date cannot occur in, e.g.
// the 31st is absent from February, April, June, September and November.
func (t token) absentMonths() []time.Month {
	var out []time.Month
	switch t.kind {
	case kindMonthly:
		for m := time.January; m <= time.December; m++ {
			if t.every >= 2 && (int(m)-1)%t.every != 0 {
				continue // not a month this chore is scheduled in
			}
			if !existsIn(t.day, m) {
				out = append(out, m)
			}
		}
	case kindYearly:
		if !existsIn(t.day, t.month) {
			out = append(out, t.month)
		}
	}
	return out
}

// existsIn reports whether day ever occurs in month m. A leap year is used so
// feb 29 counts as real while apr 31 does not: time.Date rolls the overflow into
// the next month, which is how the check reads without a month-length table.
func existsIn(day int, m time.Month) bool {
	return time.Date(2000, m, day, 0, 0, 0, 0, time.UTC).Month() == m
}

// ─── dates ──────────────────────────────────────────────────────────────────

type kind int

const (
	kindDaily kind = iota
	kindWeekly
	kindMonthly
	kindYearly
	kindOnce
)

type token struct {
	kind    kind
	weekday time.Weekday
	month   time.Month
	day     int
	date    time.Time
	every   int // months between occurrences; 0 = the natural interval
}

func (t token) match(date time.Time) bool {
	switch t.kind {
	case kindDaily:
		return true
	case kindWeekly:
		return date.Weekday() == t.weekday
	case kindMonthly:
		if date.Day() != t.day {
			return false
		}
		return t.every < 2 || (int(date.Month())-1)%t.every == 0
	case kindYearly:
		return date.Month() == t.month && date.Day() == t.day
	case kindOnce:
		return sameDay(date, t.date)
	}
	return false
}

// parseToken reads one date, e.g. "daily", "sat", "1st", "apr 5",
// "2026-11-23" or "1st every 3 months".
func parseToken(raw string) (token, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "" {
		return token{}, fmt.Errorf("empty date")
	}

	base, every := s, 0
	if i := strings.Index(s, " every "); i >= 0 {
		base = strings.TrimSpace(s[:i])
		n, err := parseEvery(s[i+len(" every "):])
		if err != nil {
			return token{}, fmt.Errorf("%q: %w", raw, err)
		}
		every = n
	}

	var tok token
	switch base {
	case "daily", "everyday", "every day":
		tok = token{kind: kindDaily}
	default:
		if d, err := time.Parse("2006-01-02", base); err == nil {
			tok = token{kind: kindOnce, date: d}
			break
		}
		m, day, ok, err := parseMonthDay(base)
		if err != nil {
			return token{}, fmt.Errorf("%q: %w", raw, err)
		}
		if ok {
			tok = token{kind: kindYearly, month: m, day: day}
			break
		}
		if day, ok := parseDayOfMonth(base); ok {
			tok = token{kind: kindMonthly, day: day}
			break
		}
		switch days := weekdaysFor(base); len(days) {
		case 1:
			tok = token{kind: kindWeekly, weekday: days[0]}
		case 0:
			return token{}, fmt.Errorf("unrecognized date %q", raw)
		default:
			return token{}, fmt.Errorf("%q is ambiguous (%s)", raw, joinWeekdays(days))
		}
	}

	if every != 0 {
		if tok.kind != kindMonthly {
			return token{}, fmt.Errorf("%q: an interval only applies to a day of the month", raw)
		}
		tok.every = every
	}
	return tok, nil
}

func parseEvery(s string) (int, error) {
	fields := strings.Fields(s)
	if len(fields) != 2 {
		return 0, fmt.Errorf("bad interval %q (want e.g. \"3 months\")", s)
	}
	n, err := strconv.Atoi(fields[0])
	if err != nil || n < 1 {
		return 0, fmt.Errorf("bad interval %q", s)
	}
	if unit := strings.TrimSuffix(strings.ToLower(fields[1]), "s"); unit != "month" {
		return 0, fmt.Errorf("unsupported interval unit %q (only months)", fields[1])
	}
	return n, nil
}

func parseMonthDay(s string) (time.Month, int, bool, error) {
	fields := strings.Fields(s)
	if len(fields) != 2 {
		return 0, 0, false, nil
	}
	month, ok, err := parseMonth(fields[0])
	if err != nil {
		return 0, 0, false, err
	}
	if !ok {
		return 0, 0, false, nil
	}
	day, ok := parseDayOfMonth(fields[1])
	if !ok {
		return 0, 0, false, nil
	}
	return month, day, true, nil
}

// parseMonth matches a month name by unique prefix, e.g. "apr" or "se".
func parseMonth(s string) (time.Month, bool, error) {
	var found []time.Month
	for m := time.January; m <= time.December; m++ {
		if strings.HasPrefix(strings.ToLower(m.String()), s) {
			found = append(found, m)
		}
	}
	switch len(found) {
	case 0:
		return 0, false, nil
	case 1:
		return found[0], true, nil
	default:
		return 0, false, fmt.Errorf("ambiguous month (%s)", joinMonths(found))
	}
}

func parseDayOfMonth(s string) (int, bool) {
	digits := s
	for _, suffix := range []string{"st", "nd", "rd", "th"} {
		if strings.HasSuffix(digits, suffix) {
			digits = digits[:len(digits)-len(suffix)]
			break
		}
	}
	n, err := strconv.Atoi(digits)
	if err != nil || n < 1 || n > 31 {
		return 0, false
	}
	return n, true
}

var weekdays = []time.Weekday{
	time.Monday, time.Tuesday, time.Wednesday, time.Thursday,
	time.Friday, time.Saturday, time.Sunday,
}

func weekdaysFor(key string) []time.Weekday {
	k := strings.ToLower(strings.TrimSpace(key))
	if k == "" {
		return nil
	}
	// Weekdays are often written in the plural: "tues" and "weds" are
	// prefixes of no weekday name, but their singular forms are.
	k = strings.TrimSuffix(k, "s")
	if k == "" {
		return nil
	}
	var out []time.Weekday
	for _, day := range weekdays {
		if strings.HasPrefix(strings.ToLower(day.String()), k) {
			out = append(out, day)
		}
	}
	return out
}

func joinMonths(months []time.Month) string {
	names := make([]string, len(months))
	for i, m := range months {
		names[i] = m.String()
	}
	return strings.Join(names, ", ")
}

func joinWeekdays(days []time.Weekday) string {
	names := make([]string, len(days))
	for i, d := range days {
		names[i] = d.String()
	}
	return strings.Join(names, ", ")
}

func mondayOf(t time.Time) time.Time {
	d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
	return d.AddDate(0, 0, -((int(d.Weekday()) + 6) % 7))
}

func sameDay(a, b time.Time) bool {
	return a.Year() == b.Year() && a.YearDay() == b.YearDay()
}
