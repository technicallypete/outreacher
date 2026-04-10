package importer

import (
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"
)

// nullable returns a *string, nil if the trimmed string is empty.
func nullable(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

// nullableInt32 parses a plain integer string. Returns nil if empty or unparseable.
func nullableInt32(s string) *int32 {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	n, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return nil
	}
	v := int32(n)
	return &v
}

// parseDollarAmount parses Revli-style dollar strings like "$1M", "$500K", "$1.5B".
// Returns nil if empty or unparseable.
func parseDollarAmount(s string) *int64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	s = strings.TrimPrefix(s, "$")
	s = strings.ReplaceAll(s, ",", "")
	multiplier := int64(1)
	upper := strings.ToUpper(s)
	switch {
	case strings.HasSuffix(upper, "B"):
		multiplier = 1_000_000_000
		s = s[:len(s)-1]
	case strings.HasSuffix(upper, "M"):
		multiplier = 1_000_000
		s = s[:len(s)-1]
	case strings.HasSuffix(upper, "K"):
		multiplier = 1_000
		s = s[:len(s)-1]
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return nil
	}
	v := int64(f * float64(multiplier))
	return &v
}

// boolFromNonEmpty returns true if s is non-empty, false if empty, nil if s is
// the special sentinel "nil". Used for the "Currently Hiring" field.
func boolPtr(v bool) *bool {
	return &v
}

// joinLocation builds a location string from city/state/country parts, omitting blanks.
func joinLocation(parts ...string) *string {
	var kept []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			kept = append(kept, p)
		}
	}
	if len(kept) == 0 {
		return nil
	}
	s := strings.Join(kept, ", ")
	return &s
}

// buildIntel marshals non-empty key/value pairs into a compact JSON object string.
// Returns nil if all values are empty.
func buildIntel(pairs ...string) *string {
	if len(pairs)%2 != 0 {
		panic("buildIntel: pairs must be even")
	}
	var parts []string
	for i := 0; i < len(pairs); i += 2 {
		v := strings.TrimSpace(pairs[i+1])
		if v != "" {
			k := pairs[i]
			// Simple JSON encoding — values may contain quotes so escape them.
			v = strings.ReplaceAll(v, `\`, `\\`)
			v = strings.ReplaceAll(v, `"`, `\"`)
			parts = append(parts, fmt.Sprintf(`"%s":"%s"`, k, v))
		}
	}
	if len(parts) == 0 {
		return nil
	}
	s := "{" + strings.Join(parts, ",") + "}"
	return &s
}

// parseCSV reads all rows from a CSV string, returning header column index map
// and data rows. Returns an error for malformed CSV or empty data.
func parseCSV(content string) (col map[string]int, rows [][]string, err error) {
	r := csv.NewReader(strings.NewReader(content))
	r.LazyQuotes = true
	r.TrimLeadingSpace = true
	r.FieldsPerRecord = -1 // allow rows with fewer fields than the header
	all, err := r.ReadAll()
	if err != nil {
		return nil, nil, fmt.Errorf("parse csv: %w", err)
	}
	if len(all) < 2 {
		return nil, nil, fmt.Errorf("CSV has no data rows")
	}
	col = make(map[string]int, len(all[0]))
	for i, h := range all[0] {
		col[strings.TrimSpace(h)] = i
	}
	return col, all[1:], nil
}

// get returns the trimmed value of a named column, or "" if not present.
func get(row []string, col map[string]int, name string) string {
	i, ok := col[name]
	if !ok || i >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[i])
}
