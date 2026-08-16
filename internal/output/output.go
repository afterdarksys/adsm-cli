// Package output renders command results in the format the user asked for.
//
// Every command writes through here rather than calling fmt directly, so that
// -o json stays machine-parseable across the whole CLI. A single command that
// prints a stray human-readable line to stdout breaks every script piping it
// into jq.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"gopkg.in/yaml.v3"
)

// Format is an output encoding.
type Format string

const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
	FormatYAML  Format = "yaml"
)

// ParseFormat validates a --output value.
func ParseFormat(s string) (Format, error) {
	switch Format(strings.ToLower(strings.TrimSpace(s))) {
	case FormatTable:
		return FormatTable, nil
	case FormatJSON:
		return FormatJSON, nil
	case FormatYAML:
		return FormatYAML, nil
	default:
		return "", fmt.Errorf("unknown output format %q: want table, json, or yaml", s)
	}
}

// Table is a simple column-oriented result set.
type Table struct {
	Headers []string
	Rows    [][]string
}

// Render writes v to w in the requested format.
//
// For table format v must be a *Table; structured formats accept anything
// json/yaml can encode.
func Render(w io.Writer, f Format, v any) error {
	switch f {
	case FormatJSON:
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(v)

	case FormatYAML:
		enc := yaml.NewEncoder(w)
		enc.SetIndent(2)
		if err := enc.Encode(v); err != nil {
			return err
		}
		return enc.Close()

	case FormatTable:
		t, ok := v.(*Table)
		if !ok {
			return fmt.Errorf("table output requires a *output.Table, got %T", v)
		}
		return renderTable(w, t)

	default:
		return fmt.Errorf("unsupported output format %q", f)
	}
}

func renderTable(w io.Writer, t *Table) error {
	if t == nil {
		return nil
	}
	tw := tabwriter.NewWriter(w, 0, 0, 3, ' ', 0)
	if len(t.Headers) > 0 {
		if _, err := fmt.Fprintln(tw, strings.Join(t.Headers, "\t")); err != nil {
			return err
		}
	}
	for _, row := range t.Rows {
		if _, err := fmt.Fprintln(tw, strings.Join(row, "\t")); err != nil {
			return err
		}
	}
	return tw.Flush()
}
