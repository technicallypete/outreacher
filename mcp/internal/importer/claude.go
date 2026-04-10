package importer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

const extractBatchSize = 10

// extractedRow is the JSON shape Claude returns for each contact.
type extractedRow struct {
	FirstName   string  `json:"first_name"`
	LastName    string  `json:"last_name"`
	Email       string  `json:"email"`
	LinkedinURL string  `json:"linkedin_url"`
	Company     string  `json:"company"`
	CompanyURL  string  `json:"company_linkedin_url"`
	Website     string  `json:"website"`
	Industry    string  `json:"industry"`
	JobTitle    string  `json:"job_title"`
	Location    string  `json:"location"`
	Phone       string  `json:"phone"`
	Score       float32 `json:"score"`
}

// extractionResult is the top-level shape returned by the tool call.
type extractionResult struct {
	Rows []extractedRow `json:"rows"`
}

// extractionToolSchema is the JSON schema passed to Claude as a tool definition.
var extractionToolSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"rows": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"first_name":          map[string]any{"type": "string"},
					"last_name":           map[string]any{"type": "string"},
					"email":               map[string]any{"type": "string", "description": "Email address, or empty string if not present"},
					"linkedin_url":        map[string]any{"type": "string", "description": "LinkedIn profile URL from the data (any format: slug-based or member ID such as /in/ACoAAA...), or empty string if not present"},
					"company":             map[string]any{"type": "string", "description": "Company name, or empty string"},
					"company_linkedin_url": map[string]any{"type": "string", "description": "LinkedIn company page URL, or empty string"},
					"website":             map[string]any{"type": "string", "description": "Company website URL, or empty string"},
					"industry":            map[string]any{"type": "string", "description": "Industry, or empty string"},
					"job_title":           map[string]any{"type": "string", "description": "Job title, or empty string"},
					"location":            map[string]any{"type": "string", "description": "Location (city, state, country), or empty string"},
					"phone":               map[string]any{"type": "string", "description": "Phone number, or empty string"},
					"score":               map[string]any{"type": "number", "description": "Numeric lead score, or 0 if not present"},
				},
				"required": []string{
					"first_name", "last_name", "email", "linkedin_url",
					"company", "company_linkedin_url", "website", "industry",
					"job_title", "location", "phone", "score",
				},
			},
		},
	},
	"required": []string{"rows"},
}

// DefaultAnthropicModel is used when provider is anthropic and no model is specified.
const DefaultAnthropicModel = string(anthropic.ModelClaudeHaiku4_5)

// Extract routes to the appropriate LLM provider for contact extraction.
// provider must be "openai" or "anthropic" (default). model defaults per provider.
// Large CSVs are automatically split into batches of extractBatchSize rows and
// extracted in parallel to keep individual API calls fast.
func Extract(ctx context.Context, provider, apiKey, model, content string) ([]Row, error) {
	lines := strings.Split(content, "\n")
	if len(lines) < 2 {
		return nil, nil
	}
	header := lines[0]

	// Collect non-empty data lines.
	var dataLines []string
	for _, l := range lines[1:] {
		if strings.TrimSpace(l) != "" {
			dataLines = append(dataLines, l)
		}
	}
	if len(dataLines) == 0 {
		return nil, nil
	}

	// Small enough to extract in one shot.
	if len(dataLines) <= extractBatchSize {
		return extractOne(ctx, provider, apiKey, model, content)
	}

	// Build batches: each is header + up to extractBatchSize data rows.
	var batches []string
	for i := 0; i < len(dataLines); i += extractBatchSize {
		end := i + extractBatchSize
		if end > len(dataLines) {
			end = len(dataLines)
		}
		batches = append(batches, strings.Join(append([]string{header}, dataLines[i:end]...), "\n"))
	}

	// Extract all batches in parallel.
	type result struct {
		rows []Row
		err  error
	}
	results := make([]result, len(batches))
	var wg sync.WaitGroup
	for i, batch := range batches {
		wg.Add(1)
		go func(i int, b string) {
			defer wg.Done()
			rows, err := extractOne(ctx, provider, apiKey, model, b)
			results[i] = result{rows, err}
		}(i, batch)
	}
	wg.Wait()

	var all []Row
	for _, r := range results {
		if r.err != nil {
			return nil, r.err
		}
		all = append(all, r.rows...)
	}
	return all, nil
}

