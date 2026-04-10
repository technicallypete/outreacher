package tools

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	db "github.com/vitruviantech/outreacher/internal/db/gen"
)

// registerUpdateLeadStatus advances a lead through the status flow. The lead
// must belong to the active campaign. campaign_id arg overrides the startup default.
func registerUpdateLeadStatus(s *server.MCPServer, q *db.Queries, defaultCampaignID int32) {
	tool := mcp.NewTool("update_lead_status",
		mcp.WithDescription("Update the status of a lead."),
		mcp.WithNumber("id",
			mcp.Description("Lead ID"),
			mcp.Required(),
		),
		mcp.WithString("status",
			mcp.Description("New status"),
			mcp.Required(),
			mcp.Enum("new", "contacted", "qualified", "disqualified", "converted"),
		),
		mcp.WithNumber("campaign_id",
			mcp.Description("Campaign the lead belongs to. Defaults to the current campaign."),
		),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id, ok := req.Params.Arguments["id"].(float64)
		if !ok {
			return mcp.NewToolResultError("id is required"), nil
		}
		status, ok := req.Params.Arguments["status"].(string)
		if !ok {
			return mcp.NewToolResultError("status is required"), nil
		}

		campaignID := defaultCampaignID
		if v, ok := req.Params.Arguments["campaign_id"].(float64); ok && v > 0 {
			campaignID = int32(v)
		}

		row, err := q.UpdateLeadStatus(ctx, db.UpdateLeadStatusParams{
			ID:         int32(id),
			CampaignID: campaignID,
			Status:     db.AppLeadStatus(status),
		})
		if err != nil {
			return nil, err
		}

		out, err := json.Marshal(row)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(string(out)), nil
	})
}
