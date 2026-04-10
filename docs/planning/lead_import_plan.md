# Lead Import Plan: Gojiberry CSV

## Goal

Import leads from a Gojiberry-format CSV into the database, normalizing contacts,
companies, signals, and keywords into their respective tables. Enables prompts like
"add these new leads to the system" by passing CSV content to an MCP tool.

---

## Data Model Changes

### New columns

```sql
-- leads
ALTER TABLE app.leads ADD COLUMN linkedin_url TEXT;
ALTER TABLE app.leads ADD COLUMN location    TEXT;
ALTER TABLE app.leads ADD COLUMN phone       TEXT;

-- companies
ALTER TABLE app.companies ADD COLUMN linkedin_url TEXT;
```

### New table: `app.signal_keywords`

Signals (e.g. "LinkedIn post engagement") are company-level intent events.
Keywords (e.g. "enterprise modernization") are the topics that triggered them.
One signal can have many keywords.

```sql
CREATE TABLE app.signal_keywords (
    id        SERIAL PRIMARY KEY,
    signal_id INTEGER NOT NULL REFERENCES app.signals(id),
    keyword   TEXT    NOT NULL,
    UNIQUE (signal_id, keyword)
);
```

### Unique constraints (required for upserts)

```sql
ALTER TABLE app.companies ADD CONSTRAINT companies_name_unique UNIQUE (name);
ALTER TABLE app.signals   ADD CONSTRAINT signals_description_unique UNIQUE (description);
```

### Signals are company-level

Signals represent organizational buying intent, not individual contact behavior.
A contact surfaces a signal, but the signal belongs to the company.

```
app.companies ──< app.company_signals >── app.signals ──< app.signal_keywords
                                               │
                                         e.g. "LinkedIn post engagement"
                                               │
                                         keyword: "enterprise modernization"
```

---

## CSV Field Mapping (Gojiberry format)

| CSV field | Maps to |
|---|---|
| First Name + Last Name | `leads.name` |
| Email | `leads.email` (primary, unique) |
| Phone | `leads.phone` |
| Location | `leads.location` |
| Job Title | `leads.title` |
| Total Score | `leads.score` |
| Profile URL | `leads.linkedin_url` |
| Company | `companies.name` |
| Website | `companies.domain` |
| Industry | `companies.industry` |
| Company URL | `companies.linkedin_url` |
| Intent | Normalized → `signals.description` + stored as lead note |
| Intent Keyword | `signal_keywords.keyword` |

### Intent normalization

The `Intent` field contains HTML (e.g. `Just engaged with a <a href='...'>LinkedIn post</a>`).
The raw HTML is stored as a note on the lead for traceability.
The description is normalized to a clean string (e.g. `"LinkedIn post engagement"`)
for the `signals` row.

---

## Import Logic (per CSV row)

1. **Upsert company** by `name` — update `domain`, `industry`, `linkedin_url` if new
2. **Upsert signal** by `description` — normalized from Intent HTML
3. **Upsert signal_keyword** by `(signal_id, keyword)` — from Intent Keyword field
4. **Upsert company_signal** by `(company_id, signal_id)` — link company to signal
5. **Upsert lead** by `email` — update fields if contact already exists
6. **Create note** on lead — raw Intent HTML for traceability

---

## New MCP Tool: `import_leads`

```
name: import_leads
description: Import leads from a Gojiberry CSV. Pass the raw CSV text.
parameters:
  csv: string  — full CSV content including header row
returns:
  summary: { companies, signals, keywords, leads } counts
```

Claude can read a CSV file and pass its contents directly to this tool:
- *"Import the leads from samples/gojiberry-selected-contacts.csv"*
- *"Add these new leads to the system"* (with CSV pasted inline)

---

## Implementation Steps

1. `mcp/migrations/002_add_lead_fields.sql` — new columns, new table, unique constraints
2. `mcp/internal/db/queries/companies.sql` — UpsertCompany
3. `mcp/internal/db/queries/signals.sql` — UpsertSignal, UpsertSignalKeyword, UpsertCompanySignal
4. Update `mcp/internal/db/queries/leads.sql` — UpsertLead
5. Regenerate sqlc (`docker run ... sqlc/sqlc:latest generate`)
6. `mcp/internal/tools/import_leads.go` — CSV parsing + orchestration
7. Register tool in `mcp/internal/tools/tools.go`
