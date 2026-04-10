package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	db "github.com/vitruviantech/outreacher/internal/db/gen"
)

// registerGetLead fetches full details for a single lead including company info
// and all notes. The lead must belong to the active campaign (prevents cross-campaign
// data leakage). campaign_id arg overrides the startup default.
func registerGetLead(s *server.MCPServer, q *db.Queries, defaultCampaignID int32) {
	tool := mcp.NewTool("get_lead",
		mcp.WithDescription("Get full details for a single lead, including company info and all notes."),
		mcp.WithNumber("id",
			mcp.Description("Lead ID"),
			mcp.Required(),
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

		campaignID := defaultCampaignID
		if v, ok := req.Params.Arguments["campaign_id"].(float64); ok && v > 0 {
			campaignID = int32(v)
		}

		lead, err := q.GetLead(ctx, db.GetLeadParams{
			ID:         int32(id),
			CampaignID: campaignID,
		})
		if err != nil {
			return nil, fmt.Errorf("get lead: %w", err)
		}

		notes, err := q.GetNotesByLead(ctx, lead.ID)
		if err != nil {
			return nil, fmt.Errorf("get notes: %w", err)
		}

		result := map[string]any{
			"id":         lead.ID,
			"name":       lead.Name,
			"email":      lead.Email,
			"title":      lead.Title,
			"status":     lead.Status,
			"score":      lead.Score,
			"company_id": lead.CompanyID,
			"company":    lead.Company,
			"domain":     lead.Domain,
			"industry":   lead.Industry,
			"notes":      notes,
		}

		out, err := json.Marshal(result)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(string(out)), nil
	})
}
