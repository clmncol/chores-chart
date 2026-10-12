# Development

## Layout

```
main.go              thin entry point
cmd/                 cobra CLI (render)
internal/chore/      list parser + weekly resolution
internal/chart/      PDF rendering (maroto) — no file I/O
```

`internal/chore` turns the list into a `Week` — seven slices of task strings,
Monday first — and knows nothing about PDFs. `internal/chart` turns a `Week`
into PDF bytes and knows nothing about files or the list format. That split is
the seam to build on: a new output format belongs behind `cmd/` and should reuse
`chore.Week` rather than re-parsing the list.

## Tasks

```sh
mise run generate    # render this week's chart
mise run build       # -> bin/chores
mise run clean       # remove out/, bin/, and chores.pdf
go test ./...        # parser + schedule tests
```

## Possible improvements

Ideas worth considering if I'm bored.

### Intermediate format (`--format json|md`)

The resolved week (`chore.Week`) is already the intermediate representation

```sh
chores render --format json   # resolved week as JSON
chores render --format md     # same data as a markdown checklist
chores render --format pdf    # current output (default)
```

What is _not_ worth it yet: splitting the renderer into a separate program that consumes that
format. That adds a schema to maintain, serialization code, and a second binary that drifts out of
sync. Stay one binary with a `--format` flag until a real second consumer exists; the
`internal/chore` / `internal/chart` split is already the seam a future split would use.

### Anchor "every N months" to a start date

`1st every 3 months` is currently anchored to calendar quarters (Jan/Apr/Jul/Oct) regardless of when
it starts. Something like `1st every 3 months from nov` could be useful

### Guard against page overflow

Row heights are fixed, so a long list spills onto a second page without warning. 

### Render a range of weeks

`chores render --weeks 4` to get a month of charts in one PDF for batch printing. 
