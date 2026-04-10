package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/server"
	dbpkg "github.com/vitruviantech/outreacher/internal/db"
	dbgen "github.com/vitruviantech/outreacher/internal/db/gen"
	"github.com/vitruviantech/outreacher/internal/tools"
)

func bearerAuth(apiKey string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if token != apiKey {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	var (
		databaseURL     string
		llmProvider     string
		openaiApiKey    string
		anthropicApiKey string
		llmModel        string
		port            string
		baseURL         string
		mcpAPIKey       string
	)
	flag.StringVar(&databaseURL, "database-url", "", "PostgreSQL connection string (overrides DATABASE_URL env var)")
	flag.StringVar(&llmProvider, "llm-provider", "", "LLM provider: openai or anthropic (overrides MCP_LLM_PROVIDER env var, default openai)")
	flag.StringVar(&openaiApiKey, "openai-api-key", "", "OpenAI API key (overrides OPENAI_API_KEY env var)")
	flag.StringVar(&anthropicApiKey, "anthropic-api-key", "", "Anthropic API key (overrides ANTHROPIC_API_KEY env var)")
	flag.StringVar(&llmModel, "llm-model", "", "LLM model name (overrides MCP_LLM_MODEL env var)")
	flag.StringVar(&port, "port", "", "HTTP port to listen on (overrides MCP_PORT env var, default 3001)")
	flag.StringVar(&baseURL, "base-url", "", "Public base URL (overrides MCP_URL env var)")
	flag.StringVar(&mcpAPIKey, "mcp-api-key", "", "Bearer token for HTTP auth (overrides MCP_API_KEY env var)")
	flag.Parse()

	// Env var fallbacks.
	if llmProvider == "" {
		llmProvider = os.Getenv("MCP_LLM_PROVIDER")
	}
	if llmProvider == "" {
		llmProvider = "openai"
	}
	if openaiApiKey == "" {
		openaiApiKey = os.Getenv("OPENAI_API_KEY")
	}
	if anthropicApiKey == "" {
		anthropicApiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	var llmApiKey string
	if llmProvider == "openai" {
		llmApiKey = openaiApiKey
	} else {
		llmApiKey = anthropicApiKey
	}
	if llmModel == "" {
		llmModel = os.Getenv("MCP_LLM_MODEL")
	}
	if port == "" {
		port = os.Getenv("MCP_PORT")
	}
	if port == "" {
		port = "3001"
	}
	if baseURL == "" {
		baseURL = os.Getenv("MCP_URL")
	}
	if baseURL == "" {
		baseURL = fmt.Sprintf("http://localhost:%s", port)
	}
	if mcpAPIKey == "" {
		mcpAPIKey = os.Getenv("MCP_API_KEY")
	}

	ctx := context.Background()

	pool, err := dbpkg.NewPool(ctx, databaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	q := dbgen.New(pool)

	s := server.NewMCPServer("Outreacher", "0.1.0")
	// All domain tool calls carry an explicit campaign_id from the session.
	// Org/user/campaign CRUD is handled by the REST API, not MCP tools.
	tools.Register(s, q, 0, tools.LLMConfig{
		Provider: llmProvider,
		ApiKey:   llmApiKey,
		Model:    llmModel,
	})

	sseServer := server.NewSSEServer(s, server.WithBaseURL(baseURL))

	var handler http.Handler = sseServer
	if mcpAPIKey != "" {
		handler = bearerAuth(mcpAPIKey, sseServer)
		log.Printf("MCP API key auth enabled")
	} else {
		log.Printf("Warning: no MCP API key set — server is unauthenticated")
	}

	addr := ":" + port
	log.Printf("MCP SSE server listening on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("http: %v", err)
	}
}
