package tools

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	db "github.com/vitruviantech/outreacher/internal/db/gen"
	"github.com/vitruviantech/outreacher/internal/importer"
)

// intentRe strips HTML tags from the Intent field.
var intentRe = regexp.MustCompile(`<[^>]+>`)

// normalizeIntent strips HTML and produces a clean signal description.
func normalizeIntent(raw string) string {
	clean := intentRe.ReplaceAllString(raw, "")
	clean = strings.TrimSpace(clean)
	lower := strings.ToLower(clean)
	switch {
	case strings.Contains(lower, "linkedin post"):
		return "LinkedIn post engagement"
	case strings.Contains(lower, "linkedin article"):
		return "LinkedIn article engagement"
	case strings.Contains(lower, "website"):
		return "Website visit"
	default:
		if clean != "" {
			return clean
		}
		return "Unknown intent"
	}
}

// parseKeyword strips surrounding quotes from intent keyword values.
func parseKeyword(raw string) string {
	return strings.Trim(strings.TrimSpace(raw), `"`)
}

// nullable returns a *string, nil if empty.
func nullable(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

// nullableTimestamp parses common date formats to a pgtype.Timestamptz.
// Returns a zero (invalid) value if the string is empty or unparseable.
var importDateFormats = []string{
	"Jan 02, 2006 3:04 PM",
	"Jan 02, 2006 15:04",
	"Jan 2, 2006",
	"January 2, 2006",
	"01/02/2006",
	"2006-01-02",
	"01-02-2006",
	"Apr 04 2026", // short form without time
}

func nullableTimestamp(s string) pgtype.Timestamptz {
	s = strings.TrimSpace(s)
	if s == "" {
		return pgtype.Timestamptz{}
	}
	for _, layout := range importDateFormats {
		if t, err := time.Parse(layout, s); err == nil {
			return pgtype.Timestamptz{Time: t, Valid: true}
		}
	}
	return pgtype.Timestamptz{}
}

// nullableFloat32 parses a score string to *float32, nil if empty.
func nullableFloat32(s string) *float32 {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var f float32
	if _, err := fmt.Sscanf(s, "%f", &f); err != nil {
		return nil
	}
	return &f
}

// gojiberrySummary extends importer.ImportSummary with signal/keyword counts.
type gojiberrySummary struct {
	importer.ImportSummary
	Signals  int `json:"signals"`
	Keywords int `json:"keywords"`
}

// importCSV is the shared implementation used by both import tools. campaignID
// scopes all created companies, leads, and identifiers to the target campaign.
func importCSV(ctx context.Context, q *db.Queries, csvContent string, campaignID int32, llm LLMConfig) (*mcp.CallToolResult, error) {
	_ = llm // format-detected gojiberry always uses the direct parser

	r := csv.NewReader(strings.NewReader(csvContent))
	r.LazyQuotes = true
	r.TrimLeadingSpace = true
	r.FieldsPerRecord = -1 // allow rows with fewer fields than the header

	rows, err := r.ReadAll()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("parse csv: %v", err)), nil
	}
	if len(rows) < 2 {
		return mcp.NewToolResultError("CSV has no data rows"), nil
	}

	header := rows[0]
	col := make(map[string]int, len(header))
	for i, h := range header {
		col[strings.TrimSpace(h)] = i
	}

	get := func(row []string, name string) string {
		i, ok := col[name]
		if !ok || i >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[i])
	}

	var summary gojiberrySummary
	seenSignals := make(map[string]int32)
	seenCompanies := make(map[string]int32)

	for _, row := range rows[1:] {
		firstName := get(row, "First Name")
		lastName := get(row, "Last Name")
		email := get(row, "Email")
		linkedinURL := get(row, "Profile URL")
		name := strings.TrimSpace(firstName + " " + lastName)
		if name == " " || name == "" {
			name = email
		}
		if name == "" {
			name = linkedinURL
		}

		// ── Company ──────────────────────────────────────────────
		companyName := get(row, "Company")
		var companyID *int32
		if companyName != "" {
			cid, exists := seenCompanies[companyName]
			if !exists {
				cid, err = q.UpsertCompany(ctx, db.UpsertCompanyParams{
					CampaignID: campaignID,
					Name:        companyName,
					Domain:      nullable(get(row, "Website")),
					Industry:    nullable(get(row, "Industry")),
					LinkedinUrl: nullable(get(row, "Company URL")),
				})
				if err != nil {
					return nil, fmt.Errorf("upsert company %q: %w", companyName, err)
				}
				seenCompanies[companyName] = cid
				summary.Companies++
			}
			companyID = &cid
		}

		// ── Signal + Keyword ───────────────────���──────────────────
		intentRaw := get(row, "Intent")
		keyword := parseKeyword(get(row, "Intent Keyword"))

		if intentRaw != "" && companyID != nil {
			signalDesc := normalizeIntent(intentRaw)
			signalID, exists := seenSignals[signalDesc]
			if !exists {
				signalID, err = q.UpsertSignal(ctx, signalDesc)
				if err != nil {
					return nil, fmt.Errorf("upsert signal: %w", err)
				}
				seenSignals[signalDesc] = signalID
				summary.Signals++
			}
			if err := q.UpsertCompanySignal(ctx, db.UpsertCompanySignalParams{
				CompanyID: *companyID,
				SignalID:  signalID,
			}); err != nil {
				return nil, fmt.Errorf("upsert company_signal: %w", err)
			}
			if keyword != "" {
				if err := q.UpsertSignalKeyword(ctx, db.UpsertSignalKeywordParams{
					SignalID: signalID,
					Keyword:  keyword,
				}); err != nil {
					return nil, fmt.Errorf("upsert signal_keyword: %w", err)
				}
				summary.Keywords++
			}
		}

		// ── Lead ─────────────────────────────────────────────────
		identType, identValue := "", ""
		switch {
		case email != "":
			identType, identValue = "email", email
		case linkedinURL != "":
			identType, identValue = "linkedin", linkedinURL
		default:
			summary.Skipped++
			continue
		}

		leadID, err := q.FindLeadByIdentifier(ctx, db.FindLeadByIdentifierParams{
			CampaignID: campaignID,
			Type:    identType,
			Value:   identValue,
		})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("find lead by identifier: %w", err)
		}

		sourcedAt := nullableTimestamp(get(row, "Import Date"))

		if errors.Is(err, pgx.ErrNoRows) {
			leadID, err = q.CreateLead(ctx, db.CreateLeadParams{
				CampaignID:  campaignID,
				Name:        name,
				Email:       nullable(email),
				CompanyID:   companyID,
				Title:       nullable(get(row, "Job Title")),
				Score:       nullableFloat32(get(row, "Total Score")),
				LinkedinUrl: nullable(linkedinURL),
				Location:    nullable(get(row, "Location")),
				Phone:       nullable(get(row, "Phone")),
				SourcedAt:   sourcedAt,
			})
			if err != nil {
				return nil, fmt.Errorf("create lead %q: %w", identValue, err)
			}
			summary.Leads++
			summary.Rows = append(summary.Rows, importer.ImportedRow{ID: leadID, Name: name, Company: companyName, Action: "created"})

			if err := q.UpsertIdentifier(ctx, db.UpsertIdentifierParams{
				CampaignID: campaignID,
				LeadID:     leadID,
				Type:       identType,
				Value:      identValue,
			}); err != nil {
				return nil, fmt.Errorf("upsert identifier: %w", err)
			}
			// Register secondary identifier if both are present
			if identType == "email" && linkedinURL != "" {
				if err := q.UpsertIdentifier(ctx, db.UpsertIdentifierParams{
					CampaignID: campaignID,
					LeadID:     leadID,
					Type:       "linkedin",
					Value:      linkedinURL,
				}); err != nil {
					return nil, fmt.Errorf("upsert linkedin identifier: %w", err)
				}
			} else if identType == "linkedin" && email != "" {
				if err := q.UpsertIdentifier(ctx, db.UpsertIdentifierParams{
					CampaignID: campaignID,
					LeadID:     leadID,
					Type:       "email",
					Value:      email,
				}); err != nil {
					return nil, fmt.Errorf("upsert email identifier: %w", err)
				}
			}
		} else {
			if err := q.UpdateLeadFields(ctx, db.UpdateLeadFieldsParams{
				ID:          leadID,
				CampaignID:  campaignID,
				Name:        name,
				Email:       nullable(email),
				CompanyID:   companyID,
				Title:       nullable(get(row, "Job Title")),
				Score:       nullableFloat32(get(row, "Total Score")),
				LinkedinUrl: nullable(linkedinURL),
				Location:    nullable(get(row, "Location")),
				Phone:       nullable(get(row, "Phone")),
				SourcedAt:   sourcedAt,
			}); err != nil {
				return nil, fmt.Errorf("update lead %d: %w", leadID, err)
			}
			summary.Rows = append(summary.Rows, importer.ImportedRow{ID: leadID, Name: name, Company: companyName, Action: "updated"})
		}

		// Store raw intent as a note
		if intentRaw != "" {
			clean := intentRe.ReplaceAllString(intentRaw, "")
			clean = strings.TrimSpace(clean)
			if clean != "" {
				if _, err := q.CreateNote(ctx, db.CreateNoteParams{
					LeadID:  leadID,
					Content: fmt.Sprintf("[Gojiberry] %s", clean),
				}); err != nil {
					return nil, fmt.Errorf("create note: %w", err)
				}
			}
		}
	}

	out, _ := json.Marshal(struct {
		gojiberrySummary
		Method string `json:"method"`
	}{summary, "csv"})
	return mcp.NewToolResultText(string(out)), nil
}

