package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	db "github.com/vitruviantech/outreacher/internal/db/gen"
)

// registerSearchCompanies searches companies within a campaign with optional filters.
// All filter args are optional; omitting them returns all companies (up to 50).
func registerSearchCompanies(s *server.MCPServer, q *db.Queries, defaultCampaignID int32) {
	tool := mcp.NewTool("search_companies",
		mcp.WithDescription("Search companies within the current campaign. All filters are optional. Returns up to 50 results."),
		mcp.WithString("query",
			mcp.Description("Partial name match (case-insensitive)"),
		),
		mcp.WithString("industry",
			mcp.Description("Industry substring to filter by"),
		),
		mcp.WithString("funding_stage",
			mcp.Description("Funding stage to filter by (e.g. 'Seed', 'Series A')"),
		),
		mcp.WithString("is_vc",
			mcp.Description("Filter to VC firms only: 'true' or 'false'. Omit to return all."),
		),
		mcp.WithString("is_hiring",
			mcp.Description("Filter to companies currently hiring: 'true' or 'false'. Omit to return all."),
		),
		mcp.WithNumber("campaign_id",
			mcp.Description("Campaign to search within. Defaults to the current campaign."),
		),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		campaignID := defaultCampaignID
		if v, ok := req.Params.Arguments["campaign_id"].(float64); ok && v > 0 {
			campaignID = int32(v)
		}

		strArg := func(name string) string {
			v, _ := req.Params.Arguments[name].(string)
			return v
		}

		companies, err := q.SearchCompanies(ctx, db.SearchCompaniesParams{
			CampaignID: campaignID,
			Column2:    strArg("query"),
			Column3:    strArg("is_vc"),
			Column4:    strArg("funding_stage"),
			Column5:    strArg("industry"),
			Column6:    strArg("is_hiring"),
		})
		if err != nil {
			return nil, fmt.Errorf("search companies: %w", err)
		}
		if companies == nil {
			companies = []db.SearchCompaniesRow{}
		}

		out, err := json.Marshal(companies)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(string(out)), nil
	})
}

// registerGetCompany fetches full company detail including all enriched fields
// and the intel JSON blob. Scoped to the active campaign.
func registerGetCompany(s *server.MCPServer, q *db.Queries, defaultCampaignID int32) {
	tool := mcp.NewTool("get_company",
		mcp.WithDescription("Get full details for a single company, including all enriched fields and AI-generated intel."),
		mcp.WithNumber("id",
			mcp.Description("Company ID"),
			mcp.Required(),
		),
		mcp.WithNumber("campaign_id",
			mcp.Description("Campaign the company belongs to. Defaults to the current campaign."),
		),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, ok := req.Params.Arguments["id"].(float64)
		if !ok {
			return mcp.NewToolResultError("id is required"), nil
		}
		campaignID := defaultCampaignID
		if v, ok := req.Params.Arguments["campaign_id"].(float64); ok && v > 0 {
			campaignID = int32(v)
		}

		company, err := q.GetCompany(ctx, db.GetCompanyParams{
			ID:         int32(id),
			CampaignID: campaignID,
		})
		if err != nil {
			return nil, fmt.Errorf("get company: %w", err)
		}

		out, err := json.Marshal(company)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(string(out)), nil
	})
}
