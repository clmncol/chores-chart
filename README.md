# chores

Render a printable US Letter chore chart for one week from a plain-text
chore list. The list covers the whole year; the renderer keeps the week
containing the reference date (today by default) and lays it out day by day with
checkboxes.

## The list

One chore per line: `<task> | <when>`. No `|` means every day. Blank lines and
`#` comments are ignored, and line order is the order chores appear.

```
Make bed
Take out trash | mon
Pay bills | 1st
Replace HVAC filters | 1st every 3 months
Clean gutters | apr 5, oct 5
Replace faucet cartridge | 2026-11-23
```

`<when>` is one or more comma-separated dates. The shape of a date sets its
cadence:

| date                 | cadence                                  |
| -------------------- | ---------------------------------------- |
| `daily`              | every day                                |
| `sat`                | weekly (`mon`, `tues`, `weds`, ...)      |
| `1st`, `15th`        | day of the month                         |
| `1st every 3 months` | day of the month, every N months         |
| `apr 5`              | that date each year (two = twice a year) |
| `2026-11-23`         | a one-off on that date                   |

Names match by unique prefix (`apr`, `se`, `sat`), and ambiguous ones error out
(`t`, `m 5`). Monthly and yearly chores appear only in the week containing
their date.

## Usage

```sh
chores render                # ./chores.txt -> ./chores.pdf
chores render -i list.txt    # another list
chores render -d 2026-11-01  # preview any week
chores render -o week.pdf    # another output path
chores render -o out         # a directory gets chores.pdf
```

Flags: `-i/--input` (default `chores.txt`), `-o/--output` (default `.`, the
current directory), `-d/--date` (reference date `YYYY-MM-DD`, default today).
Passing a directory — existing, or with a trailing slash — writes
`chores.pdf` inside it.

## Development

See [DEVELOPMENT.md](DEVELOPMENT.md) for the code layout, build tasks, and
possible improvements.