// extractOne sends a single CSV chunk (header + ≤extractBatchSize rows) to the LLM.
func extractOne(ctx context.Context, provider, apiKey, model, content string) ([]Row, error) {
	if provider == "openai" {
		return ExtractWithOpenAI(ctx, apiKey, model, content)
	}
	return ExtractWithClaude(ctx, apiKey, model, content)
}

// ExtractWithClaude calls an Anthropic model to extract structured contact rows
// from arbitrary CSV or tabular text content. It uses tool_use to get reliable
// JSON output, replicating the parsing intelligence Claude applies when reading
// files. apiKey must be a valid Anthropic API key. model defaults to
// DefaultLLMModel if empty.
func ExtractWithClaude(ctx context.Context, apiKey, model, content string) ([]Row, error) {
	if model == "" {
		model = DefaultAnthropicModel
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	schemaBytes, err := json.Marshal(extractionToolSchema)
	if err != nil {
		return nil, fmt.Errorf("marshal schema: %w", err)
	}
	var schemaParam anthropic.ToolInputSchemaParam
	if err := json.Unmarshal(schemaBytes, &schemaParam); err != nil {
		return nil, fmt.Errorf("unmarshal schema param: %w", err)
	}

	msg, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: 8192,
		Tools: []anthropic.ToolUnionParam{
			{OfTool: &anthropic.ToolParam{
				Name:        "extract_contacts",
				Description: anthropic.String("Extract all contact records from the provided data."),
				InputSchema: schemaParam,
			}},
		},
		ToolChoice: anthropic.ToolChoiceUnionParam{
			OfTool: &anthropic.ToolChoiceToolParam{
				Name: "extract_contacts",
			},
		},
		Messages: []anthropic.MessageParam{
			{
				Role: anthropic.MessageParamRoleUser,
				Content: []anthropic.ContentBlockParamUnion{
					anthropic.NewTextBlock(fmt.Sprintf(
						"Extract all contacts from the following data. "+
							"For linkedin_url, use any LinkedIn profile URL present in the data — "+
							"including member ID format URLs like https://www.linkedin.com/in/ACoAAA... "+
							"Strip any HTML tags from field values. "+
							"Use empty string for any field that is not present.\n\n%s",
						content,
					)),
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("claude api: %w", err)
	}

	// Find the tool_use block in the response.
	var result extractionResult
	for _, block := range msg.Content {
		if block.Type == "tool_use" {
			inputBytes, err := json.Marshal(block.Input)
			if err != nil {
				return nil, fmt.Errorf("marshal tool input: %w", err)
			}
			if err := json.Unmarshal(inputBytes, &result); err != nil {
				return nil, fmt.Errorf("unmarshal extraction result: %w", err)
			}
			break
		}
	}

	return toRows(result.Rows), nil
}

// toRows converts Claude's extracted rows into the importer Row type.
func toRows(extracted []extractedRow) []Row {
	rows := make([]Row, 0, len(extracted))
	for _, e := range extracted {
		row := Row{}

		if e.Company != "" {
			row.Company = &NormalizedCompany{
				Name:        e.Company,
				Domain:      strPtr(e.Website),
				Industry:    strPtr(e.Industry),
				LinkedinURL: strPtr(e.CompanyURL),
			}
		}

		lead := &NormalizedLead{
			FirstName:   e.FirstName,
			LastName:    e.LastName,
			Email:       strPtr(e.Email),
			LinkedinURL: strPtr(e.LinkedinURL),
			Title:       strPtr(e.JobTitle),
			Location:    strPtr(e.Location),
			Phone:       strPtr(e.Phone),
		}
		if e.Score != 0 {
			f := e.Score
			lead.Score = &f
		}
		row.Lead = lead
		rows = append(rows, row)
	}
	return rows
}

// strPtr returns nil if s is empty, otherwise a pointer to s.
func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
