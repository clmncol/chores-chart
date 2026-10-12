// Package chart draws a resolved week as a US Letter PDF. It knows nothing
// about the chore list format or the filesystem: it consumes a schedule and
// writes bytes.
package chart

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/checkbox"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/border"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/consts/pagesize"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"

	"chores/internal/chore"
)

// The page is a fine 40-unit grid: at Letter size with the default 10mm side
// margins a unit is ~4.9mm, which lands the two day columns at half the text
// width each with a gutter between them.
const (
	grid   = 40
	half   = 19 // units per day column
	gutter = 2  // blank units separating the two day columns

	boxW  = 2 // checkbox column, two units so the box lines up with the heading
	textW = half - boxW
	nameW = 11 // weekday cell in a day heading
	dateW = half - nameW

	headH = 7.5 // day heading row height
	taskH = 6.5 // one task row height
	box   = 4.5 // checkbox square, in mm

	title = "Chore Chart"
)

// A small palette, kept light so the chart stays printer-friendly.
var (
	ink    = rgb(0x11, 0x18, 0x27)
	muted  = rgb(0x6b, 0x72, 0x80)
	rule   = rgb(0xcb, 0xd5, 0xe1)
	headBg = rgb(0xf1, 0xf5, 0xf9)
	headFg = rgb(0x33, 0x41, 0x55)
)

// Render draws the week and writes a PDF to out.
func Render(w chore.Week, out io.Writer) error {
	doc, err := build(w).Generate()
	if err != nil {
		return err
	}
	_, err = out.Write(doc.GetBytes())
	return err
}

func build(w chore.Week) core.Maroto {
	cfg := config.NewBuilder().
		WithPageSize(pagesize.Letter).
		WithMaxGridSize(grid).
		WithTopMargin(14).
		WithBottomMargin(14).
		Build()
	m := maroto.New(cfg)

	m.AddRow(12, text.NewCol(grid, title, props.Text{
		Size:  23,
		Style: fontstyle.Bold,
		Align: align.Center,
		Color: ink,
		Top:   0.5,
	}))
	m.AddRow(6, text.NewCol(grid, weekRange(w.Start), props.Text{
		Size:  10,
		Align: align.Center,
		Color: muted,
		Top:   0.5,
	}))
	m.AddRows(spacer(3))
	m.AddRows(hairline())
	m.AddRows(spacer(6))

	// Two columns of days: Mon-Thu on the left, Fri-Sun on the right. Each slot
	// pairs a left and right day into shared rows, so the columns stay aligned
	// and the week fits on one page.
	left := []int{0, 1, 2, 3}
	right := []int{4, 5, 6}
	for slot := range left {
		l, r := dayAt(w, left, slot), dayAt(w, right, slot)
		m.AddRows(headingRow(l, r))
		for i := 0; i < max(len(l.tasks), len(r.tasks)); i++ {
			m.AddRows(taskRow(l, r, i))
		}
		if slot < len(left)-1 {
			m.AddRows(spacer(5))
		}
	}

	return m
}

type dayCol struct {
	date  time.Time
	tasks []string
	has   bool
}

func dayAt(w chore.Week, days []int, slot int) dayCol {
	if slot >= len(days) {
		return dayCol{}
	}
	i := days[slot]
	return dayCol{date: w.Start.AddDate(0, 0, i), tasks: w.Days[i], has: true}
}

func headingRow(l, r dayCol) core.Row {
	cols := headingCols(l)
	cols = append(cols, gutterCol())
	cols = append(cols, headingCols(r)...)
	return row.New(headH).Add(cols...)
}

// headingCols draws a day's heading as a tinted band: the weekday on the left
// and its date on the right.
func headingCols(day dayCol) []core.Col {
	var name, date string
	bg, fg := (*props.Color)(nil), ink
	if day.has {
		name = strings.ToUpper(day.date.Format("Monday"))
		date = day.date.Format("Jan 2")
		bg, fg = headBg, headFg
	}

	return []core.Col{
		text.NewCol(nameW, name, props.Text{
			Size: 9, Style: fontstyle.Bold, Color: fg, Top: 2.1, Left: 2,
		}).WithStyle(&props.Cell{BackgroundColor: bg}),
		text.NewCol(dateW, date, props.Text{
			Size: 8, Color: fg, Top: 2.35, Align: align.Right, Right: 2,
		}).WithStyle(&props.Cell{BackgroundColor: bg}),
	}
}

func taskRow(l, r dayCol, i int) core.Row {
	cols := taskCells(l.tasks, i)
	cols = append(cols, gutterCol())
	cols = append(cols, taskCells(r.tasks, i)...)
	return row.New(taskH).Add(cols...)
}

func taskCells(tasks []string, i int) []core.Col {
	if i >= len(tasks) {
		return []core.Col{col.New(boxW), col.New(textW)}
	}
	return []core.Col{
		checkbox.NewCol(boxW, "", props.Checkbox{Size: box, Top: (taskH - box) / 2, Left: 2}),
		text.NewCol(textW, tasks[i], props.Text{Size: 9.5, Color: ink, Top: 1.8, Left: 1}),
	}
}

func gutterCol() core.Col {
	return col.New(gutter)
}

func hairline() core.Row {
	return row.New(1).Add(col.New(grid).WithStyle(&props.Cell{
		BorderType:      border.Bottom,
		BorderColor:     rule,
		BorderThickness: 0.3,
	}))
}

func spacer(height float64) core.Row {
	return row.New(height).Add(col.New(grid))
}

func weekRange(start time.Time) string {
	end := start.AddDate(0, 0, 6)
	if start.Month() == end.Month() {
		return fmt.Sprintf("%s – %d, %d", start.Format("January 2"), end.Day(), end.Year())
	}
	return fmt.Sprintf("%s – %s, %d", start.Format("January 2"), end.Format("January 2"), end.Year())
}

func rgb(r, g, b int) *props.Color {
	return &props.Color{Red: r, Green: g, Blue: b}
}
