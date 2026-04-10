package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	db "github.com/vitruviantech/outreacher/internal/db/gen"
)

// RegisterCampaignTools registers campaign management tools. These are only
// used by the stdio binary (Claude Desktop) where a single user manages
// multiple campaigns within one org. The SSE server omits these because
// campaign CRUD is handled by the REST API in the web app.
func RegisterCampaignTools(s *server.MCPServer, q *db.Queries, defaultOrgID, defaultCampaignID int32) {
	registerGetCampaign(s, q, defaultCampaignID)
	registerListCampaigns(s, q, defaultOrgID)
	registerCreateCampaign(s, q, defaultOrgID)
	registerRenameCampaign(s, q, defaultCampaignID)
}

func registerGetCampaign(s *server.MCPServer, q *db.Queries, defaultCampaignID int32) {
	tool := mcp.NewTool("get_campaign",
		mcp.WithDescription("Get details (id, name, slug) for the current campaign, or a specified one."),
		mcp.WithNumber("campaign_id",
			mcp.Description("Campaign ID. Defaults to the current campaign."),
		),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		campaignID := defaultCampaignID
		if v, ok := req.Params.Arguments["campaign_id"].(float64); ok && v > 0 {
			campaignID = int32(v)
		}
		campaign, err := q.GetCampaign(ctx, campaignID)
		if err != nil {
			return nil, fmt.Errorf("get campaign: %w", err)
		}
		out, err := json.Marshal(campaign)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(string(out)), nil
	})
}

func registerListCampaigns(s *server.MCPServer, q *db.Queries, defaultOrgID int32) {
	tool := mcp.NewTool("list_campaigns",
		mcp.WithDescription("List all campaigns for the current organization, or a specified org."),
		mcp.WithNumber("org_id",
			mcp.Description("Organization ID to list campaigns for. Defaults to the current org."),
		),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		orgID := defaultOrgID
		if v, ok := req.Params.Arguments["org_id"].(float64); ok && v > 0 {
			orgID = int32(v)
		}
		campaigns, err := q.ListCampaigns(ctx, orgID)
		if err != nil {
			return nil, err
		}
		if campaigns == nil {
			campaigns = []db.ListCampaignsRow{}
		}
		out, err := json.Marshal(campaigns)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(string(out)), nil
	})
}

func registerCreateCampaign(s *server.MCPServer, q *db.Queries, defaultOrgID int32) {
	tool := mcp.NewTool("create_campaign",
		mcp.WithDescription("Create a new campaign under the current organization (or a specified org)."),
		mcp.WithString("name",
			mcp.Description("Campaign name"),
			mcp.Required(),
		),
		mcp.WithString("slug",
			mcp.Description("URL-safe identifier (auto-derived from name if omitted)"),
		),
		mcp.WithNumber("org_id",
			mcp.Description("Organization ID. Defaults to the current org."),
		),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, ok := req.Params.Arguments["name"].(string)
		if !ok || name == "" {
			return mcp.NewToolResultError("name is required"), nil
		}
		slug, _ := req.Params.Arguments["slug"].(string)
		if slug == "" {
			slug = slugify(name)
		}
		orgID := defaultOrgID
		if v, ok := req.Params.Arguments["org_id"].(float64); ok && v > 0 {
			orgID = int32(v)
		}

		campaign, err := q.CreateCampaign(ctx, db.CreateCampaignParams{
			OrganizationID: orgID,
			Name:           name,
			Slug:           slug,
			IsDefault:      false,
		})
		if err != nil {
			return nil, fmt.Errorf("create campaign: %w", err)
		}

		out, err := json.Marshal(campaign)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(string(out)), nil
	})
}

func registerRenameCampaign(s *server.MCPServer, q *db.Queries, defaultCampaignID int32) {
	tool := mcp.NewTool("rename_campaign",
		mcp.WithDescription("Rename a campaign. The slug is kept unchanged so existing references stay stable."),
		mcp.WithString("name",
			mcp.Description("New display name for the campaign"),
			mcp.Required(),
		),
		mcp.WithNumber("campaign_id",
			mcp.Description("Campaign to rename. Defaults to the current campaign."),
		),
	)
	s.AddTool(tool, func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name, ok := req.Params.Arguments["name"].(string)
		if !ok || name == "" {
			return mcp.NewToolResultError("name is required"), nil
		}
		campaignID := defaultCampaignID
		if v, ok := req.Params.Arguments["campaign_id"].(float64); ok && v > 0 {
			campaignID = int32(v)
		}

		updated, err := q.RenameCampaign(ctx, db.RenameCampaignParams{
			ID:   campaignID,
			Name: name,
		})
		if err != nil {
			return nil, fmt.Errorf("rename campaign: %w", err)
		}

		out, err := json.Marshal(updated)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(string(out)), nil
	})
}