// registerImportLeads imports leads from a Gojiberry CSV string. Accepts an
// optional campaign_id to target a specific campaign; defaults to the startup campaign.
func registerImportLeads(s *server.MCPServer, q *db.Queries, defaultCampaignID int32, llm LLMConfig) {
	tool := mcp.NewTool("import_leads",
		mcp.WithDescription(`DEPRECATED. Use import_csv for all imports.`),
		mcp.WithString("csv",
			mcp.Description("Raw CSV content including the header row. Max 10 data rows per batch."),
			mcp.Required(),
		),
		mcp.WithNumber("campaign_id",
			mcp.Description("Campaign to import leads into. Defaults to the current campaign."),
		),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		csvContent, ok := req.Params.Arguments["csv"].(string)
		if !ok || csvContent == "" {
			return mcp.NewToolResultError("csv is required"), nil
		}
		campaignID := defaultCampaignID
		if v, ok := req.Params.Arguments["campaign_id"].(float64); ok && v > 0 {
			campaignID = int32(v)
		}
		return importCSV(ctx, q, csvContent, campaignID, llm)
	})
}

// registerImportLeadsFile imports leads from a Gojiberry CSV file on disk.
// Accepts an optional campaign_id to target a specific campaign; defaults to the startup campaign.
func registerImportLeadsFile(s *server.MCPServer, q *db.Queries, defaultCampaignID int32, llm LLMConfig) {
	tool := mcp.NewTool("import_leads_file",
		mcp.WithDescription("RARELY NEEDED. Only use when the file is physically on the MCP server's filesystem. If the user gave you a file or you can read it yourself, use import_leads with the text content instead. GOJIBERRY FORMAT ONLY."),
		mcp.WithString("path",
			mcp.Description("Absolute path to the CSV file"),
			mcp.Required(),
		),
		mcp.WithNumber("campaign_id",
			mcp.Description("Campaign to import leads into. Defaults to the current campaign."),
		),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		path, ok := req.Params.Arguments["path"].(string)
		if !ok || path == "" {
			return mcp.NewToolResultError("path is required"), nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf(
				"cannot read %q from the MCP server filesystem: %v\n\nIf this file is on your local machine (not the server), read its contents yourself and call import_leads with the csv parameter instead.",
				path, err,
			)), nil
		}
		campaignID := defaultCampaignID
		if v, ok := req.Params.Arguments["campaign_id"].(float64); ok && v > 0 {
			campaignID = int32(v)
		}
		return importCSV(ctx, q, string(data), campaignID, llm)
	})
}
