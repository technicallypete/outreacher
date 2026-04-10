package importer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

const DefaultOpenAIModel = "gpt-4o-mini"

// ExtractWithOpenAI calls an OpenAI model to extract structured contact rows
// from arbitrary CSV or tabular text content. It uses function calling to get
// reliable JSON output. model defaults to DefaultOpenAIModel if empty.
func ExtractWithOpenAI(ctx context.Context, apiKey, model, content string) ([]Row, error) {
	if model == "" {
		model = DefaultOpenAIModel
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	client := openai.NewClient(option.WithAPIKey(apiKey))

	resp, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.ChatModel(model),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(fmt.Sprintf(
				"Extract all contacts from the following data. "+
					"For linkedin_url, use any LinkedIn profile URL present in the data — "+
					"including member ID format URLs like https://www.linkedin.com/in/ACoAAA... "+
					"Strip any HTML tags from field values. "+
					"Use empty string for any field that is not present.\n\n%s",
				content,
			)),
		},
		Tools: []openai.ChatCompletionToolParam{
			{
				Function: openai.FunctionDefinitionParam{
					Name:        "extract_contacts",
					Description: openai.String("Extract all contact records from the provided data."),
					Parameters:  openai.FunctionParameters(extractionToolSchema),
				},
			},
		},
		ToolChoice: openai.ChatCompletionToolChoiceOptionUnionParam{
			OfChatCompletionNamedToolChoice: &openai.ChatCompletionNamedToolChoiceParam{
				Function: openai.ChatCompletionNamedToolChoiceFunctionParam{
					Name: "extract_contacts",
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("openai api: %w", err)
	}

	if len(resp.Choices) == 0 || len(resp.Choices[0].Message.ToolCalls) == 0 {
		return nil, fmt.Errorf("openai returned no tool call")
	}

	var result extractionResult
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.ToolCalls[0].Function.Arguments), &result); err != nil {
		return nil, fmt.Errorf("unmarshal extraction result: %w", err)
	}

	return toRows(result.Rows), nil
}
