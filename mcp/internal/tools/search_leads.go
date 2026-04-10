package tools

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	db "github.com/vitruviantech/outreacher/internal/db/gen"
)

// registerSearchLeads searches leads within a campaign by name/email, status, and
// company. All filters are optional and AND-ed. campaign_id arg overrides the
// startup default so Claude can target a specific campaign mid-session.
func registerSearchLeads(s *server.MCPServer, q *db.Queries, defaultCampaignID int32) {
	tool := mcp.NewTool("search_leads",
		mcp.WithDescription("Search leads by name, email, status, or company. All filters are optional and AND-ed together."),
		mcp.WithString("query",
			mcp.Description("Search by name or email (case-insensitive substring)"),
		),
		mcp.WithString("status",
			mcp.Description("Filter by lead status"),
			mcp.Enum("new", "contacted", "qualified", "disqualified", "converted"),
		),
		mcp.WithString("company",
			mcp.Description("Filter by company name (substring)"),
		),
		mcp.WithNumber("campaign_id",
			mcp.Description("Campaign to search within. Defaults to the current campaign."),
		),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.Params.Arguments

		strArg := func(key string) string {
			if v, ok := args[key].(string); ok {
				return v
			}
			return ""
		}

		campaignID := defaultCampaignID
		if v, ok := args["campaign_id"].(float64); ok && v > 0 {
			campaignID = int32(v)
		}

		rows, err := q.SearchLeads(ctx, db.SearchLeadsParams{
			CampaignID: campaignID,
			Column2:    strArg("query"),
			Column3:    strArg("status"),
			Column4:    strArg("company"),
		})
		if err != nil {
			return nil, err
		}
		if rows == nil {
			rows = []db.SearchLeadsRow{}
		}

		out, err := json.Marshal(rows)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(string(out)), nil
	})
}
