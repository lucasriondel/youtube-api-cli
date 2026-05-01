package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/mattn/go-runewidth"
)

// Format selects how a command renders its result.
type Format int

const (
	Table Format = iota
	JSON
	Plain
)

// FormatFromFlags converts the standard --json/--plain flag pair into a Format.
func FormatFromFlags(jsonFlag, plainFlag bool) Format {
	switch {
	case jsonFlag:
		return JSON
	case plainFlag:
		return Plain
	default:
		return Table
	}
}

// Render writes rows in the chosen format.
//   - JSON: marshals `data` directly.
//   - Plain: tab-separated values, no header, stable for scripting.
//   - Table: header + tab-aligned columns for humans.
func Render(w io.Writer, format Format, headers []string, rows [][]string, data any) error {
	switch format {
	case JSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	case Plain:
		for _, row := range rows {
			fmt.Fprintln(w, strings.Join(row, "\t"))
		}
		return nil
	default:
		return renderTable(w, headers, rows)
	}
}

func renderTable(w io.Writer, headers []string, rows [][]string) error {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = runewidth.StringWidth(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i >= len(widths) {
				break
			}
			if l := runewidth.StringWidth(cell); l > widths[i] {
				widths[i] = l
			}
		}
	}
	writeRow(w, headers, widths)
	sep := make([]string, len(headers))
	for i, width := range widths {
		sep[i] = strings.Repeat("-", width)
	}
	writeRow(w, sep, widths)
	for _, row := range rows {
		writeRow(w, row, widths)
	}
	return nil
}

func writeRow(w io.Writer, row []string, widths []int) {
	parts := make([]string, len(row))
	for i, cell := range row {
		if i >= len(widths) {
			parts[i] = cell
			continue
		}
		pad := widths[i] - runewidth.StringWidth(cell)
		if pad < 0 {
			pad = 0
		}
		parts[i] = cell + strings.Repeat(" ", pad)
	}
	fmt.Fprintln(w, strings.Join(parts, "  "))
}
