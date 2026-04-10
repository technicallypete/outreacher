package tools

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerDebugEnv registers a tool that reports which expected env vars are
// set (names and whether they have a value — never the values themselves).
// Remove this tool once configuration is confirmed working.
func registerDebugEnv(s *server.MCPServer) {
	tool := mcp.NewTool("debug_env",
		mcp.WithDescription("Reports which environment variables are set in the MCP server process. Use this to verify configuration."),
	)
	s.AddTool(tool, func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		vars := []string{
			"DATABASE_URL",
			"DATABASE_ADMIN_URL",
			"MCP_URL",
			"MCP_ORG_SLUG",
			"MCP_USER_SLUG",
			"MCP_LLM_PROVIDER",
			"MCP_LLM_MODEL",
			"ANTHROPIC_API_KEY",
			"OPENAI_API_KEY",
		}
		result := make(map[string]string, len(vars))
		for _, k := range vars {
			v := os.Getenv(k)
			if v == "" {
				result[k] = "NOT SET"
			} else {
				// Show first 4 chars only so the user can confirm which key/URL
				// is loaded without exposing secrets.
				preview := v
				if len(preview) > 4 {
					preview = v[:4] + strings.Repeat("*", min(len(v)-4, 8))
				}
				result[k] = "set (" + preview + ")"
			}
		}
		out, _ := json.Marshal(result)
		return mcp.NewToolResultText(string(out)), nil
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
