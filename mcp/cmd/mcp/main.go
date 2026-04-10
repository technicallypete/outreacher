package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/server"
	dbpkg "github.com/vitruviantech/outreacher/internal/db"
	dbgen "github.com/vitruviantech/outreacher/internal/db/gen"
	"github.com/vitruviantech/outreacher/internal/tenant"
	"github.com/vitruviantech/outreacher/internal/tools"
)

func main() {
	var (
		databaseURL     string
		llmProvider     string
		openaiApiKey    string
		anthropicApiKey string
		llmModel        string
	)
	flag.StringVar(&databaseURL, "database-url", "", "PostgreSQL connection string (overrides DATABASE_URL env var)")
	flag.StringVar(&llmProvider, "llm-provider", "", "LLM provider: openai or anthropic (overrides MCP_LLM_PROVIDER env var, default openai)")
	flag.StringVar(&openaiApiKey, "openai-api-key", "", "OpenAI API key (overrides OPENAI_API_KEY env var)")
	flag.StringVar(&anthropicApiKey, "anthropic-api-key", "", "Anthropic API key (overrides ANTHROPIC_API_KEY env var)")
	flag.StringVar(&llmModel, "llm-model", "", "LLM model name (overrides MCP_LLM_MODEL env var)")
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

	ctx := context.Background()

	pool, err := dbpkg.NewPool(ctx, databaseURL)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer pool.Close()

	q := dbgen.New(pool)

	t, err := tenant.Bootstrap(ctx, q)
	if err != nil {
		log.Fatalf("tenant bootstrap: %v", err)
	}

	s := server.NewMCPServer("Outreacher", "0.1.0")
	tools.RegisterCampaignTools(s, q, t.OrgID, t.CampaignID)
	tools.Register(s, q, t.CampaignID, tools.LLMConfig{
		Provider: llmProvider,
		ApiKey:   llmApiKey,
		Model:    llmModel,
	})

	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("stdio: %v", err)
	}
}
