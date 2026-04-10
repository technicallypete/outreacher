# CSV Preprocessing: Server-Side Normalization

## Problem

When Claude Desktop receives a CSV file to import, it performs its own preprocessing
before calling MCP tools — stripping BOM characters, normalizing line endings,
re-quoting fields, and stripping HTML. This preprocessing:

1. **Adds latency** — Claude spends time analyzing and transforming content before
   the first tool call, causing noticeable delays (up to several minutes for small files).
2. **Risks data loss** — Claude may alter field values (e.g., re-quoting intent HTML,
   normalizing unicode) in ways that break downstream parsing.
3. **Is unnecessary** — the Go server can handle all normalization cheaply and
   deterministically.

## Solution

Move all normalization into the Go importer. Tell Claude (via tool descriptions) to
pass raw file content verbatim — its only job is to read the file and send it.

### Tool description framing

**Before (prohibitive — confuses Claude):**
> "do NOT sanitize or tokenize the content before calling this tool"

**After (descriptive — clarifies Claude's role):**
> "Pass the verbatim file content — do not reformat or modify the CSV. The server
> handles all normalization."

The framing shift matters: prohibitive wording makes Claude uncertain about what
it's allowed to do; descriptive wording tells it that its job is simpler than it
thought.

## Server-Side Preprocessing

A `Preprocess(s string) string` function is added to the importer package and called
at the top of every parse entry point (`parseCSV` and `importCSV`).

### What it handles

| Issue | Fix |
|---|---|
| UTF-8 BOM (`\xEF\xBB\xBF`) | Strip from start of input |
| Windows line endings (CRLF) | Replace `\r\n` → `\n` |
| Bare carriage returns (`\r`) | Replace `\r` → `\n` |
| Leading/trailing whitespace | `strings.TrimSpace` |

### What's already handled

| Issue | Mechanism |
|---|---|
| Unbalanced quotes | `csv.Reader.LazyQuotes = true` |
| Variable-width rows | `csv.Reader.FieldsPerRecord = -1` |
| Leading spaces in fields | `csv.Reader.TrimLeadingSpace = true` |
| HTML in Intent field | `intentRe.ReplaceAllString` in `normalizeIntent` |

## Files Changed

- `mcp/internal/importer/parse.go` — add `Preprocess()`, call it in `parseCSV()`
- `mcp/internal/tools/import_leads.go` — call `importer.Preprocess()` in `importCSV()`,
  update tool descriptions
- `mcp/internal/tools/import_csv.go` — update tool descriptions

## Expected Outcome

Claude reads the file with its native tools and passes the raw string directly to
`import_csv`. No preprocessing step. First tool call happens immediately.
