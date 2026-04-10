package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	db "github.com/vitruviantech/outreacher/internal/db/gen"
	"github.com/vitruviantech/outreacher/internal/importer"
)

// registerImportCSV registers the import_csv tool, which accepts CSV content
// and routes to LLM-based extraction when an API key is configured, falling
// back to format-specific parsers otherwise.
func registerImportCSV(s *server.MCPServer, q *db.Queries, defaultCampaignID int32, llm LLMConfig) {
	tool := mcp.NewTool("import_csv",
		mcp.WithDescription("Import CSV data. Use for ALL imports. When the user gives you a file, call this tool immediately without any prior output or analysis. Read the ENTIRE file using your file-read tool (not head/tail/bash/cat) and pass the raw bytes verbatim — do NOT reformat, reorder, add, remove, or modify any fields, columns, or rows. CSV rows may span multiple lines due to quoted fields; line-based tools will corrupt the data. The server handles all parsing and extraction. Pass the entire file in one call; do not batch."),
		mcp.WithString("format",
			mcp.Description("Source format (gojiberry, revli_startup_contacts, revli_investor_contacts, revli_startup_companies, revli_investor_companies). Omit to auto-detect from headers."),
		),
		mcp.WithString("csv",
			mcp.Description("Raw CSV content including the header row. No size limit — pass the full file content."),
			mcp.Required(),
		),
		mcp.WithNumber("campaign_id",
			mcp.Description("Campaign to import into. Defaults to the current campaign."),
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
		return runImport(ctx, q, csvContent, req.Params.Arguments["format"], campaignID, llm)
	})
}

// registerImportCSVFile registers import_csv_file, which reads a CSV from disk
// and imports it using the same extraction path as import_csv.
func registerImportCSVFile(s *server.MCPServer, q *db.Queries, defaultCampaignID int32, llm LLMConfig) {
	tool := mcp.NewTool("import_csv_file",
		mcp.WithDescription("DEPRECATED. Use import_csv for all imports."),
		mcp.WithString("format",
			mcp.Description("Source format (gojiberry, revli_startup_contacts, revli_investor_contacts, revli_startup_companies, revli_investor_companies). Omit to auto-detect from headers."),
		),
		mcp.WithString("path",
			mcp.Description("Absolute path to the CSV file on disk"),
			mcp.Required(),
		),
		mcp.WithNumber("campaign_id",
			mcp.Description("Campaign to import into. Defaults to the current campaign."),
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
				"cannot read %q from the MCP server filesystem: %v\n\nIf this file is on your local machine (not the server), read its contents yourself and call import_csv with the csv parameter instead.",
				path, err,
			)), nil
		}
		campaignID := defaultCampaignID
		if v, ok := req.Params.Arguments["campaign_id"].(float64); ok && v > 0 {
			campaignID = int32(v)
		}
		return runImport(ctx, q, string(data), req.Params.Arguments["format"], campaignID, llm)
	})
}

// runImport is the shared dispatch used by both import tools. Known formats
// (gojiberry, revli_*) always use the fast direct parsers. LLM extraction is
// only used as a fallback for CSV files whose format cannot be auto-detected.
func runImport(ctx context.Context, q *db.Queries, csvContent string, formatArg any, campaignID int32, llm LLMConfig) (*mcp.CallToolResult, error) {
	// Resolve format: explicit arg → auto-detect → unknown.
	format, _ := formatArg.(string)
	if format == "" {
		var detectErr error
		format, detectErr = importer.DetectFormat(csvContent)
		firstLine := csvContent
		if i := strings.Index(csvContent, "\n"); i >= 0 {
			firstLine = csvContent[:i]
		}
		if len(firstLine) > 120 {
			firstLine = firstLine[:120]
		}
		log.Printf("[import] detect format=%q err=%v firstLine=%q", format, detectErr, firstLine)
	}

	// Known formats: use the fast direct parser regardless of LLM config.
	if format != "" {

	switch format {
	case "gojiberry":
		return importCSV(ctx, q, csvContent, campaignID, llm)

	case "revli_startup_contacts":
		rows, err := importer.ParseRevliStartupContacts(csvContent)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("parse: %v", err)), nil
		}
		summary, err := importer.Write(ctx, q, rows, campaignID)
		if err != nil {
			return nil, err
		}
		out, _ := json.Marshal(summary)
		return mcp.NewToolResultText(string(out)), nil

	case "revli_investor_contacts":
		rows, err := importer.ParseRevliInvestorContacts(csvContent)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("parse: %v", err)), nil
		}
		summary, err := importer.Write(ctx, q, rows, campaignID)
		if err != nil {
			return nil, err
		}
		out, _ := json.Marshal(summary)
		return mcp.NewToolResultText(string(out)), nil

	case "revli_startup_companies":
		rows, err := importer.ParseRevliStartupCompanies(csvContent)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("parse: %v", err)), nil
		}
		summary, err := importer.Write(ctx, q, rows, campaignID)
		if err != nil {
			return nil, err
		}
		out, _ := json.Marshal(summary)
		return mcp.NewToolResultText(string(out)), nil

	case "revli_investor_companies":
		rows, err := importer.ParseRevliInvestorCompanies(csvContent)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("parse: %v", err)), nil
		}
		summary, err := importer.Write(ctx, q, rows, campaignID)
		if err != nil {
			return nil, err
		}
		out, _ := json.Marshal(summary)
		return mcp.NewToolResultText(string(out)), nil
	}
	}

	// Unknown format: fall back to LLM extraction if an API key is configured.
	if llm.ApiKey != "" {
		rows, err := importer.Extract(ctx, llm.Provider, llm.ApiKey, llm.Model, csvContent)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("llm extraction: %v", err)), nil
		}
		summary, err := importer.Write(ctx, q, rows, campaignID)
		if err != nil {
			return nil, err
		}
		out, _ := json.Marshal(struct {
			importer.ImportSummary
			Method string `json:"method"`
		}{summary, llm.Provider})
		return mcp.NewToolResultText(string(out)), nil
	}

	return mcp.NewToolResultError("could not detect CSV format — pass format explicitly (gojiberry, revli_startup_contacts, revli_investor_contacts, revli_startup_companies, revli_investor_companies)"), nil
}
