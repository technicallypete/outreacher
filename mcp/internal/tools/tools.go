package tools

import (
	"github.com/mark3labs/mcp-go/server"
	db "github.com/vitruviantech/outreacher/internal/db/gen"
)

// LLMConfig holds provider/key/model settings for LLM-assisted import extraction.
// ApiKey is optional; when empty, import tools fall back to format-specific CSV parsers.
type LLMConfig struct {
	Provider string
	ApiKey   string
	Model    string
}

// Register adds all MCP tools to the server. campaignID is the default operating
// campaign resolved at startup; domain tools use it when no campaign_id arg is given.
// Org/user/campaign CRUD is handled exclusively by the REST API, not MCP tools.
func Register(s *server.MCPServer, q *db.Queries, campaignID int32, llm LLMConfig) {
	registerSearchLeads(s, q, campaignID)
	registerGetLead(s, q, campaignID)
	registerUpdateLeadStatus(s, q, campaignID)
	registerCreateFollowupNote(s, q, campaignID)
	registerImportLeads(s, q, campaignID, llm)
	registerImportLeadsFile(s, q, campaignID, llm)
	registerImportCSV(s, q, campaignID, llm)
	registerImportCSVFile(s, q, campaignID, llm)
	registerSearchCompanies(s, q, campaignID)
	registerGetCompany(s, q, campaignID)
	registerDebugEnv(s)
}
