package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	db "github.com/vitruviantech/outreacher/internal/db/gen"
)

// registerCreateFollowupNote appends a follow-up note to a lead. The lead must
// belong to the active campaign (verified via GetLead). campaign_id arg overrides
// the startup default so Claude can target a specific campaign mid-session.
func registerCreateFollowupNote(s *server.MCPServer, q *db.Queries, defaultCampaignID int32) {
	tool := mcp.NewTool("create_followup_note",
		mcp.WithDescription("Append a follow-up note to a lead."),
		mcp.WithNumber("lead_id",
			mcp.Description("Lead ID"),
			mcp.Required(),
		),
		mcp.WithString("content",
			mcp.Description("Note content"),
			mcp.Required(),
		),
		mcp.WithNumber("campaign_id",
			mcp.Description("Campaign the lead belongs to. Defaults to the current campaign."),
		),
	)

	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		leadID, ok := req.Params.Arguments["lead_id"].(float64)
		if !ok {
			return mcp.NewToolResultError("lead_id is required"), nil
		}
		content, ok := req.Params.Arguments["content"].(string)
		if !ok || content == "" {
			return mcp.NewToolResultError("content is required"), nil
		}

		campaignID := defaultCampaignID
		if v, ok := req.Params.Arguments["campaign_id"].(float64); ok && v > 0 {
			campaignID = int32(v)
		}

		// Verify the lead belongs to the campaign before appending a note.
		if _, err := q.GetLead(ctx, db.GetLeadParams{
			ID:         int32(leadID),
			CampaignID: campaignID,
		}); err != nil {
			return nil, fmt.Errorf("lead %d not found in campaign %d: %w", int32(leadID), campaignID, err)
		}

		note, err := q.CreateNote(ctx, db.CreateNoteParams{
			LeadID:  int32(leadID),
			Content: content,
		})
		if err != nil {
			return nil, err
		}

		out, err := json.Marshal(note)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(string(out)), nil
	})
}
