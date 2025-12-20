package gemini

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/looplj/axonhub/internal/llm"
	"github.com/looplj/axonhub/internal/llm/transformer/shared"
)

// =============================================================================
// Basic Tests for convertLLMToGeminiRequest
// =============================================================================

func TestConvertLLMToGeminiRequest_Basic(t *testing.T) {
	tests := []struct {
		name     string
		input    *llm.Request
		validate func(t *testing.T, result *GenerateContentRequest)
	}{
		{
			name: "simple text request",
			input: &llm.Request{
				Model:     "gemini-2.5-flash",
				MaxTokens: lo.ToPtr(int64(1024)),
				Messages: []llm.Message{
					{
						Role: "user",
						Content: llm.MessageContent{
							Content: lo.ToPtr("Hello, Gemini!"),
						},
					},
				},
			},
			validate: func(t *testing.T, result *GenerateContentRequest) {
				t.Helper()
				require.NotNil(t, result.GenerationConfig)
				require.Equal(t, int64(1024), result.GenerationConfig.MaxOutputTokens)
				require.Len(t, result.Contents, 1)
				require.Equal(t, "user", result.Contents[0].Role)
				require.Len(t, result.Contents[0].Parts, 1)
				require.Equal(t, "Hello, Gemini!", result.Contents[0].Parts[0].Text)
			},
		},
		{
			name: "request with system message",
			input: &llm.Request{
				Messages: []llm.Message{
					{
						Role: "system",
						Content: llm.MessageContent{
							Content: lo.ToPtr("You are a helpful assistant."),
						},
					},
					{
						Role: "user",
						Content: llm.MessageContent{
							Content: lo.ToPtr("Hello!"),
						},
					},
				},
			},
			validate: func(t *testing.T, result *GenerateContentRequest) {
				t.Helper()
				require.NotNil(t, result.SystemInstruction)
				require.Len(t, result.SystemInstruction.Parts, 1)
				require.Equal(t, "You are a helpful assistant.", result.SystemInstruction.Parts[0].Text)
				require.Len(t, result.Contents, 1)
				require.Equal(t, "user", result.Contents[0].Role)
			},
		},
		{
			name: "request with multiple system messages",
			input: &llm.Request{
				Messages: []llm.Message{
					{
						Role: "system",
						Content: llm.MessageContent{
							Content: lo.ToPtr("First instruction."),
						},
					},
					{
						Role: "system",
						Content: llm.MessageContent{
							Content: lo.ToPtr("Second instruction."),
						},
					},
					{
						Role: "user",
						Content: llm.MessageContent{
							Content: lo.ToPtr("Hello!"),
						},
					},
				},
			},
			validate: func(t *testing.T, result *GenerateContentRequest) {
				t.Helper()
				require.NotNil(t, result.SystemInstruction)
				require.Len(t, result.SystemInstruction.Parts, 2)
				require.Equal(t, "First instruction.", result.SystemInstruction.Parts[0].Text)
				require.Equal(t, "Second instruction.", result.SystemInstruction.Parts[1].Text)
			},
		},
		{
			name: "request with system message containing multiple content parts",
			input: &llm.Request{
				Messages: []llm.Message{
					{
						Role: "system",
						Content: llm.MessageContent{
							MultipleContent: []llm.MessageContentPart{
								{Type: "text", Text: lo.ToPtr("Part A")},
								{Type: "text", Text: lo.ToPtr("Part B")},
								{Type: "text", Text: lo.ToPtr("Part C")},
							},
						},
					},
					{
						Role: "user",
						Content: llm.MessageContent{
							Content: lo.ToPtr("Hello!"),
						},
					},
				},
			},
			validate: func(t *testing.T, result *GenerateContentRequest) {
				t.Helper()
				require.NotNil(t, result.SystemInstruction)
				require.Len(t, result.SystemInstruction.Parts, 3)
				require.Equal(t, "Part A", result.SystemInstruction.Parts[0].Text)
				require.Equal(t, "Part B", result.SystemInstruction.Parts[1].Text)
				require.Equal(t, "Part C", result.SystemInstruction.Parts[2].Text)
			},
		},
		{
			name: "request with generation config",
			input: &llm.Request{
				MaxTokens:        lo.ToPtr(int64(2048)),
				Temperature:      lo.ToPtr(0.7),
				TopP:             lo.ToPtr(0.9),
				PresencePenalty:  lo.ToPtr(0.5),
				FrequencyPenalty: lo.ToPtr(0.3),
				Seed:             lo.ToPtr(int64(42)),
				Stop: &llm.Stop{
					MultipleStop: []string{"END", "STOP"},
				},
				Messages: []llm.Message{
					{
						Role: "user",
						Content: llm.MessageContent{
							Content: lo.ToPtr("Test"),
						},
					},
				},
			},
			validate: func(t *testing.T, result *GenerateContentRequest) {
				t.Helper()
				require.NotNil(t, result.GenerationConfig)
				require.Equal(t, int64(2048), result.GenerationConfig.MaxOutputTokens)
				require.InDelta(t, float32(0.7), *result.GenerationConfig.Temperature, 0.01)
				require.InDelta(t, float32(0.9), *result.GenerationConfig.TopP, 0.01)
				require.InDelta(t, float32(0.5), *result.GenerationConfig.PresencePenalty, 0.01)
				require.InDelta(t, float32(0.3), *result.GenerationConfig.FrequencyPenalty, 0.01)
				require.Equal(t, int64(42), *result.GenerationConfig.Seed)
				require.Equal(t, []string{"END", "STOP"}, result.GenerationConfig.StopSequences)
			},
		},
		{
			name: "request with single stop sequence",
			input: &llm.Request{
				Stop: &llm.Stop{
					Stop: lo.ToPtr("END"),
				},
				Messages: []llm.Message{
					{
						Role: "user",
						Content: llm.MessageContent{
							Content: lo.ToPtr("Test"),
						},
					},
				},
			},
			validate: func(t *testing.T, result *GenerateContentRequest) {
				t.Helper()
				require.NotNil(t, result.GenerationConfig)
				require.Equal(t, []string{"END"}, result.GenerationConfig.StopSequences)
			},
		},
		{
			name: "request with max_completion_tokens",
			input: &llm.Request{
				MaxCompletionTokens: lo.ToPtr(int64(512)),
				Messages: []llm.Message{
					{
						Role: "user",
						Content: llm.MessageContent{
							Content: lo.ToPtr("Test"),
						},
					},
				},
			},
			validate: func(t *testing.T, result *GenerateContentRequest) {
				t.Helper()
				require.NotNil(t, result.GenerationConfig)
				require.Equal(t, int64(512), result.GenerationConfig.MaxOutputTokens)
			},
		},
		{
			name: "request with reasoning effort low",
			input: &llm.Request{
				ReasoningEffort: "low",
				Messages: []llm.Message{
					{
						Role: "user",
						Content: llm.MessageContent{
							Content: lo.ToPtr("Quick question"),
						},
					},
				},
			},
			validate: func(t *testing.T, result *GenerateContentRequest) {
				t.Helper()
				require.NotNil(t, result.GenerationConfig)
				require.NotNil(t, result.GenerationConfig.ThinkingConfig)
				require.True(t, result.GenerationConfig.ThinkingConfig.IncludeThoughts)
				require.Equal(t, "low", result.GenerationConfig.ThinkingConfig.ThinkingLevel)
				require.Nil(t, result.GenerationConfig.ThinkingConfig.ThinkingBudget)
			},
		},
		{
			name: "request with reasoning effort medium",
			input: &llm.Request{
				ReasoningEffort: "medium",
				Messages: []llm.Message{
					{
						Role: "user",
						Content: llm.MessageContent{
							Content: lo.ToPtr("Normal question"),
						},
					},
				},
			},
			validate: func(t *testing.T, result *GenerateContentRequest) {
				t.Helper()
				require.NotNil(t, result.GenerationConfig)
				require.NotNil(t, result.GenerationConfig.ThinkingConfig)
				require.True(t, result.GenerationConfig.ThinkingConfig.IncludeThoughts)
				require.Equal(t, "medium", result.GenerationConfig.ThinkingConfig.ThinkingLevel)
				require.Nil(t, result.GenerationConfig.ThinkingConfig.ThinkingBudget)
			},
		},
		{
			name: "request with reasoning effort high",
			input: &llm.Request{
				ReasoningEffort: "high",
				Messages: []llm.Message{
					{
						Role: "user",
						Content: llm.MessageContent{
							Content: lo.ToPtr("Complex problem"),
						},
					},
				},
			},
			validate: func(t *testing.T, result *GenerateContentRequest) {
				t.Helper()
				require.NotNil(t, result.GenerationConfig)
				require.NotNil(t, result.GenerationConfig.ThinkingConfig)
				require.True(t, result.GenerationConfig.ThinkingConfig.IncludeThoughts)
				require.Equal(t, "high", result.GenerationConfig.ThinkingConfig.ThinkingLevel)
				require.Nil(t, result.GenerationConfig.ThinkingConfig.ThinkingBudget)
			},
		},
		{
			name: "request with reasoning effort and budget preservation",
			input: &llm.Request{
				ReasoningEffort: "medium",
				ReasoningBudget: lo.ToPtr(int64(12000)),
				Messages: []llm.Message{
					{
						Role: "user",
						Content: llm.MessageContent{
							Content: lo.ToPtr("Question with custom budget"),
						},
					},
				},
			},
			validate: func(t *testing.T, result *GenerateContentRequest) {
				t.Helper()
				require.NotNil(t, result.GenerationConfig)
				require.NotNil(t, result.GenerationConfig.ThinkingConfig)
				require.Equal(t, int64(12000), *result.GenerationConfig.ThinkingConfig.ThinkingBudget)
			},
		},
		{
			name: "request with reasoning effort and budget exceeding max",
			input: &llm.Request{
				ReasoningEffort: "high",
				ReasoningBudget: lo.ToPtr(int64(50000)),
				Messages: []llm.Message{
					{
						Role: "user",
						Content: llm.MessageContent{
							Content: lo.ToPtr("Question with large budget"),
						},
					},
				},
			},
			validate: func(t *testing.T, result *GenerateContentRequest) {
				t.Helper()
				require.NotNil(t, result.GenerationConfig)
				require.NotNil(t, result.GenerationConfig.ThinkingConfig)
				require.True(t, result.GenerationConfig.ThinkingConfig.IncludeThoughts)
				// Should be capped at 24576 (Gemini max)
				require.Equal(t, int64(24576), *result.GenerationConfig.ThinkingConfig.ThinkingBudget)
			},
		},
		{
			name: "request with reasoning budget priority - budget exceeds max",
			input: &llm.Request{
				ReasoningBudget: lo.ToPtr(int64(50000)),
				Messages: []llm.Message{
					{
						Role: "user",
						Content: llm.MessageContent{
							Content: lo.ToPtr("Test"),
						},
					},
				},
			},
			validate: func(t *testing.T, result *GenerateContentRequest) {
				t.Helper()
				require.NotNil(t, result.GenerationConfig)
				require.NotNil(t, result.GenerationConfig.ThinkingConfig)
				require.Equal(t, int64(24576), *result.GenerationConfig.ThinkingConfig.ThinkingBudget)
			},
		},
		{
			name: "request with reasoning budget priority - budget overrides effort",
			input: &llm.Request{
				ReasoningEffort: "low",                  // Would map to 1024
				ReasoningBudget: lo.ToPtr(int64(20000)), // Budget should take priority
				Messages: []llm.Message{
					{
						Role: "user",
						Content: llm.MessageContent{
							Content: lo.ToPtr("Complex task"),
						},
					},
				},
			},
			validate: func(t *testing.T, result *GenerateContentRequest) {
				t.Helper()
				require.NotNil(t, result.GenerationConfig)
				require.NotNil(t, result.GenerationConfig.ThinkingConfig)
				require.Equal(t, int64(20000), *result.GenerationConfig.ThinkingConfig.ThinkingBudget)
				require.True(t, result.GenerationConfig.ThinkingConfig.IncludeThoughts)
			},
		},
		{
			name: "request with reasoning budget only - no effort",
			input: &llm.Request{
				ReasoningBudget: lo.ToPtr(int64(15000)),
				Messages: []llm.Message{
					{
						Role: "user",
						Content: llm.MessageContent{
							Content: lo.ToPtr("Task"),
						},
					},
				},
			},
			validate: func(t *testing.T, result *GenerateContentRequest) {
				t.Helper()
				require.NotNil(t, result.GenerationConfig)
				require.NotNil(t, result.GenerationConfig.ThinkingConfig)
				require.Equal(t, int64(15000), *result.GenerationConfig.ThinkingConfig.ThinkingBudget)
				require.True(t, result.GenerationConfig.ThinkingConfig.IncludeThoughts)
			},
		},
		{
			name: "request with reasoning effort only - no budget",
			input: &llm.Request{
				ReasoningEffort: "high",
				Messages: []llm.Message{
					{
						Role: "user",
						Content: llm.MessageContent{
							Content: lo.ToPtr("Task"),
						},
					},
				},
			},
			validate: func(t *testing.T, result *GenerateContentRequest) {
				t.Helper()
				require.NotNil(t, result.GenerationConfig)
				require.NotNil(t, result.GenerationConfig.ThinkingConfig)
				require.Equal(t, "high", result.GenerationConfig.ThinkingConfig.ThinkingLevel) // Should use ThinkingLevel for standard values
				require.Nil(t, result.GenerationConfig.ThinkingConfig.ThinkingBudget)
				require.True(t, result.GenerationConfig.ThinkingConfig.IncludeThoughts)
			},
		},
		{
			name: "request with ExtraBody ThinkingLevel priority",
			input: &llm.Request{
				Messages: []llm.Message{
					{
						Role: "user",
						Content: llm.MessageContent{
							Content: lo.ToPtr("Test"),
						},
					},
				},
				ExtraBody: json.RawMessage(`{"google":{"thinking_config":{"thinking_level":"high","thinking_budget":1024,"include_thoughts":true}}}`),
			},
			validate: func(t *testing.T, result *GenerateContentRequest) {
				t.Helper()
				require.NotNil(t, result.GenerationConfig)
				require.NotNil(t, result.GenerationConfig.ThinkingConfig)
				require.Equal(t, "high", result.GenerationConfig.ThinkingConfig.ThinkingLevel) // Level takes priority
				require.Nil(t, result.GenerationConfig.ThinkingConfig.ThinkingBudget)          // Budget should not be set when level is present
				require.True(t, result.GenerationConfig.ThinkingConfig.IncludeThoughts)
			},
		},
		{
			name: "request with ExtraBody minimal mapping to low",
			input: &llm.Request{
				Messages: []llm.Message{
					{
						Role: "user",
						Content: llm.MessageContent{
							Content: lo.ToPtr("Test"),
						},
					},
				},
				ExtraBody: json.RawMessage(`{"google":{"thinking_config":{"thinking_level":"minimal","include_thoughts":true}}}`),
			},
			validate: func(t *testing.T, result *GenerateContentRequest) {
				t.Helper()
				require.NotNil(t, result.GenerationConfig)
				require.NotNil(t, result.GenerationConfig.ThinkingConfig)
				require.Equal(t, "low", result.GenerationConfig.ThinkingConfig.ThinkingLevel) // minimal maps to low
				require.Nil(t, result.GenerationConfig.ThinkingConfig.ThinkingBudget)
				require.True(t, result.GenerationConfig.ThinkingConfig.IncludeThoughts)
			},
		},
		{
			name: "request with ExtraBody string budget converts to level",
			input: &llm.Request{
				Messages: []llm.Message{
					{
						Role: "user",
						Content: llm.MessageContent{
							Content: lo.ToPtr("Test"),
						},
					},
				},
				ExtraBody: json.RawMessage(`{"google":{"thinking_config":{"thinking_budget":"high","include_thoughts":true}}}`),
			},
			validate: func(t *testing.T, result *GenerateContentRequest) {
				t.Helper()
				require.NotNil(t, result.GenerationConfig)
				require.NotNil(t, result.GenerationConfig.ThinkingConfig)
				require.Equal(t, "high", result.GenerationConfig.ThinkingConfig.ThinkingLevel) // String budget converts to level
				require.Nil(t, result.GenerationConfig.ThinkingConfig.ThinkingBudget)
				require.True(t, result.GenerationConfig.ThinkingConfig.IncludeThoughts)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertLLMToGeminiRequest(tt.input)
			tt.validate(t, result)
		})
	}
}

func TestConvertLLMToGeminiRequest_Tools(t *testing.T) {
	tests := []struct {
		name     string
		input    *llm.Request
		validate func(t *testing.T, result *GenerateContentRequest)
	}{
		{
			name: "request with tools",
			input: &llm.Request{
				Messages: []llm.Message{
					{
						Role: "user",
						Content: llm.MessageContent{
							Content: lo.ToPtr("What's the weather?"),
						},
					},
				},
				Tools: []llm.Tool{
					{
						Type: "function",
						Function: llm.Function{
							Name:        "get_weather",
							Description: "Get weather information",
							Parameters:  json.RawMessage(`{"type":"object","properties":{"location":{"type":"string"}}}`),
						},
					},
				},
			},
			validate: func(t *testing.T, result *GenerateContentRequest) {
				t.Helper()
				require.Len(t, result.Tools, 1)
				require.Len(t, result.Tools[0].FunctionDeclarations, 1)
				require.Equal(t, "get_weather", result.Tools[0].FunctionDeclarations[0].Name)
				require.Equal(t, "Get weather information", result.Tools[0].FunctionDeclarations[0].Description)
			},
		},
		{
			name: "request with multiple tools",
			input: &llm.Request{
				Messages: []llm.Message{
					{
						Role: "user",
						Content: llm.MessageContent{
							Content: lo.ToPtr("Help me"),
						},
					},
				},
				Tools: []llm.Tool{
					{
						Type: "function",
						Function: llm.Function{
							Name:        "tool1",
							Description: "First tool",
						},
					},
					{
						Type: "function",
						Function: llm.Function{
							Name:        "tool2",
							Description: "Second tool",
						},
					},
				},
			},
			validate: func(t *testing.T, result *GenerateContentRequest) {
				t.Helper()
				require.Len(t, result.Tools, 1)
				require.Len(t, result.Tools[0].FunctionDeclarations, 2)
				require.Equal(t, "tool1", result.Tools[0].FunctionDeclarations[0].Name)
				require.Equal(t, "tool2", result.Tools[0].FunctionDeclarations[1].Name)
			},
		},
		{
			name: "request with google search tool",
			input: &llm.Request{
				Messages: []llm.Message{
					{
						Role: "user",
						Content: llm.MessageContent{
							Content: lo.ToPtr("Search the web"),
						},
					},
				},
				Tools: []llm.Tool{
					{
						Type: llm.ToolTypeGoogleSearch,
						Google: &llm.GoogleTools{
							Search: &llm.GoogleSearch{},
						},
					},
				},
			},
			validate: func(t *testing.T, result *GenerateContentRequest) {
				t.Helper()
				require.Len(t, result.Tools, 1)
				require.NotNil(t, result.Tools[0].GoogleSearch)
				require.Nil(t, result.Tools[0].FunctionDeclarations)
				require.Nil(t, result.Tools[0].CodeExecution)
			},
		},
		{
			name: "request with code execution tool",
			input: &llm.Request{
				Messages: []llm.Message{
					{
						Role: "user",
						Content: llm.MessageContent{
							Content: lo.ToPtr("Run some code"),
						},
					},
				},
				Tools: []llm.Tool{
					{
						Type: llm.ToolTypeGoogleCodeExecution,
						Google: &llm.GoogleTools{
							CodeExecution: &llm.GoogleCodeExecution{},
						},
					},
				},
			},
			validate: func(t *testing.T, result *GenerateContentRequest) {
				t.Helper()
				require.Len(t, result.Tools, 1)
				require.NotNil(t, result.Tools[0].CodeExecution)
				require.Nil(t, result.Tools[0].FunctionDeclarations)
				require.Nil(t, result.Tools[0].GoogleSearch)
			},
		},
		{
			name: "request with url context tool",
			input: &llm.Request{
				Messages: []llm.Message{
					{
						Role: "user",
						Content: llm.MessageContent{
							Content: lo.ToPtr("Fetch URL content"),
						},
					},
				},
				Tools: []llm.Tool{
					{
						Type: llm.ToolTypeGoogleUrlContext,
						Google: &llm.GoogleTools{
							UrlContext: &llm.GoogleUrlContext{},
						},
					},
				},
			},
			validate: func(t *testing.T, result *GenerateContentRequest) {
				t.Helper()
				require.Len(t, result.Tools, 1)
				require.NotNil(t, result.Tools[0].UrlContext)
				require.Nil(t, result.Tools[0].FunctionDeclarations)
				require.Nil(t, result.Tools[0].GoogleSearch)
				require.Nil(t, result.Tools[0].CodeExecution)
			},
		},
		{
			name: "request with mixed tools (function, google search, code execution)",
			input: &llm.Request{
				Messages: []llm.Message{
					{
						Role: "user",
						Content: llm.MessageContent{
							Content: lo.ToPtr("Use all tools"),
						},
					},
				},
				Tools: []llm.Tool{
					{
						Type: "function",
						Function: llm.Function{
							Name:        "get_weather",
							Description: "Get weather info",
							Parameters:  json.RawMessage(`{"type":"object"}`),
						},
					},
					{
						Type: llm.ToolTypeGoogleSearch,
						Google: &llm.GoogleTools{
							Search: &llm.GoogleSearch{},
						},
					},
					{
						Type: llm.ToolTypeGoogleCodeExecution,
						Google: &llm.GoogleTools{
							CodeExecution: &llm.GoogleCodeExecution{},
						},
					},
				},
			},
			validate: func(t *testing.T, result *GenerateContentRequest) {
				t.Helper()
				require.Len(t, result.Tools, 3)
				// First tool should have function declarations
				require.NotNil(t, result.Tools[0].FunctionDeclarations)
				require.Len(t, result.Tools[0].FunctionDeclarations, 1)
				require.Equal(t, "get_weather", result.Tools[0].FunctionDeclarations[0].Name)
				// Second tool should be google search
				require.NotNil(t, result.Tools[1].GoogleSearch)
				// Third tool should be code execution
				require.NotNil(t, result.Tools[2].CodeExecution)
			},
		},
		{
			name: "request with tool choice auto",
			input: &llm.Request{
				Messages: []llm.Message{
					{
						Role: "user",
						Content: llm.MessageContent{
							Content: lo.ToPtr("Test"),
						},
					},
				},
				ToolChoice: &llm.ToolChoice{
					ToolChoice: lo.ToPtr("auto"),
				},
			},
			validate: func(t *testing.T, result *GenerateContentRequest) {
				t.Helper()
				require.NotNil(t, result.ToolConfig)
				require.NotNil(t, result.ToolConfig.FunctionCallingConfig)
				require.Equal(t, "AUTO", result.ToolConfig.FunctionCallingConfig.Mode)
			},
		},
		{
			name: "request with tool choice none",
			input: &llm.Request{
				Messages: []llm.Message{
					{
						Role: "user",
						Content: llm.MessageContent{
							Content: lo.ToPtr("Test"),
						},
					},
				},
				ToolChoice: &llm.ToolChoice{
					ToolChoice: lo.ToPtr("none"),
				},
			},
			validate: func(t *testing.T, result *GenerateContentRequest) {
				t.Helper()
				require.NotNil(t, result.ToolConfig)
				require.NotNil(t, result.ToolConfig.FunctionCallingConfig)
				require.Equal(t, "NONE", result.ToolConfig.FunctionCallingConfig.Mode)
			},
		},
		{
			name: "request with tool choice required",
			input: &llm.Request{
				Messages: []llm.Message{
					{
						Role: "user",
						Content: llm.MessageContent{
							Content: lo.ToPtr("Test"),
						},
					},
				},
				ToolChoice: &llm.ToolChoice{
					ToolChoice: lo.ToPtr("required"),
				},
			},
			validate: func(t *testing.T, result *GenerateContentRequest) {
				t.Helper()
				require.NotNil(t, result.ToolConfig)
				require.NotNil(t, result.ToolConfig.FunctionCallingConfig)
				require.Equal(t, "ANY", result.ToolConfig.FunctionCallingConfig.Mode)
			},
		},
		{
			name: "request with named tool choice",
			input: &llm.Request{
				Messages: []llm.Message{
					{
						Role: "user",
						Content: llm.MessageContent{
							Content: lo.ToPtr("Test"),
						},
					},
				},
				ToolChoice: &llm.ToolChoice{
					NamedToolChoice: &llm.NamedToolChoice{
						Type: "function",
						Function: llm.ToolFunction{
							Name: "specific_function",
						},
					},
				},
			},
			validate: func(t *testing.T, result *GenerateContentRequest) {
				t.Helper()
				require.NotNil(t, result.ToolConfig)
				require.NotNil(t, result.ToolConfig.FunctionCallingConfig)
				require.Equal(t, "ANY", result.ToolConfig.FunctionCallingConfig.Mode)
				require.Equal(t, []string{"specific_function"}, result.ToolConfig.FunctionCallingConfig.AllowedFunctionNames)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertLLMToGeminiRequest(tt.input)
			tt.validate(t, result)
		})
	}
}

func TestConvertLLMMessageToGeminiContent(t *testing.T) {
	tests := []struct {
		name     string
		input    *llm.Message
		validate func(t *testing.T, result *Content)
	}{
		{
			name: "simple text message",
			input: &llm.Message{
				Role: "user",
				Content: llm.MessageContent{
					Content: lo.ToPtr("Hello"),
				},
			},
			validate: func(t *testing.T, result *Content) {
				t.Helper()
				require.NotNil(t, result)
				require.Equal(t, "user", result.Role)
				require.Len(t, result.Parts, 1)
				require.Equal(t, "Hello", result.Parts[0].Text)
			},
		},
		{
			name: "assistant role conversion",
			input: &llm.Message{
				Role: "assistant",
				Content: llm.MessageContent{
					Content: lo.ToPtr("Response"),
				},
			},
			validate: func(t *testing.T, result *Content) {
				t.Helper()
				require.Equal(t, "model", result.Role)
			},
		},
		{
			name: "message with reasoning content",
			input: &llm.Message{
				Role:             "assistant",
				ReasoningContent: lo.ToPtr("Let me think..."),
				Content: llm.MessageContent{
					Content: lo.ToPtr("The answer is 42"),
				},
			},
			validate: func(t *testing.T, result *Content) {
				t.Helper()
				require.Len(t, result.Parts, 2)
				require.True(t, result.Parts[0].Thought)
				require.Equal(t, "Let me think...", result.Parts[0].Text)
				require.False(t, result.Parts[1].Thought)
				require.Equal(t, "The answer is 42", result.Parts[1].Text)
			},
		},
		{
			name: "message with multiple content parts",
			input: &llm.Message{
				Role: "user",
				Content: llm.MessageContent{
					MultipleContent: []llm.MessageContentPart{
						{Type: "text", Text: lo.ToPtr("First part")},
						{Type: "text", Text: lo.ToPtr("Second part")},
					},
				},
			},
			validate: func(t *testing.T, result *Content) {
				t.Helper()
				require.Len(t, result.Parts, 2)
				require.Equal(t, "First part", result.Parts[0].Text)
				require.Equal(t, "Second part", result.Parts[1].Text)
			},
		},
		{
			name: "message with image URL (data URL)",
			input: &llm.Message{
				Role: "user",
				Content: llm.MessageContent{
					MultipleContent: []llm.MessageContentPart{
						{
							Type: "image_url",
							ImageURL: &llm.ImageURL{
								URL: "data:image/jpeg;base64,/9j/4AAQSkZJRg==",
							},
						},
					},
				},
			},
			validate: func(t *testing.T, result *Content) {
				t.Helper()
				require.Len(t, result.Parts, 1)
				require.NotNil(t, result.Parts[0].InlineData)
				require.Equal(t, "image/jpeg", result.Parts[0].InlineData.MIMEType)
				require.Equal(t, "/9j/4AAQSkZJRg==", result.Parts[0].InlineData.Data)
			},
		},
		{
			name: "message with image URL (regular URL)",
			input: &llm.Message{
				Role: "user",
				Content: llm.MessageContent{
					MultipleContent: []llm.MessageContentPart{
						{
							Type: "image_url",
							ImageURL: &llm.ImageURL{
								URL: "https://example.com/image.jpg",
							},
						},
					},
				},
			},
			validate: func(t *testing.T, result *Content) {
				t.Helper()
				require.Len(t, result.Parts, 1)
				require.NotNil(t, result.Parts[0].FileData)
				require.Equal(t, "https://example.com/image.jpg", result.Parts[0].FileData.FileURI)
			},
		},
		{
			name: "message with tool calls",
			input: &llm.Message{
				Role: "assistant",
				Content: llm.MessageContent{
					Content: lo.ToPtr("I'll check the weather"),
				},
				ToolCalls: []llm.ToolCall{
					{
						ID:   "call_001",
						Type: "function",
						Function: llm.FunctionCall{
							Name:      "get_weather",
							Arguments: `{"location":"NYC"}`,
						},
					},
				},
			},
			validate: func(t *testing.T, result *Content) {
				t.Helper()
				require.Len(t, result.Parts, 2)
				require.Equal(t, "I'll check the weather", result.Parts[0].Text)
				require.NotNil(t, result.Parts[1].FunctionCall)
				require.Equal(t, "call_001", result.Parts[1].FunctionCall.ID)
				require.Equal(t, "get_weather", result.Parts[1].FunctionCall.Name)
				require.Equal(t, "NYC", result.Parts[1].FunctionCall.Args["location"])
			},
		},
		{
			name: "empty message",
			input: &llm.Message{
				Role:    "user",
				Content: llm.MessageContent{},
			},
			validate: func(t *testing.T, result *Content) {
				t.Helper()
				require.Nil(t, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertLLMMessageToGeminiContent(tt.input)
			tt.validate(t, result)
		})
	}
}

func TestConvertLLMToolMessageToGeminiContent(t *testing.T) {
	tests := []struct {
		name     string
		input    *llm.Message
		req      *GenerateContentRequest
		validate func(t *testing.T, result *Content)
	}{
		{
			name: "tool message with JSON content",
			input: &llm.Message{
				Role:       "tool",
				ToolCallID: lo.ToPtr("call_123"),
				Content: llm.MessageContent{
					Content: lo.ToPtr(`{"temperature": 72, "unit": "F"}`),
				},
			},
			req: &GenerateContentRequest{},
			validate: func(t *testing.T, result *Content) {
				t.Helper()
				require.Equal(t, "user", result.Role)
				require.Len(t, result.Parts, 1)
				require.NotNil(t, result.Parts[0].FunctionResponse)
				require.Equal(t, "call_123", result.Parts[0].FunctionResponse.ID)
				require.Equal(t, 72.0, result.Parts[0].FunctionResponse.Response["temperature"])
			},
		},
		{
			name: "tool message with non-JSON content",
			input: &llm.Message{
				Role:       "tool",
				ToolCallID: lo.ToPtr("call_456"),
				Content: llm.MessageContent{
					Content: lo.ToPtr("Plain text result"),
				},
			},
			req: &GenerateContentRequest{},
			validate: func(t *testing.T, result *Content) {
				t.Helper()
				require.Equal(t, "user", result.Role)
				require.Len(t, result.Parts, 1)
				require.NotNil(t, result.Parts[0].FunctionResponse)
				require.Equal(t, "call_456", result.Parts[0].FunctionResponse.ID)
				require.Equal(t, "Plain text result", result.Parts[0].FunctionResponse.Response["result"])
			},
		},
		{
			name: "tool message without tool call ID",
			input: &llm.Message{
				Role: "tool",
				Content: llm.MessageContent{
					Content: lo.ToPtr(`{"result": "success"}`),
				},
			},
			req: &GenerateContentRequest{},
			validate: func(t *testing.T, result *Content) {
				t.Helper()
				require.Equal(t, "user", result.Role)
				require.Len(t, result.Parts, 1)
				require.NotNil(t, result.Parts[0].FunctionResponse)
				require.Equal(t, "", result.Parts[0].FunctionResponse.ID)
			},
		},
		{
			name: "tool message with name from ToolCallName",
			input: &llm.Message{
				Role:         "tool",
				ToolCallID:   lo.ToPtr("call_789"),
				ToolCallName: lo.ToPtr("get_weather"),
				Content: llm.MessageContent{
					Content: lo.ToPtr(`{"temperature": 25, "unit": "C"}`),
				},
			},
			req: &GenerateContentRequest{},
			validate: func(t *testing.T, result *Content) {
				t.Helper()
				require.Equal(t, "user", result.Role)
				require.Len(t, result.Parts, 1)
				require.NotNil(t, result.Parts[0].FunctionResponse)
				require.Equal(t, "call_789", result.Parts[0].FunctionResponse.ID)
				require.Equal(t, "get_weather", result.Parts[0].FunctionResponse.Name)
			},
		},
		{
			name: "tool message find name from previous function call",
			input: &llm.Message{
				Role:       "tool",
				ToolCallID: lo.ToPtr("call_abc"),
				Content: llm.MessageContent{
					Content: lo.ToPtr(`{"result": "success"}`),
				},
			},
			req: &GenerateContentRequest{
				Contents: []*Content{
					{
						Role: "assistant",
						Parts: []*Part{
							{
								FunctionCall: &FunctionCall{
									ID:   "call_abc",
									Name: "search_web",
									Args: map[string]any{"query": "test"},
								},
							},
						},
					},
				},
			},
			validate: func(t *testing.T, result *Content) {
				t.Helper()
				require.Equal(t, "user", result.Role)
				require.Len(t, result.Parts, 1)
				require.NotNil(t, result.Parts[0].FunctionResponse)
				require.Equal(t, "call_abc", result.Parts[0].FunctionResponse.ID)
				require.Equal(t, "search_web", result.Parts[0].FunctionResponse.Name)
			},
		},
		{
			name: "tool message with empty content",
			input: &llm.Message{
				Role:       "tool",
				ToolCallID: lo.ToPtr("call_empty"),
				Content: llm.MessageContent{
					Content: lo.ToPtr(""),
				},
			},
			req: &GenerateContentRequest{},
			validate: func(t *testing.T, result *Content) {
				t.Helper()
				require.Equal(t, "user", result.Role)
				require.Len(t, result.Parts, 1)
				require.NotNil(t, result.Parts[0].FunctionResponse)
				require.Equal(t, "call_empty", result.Parts[0].FunctionResponse.ID)
				require.Equal(t, "", result.Parts[0].FunctionResponse.Response["result"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertLLMToolResultToGeminiContent(tt.input, tt.req.Contents)
			tt.validate(t, result)
		})
	}
}

func TestConvertLLMToGeminiRequest_Modalities(t *testing.T) {
	tests := []struct {
		name     string
		input    *llm.Request
		validate func(t *testing.T, result *GenerateContentRequest)
	}{
		{
			name: "request with text modality",
			input: &llm.Request{
				Messages: []llm.Message{
					{
						Role: "user",
						Content: llm.MessageContent{
							Content: lo.ToPtr("Text only"),
						},
					},
				},
				Modalities: []string{"text"},
			},
			validate: func(t *testing.T, result *GenerateContentRequest) {
				t.Helper()
				require.NotNil(t, result.GenerationConfig)
				require.Equal(t, []string{"text"}, result.GenerationConfig.ResponseModalities)
			},
		},
		{
			name: "request with text and image modalities",
			input: &llm.Request{
				Messages: []llm.Message{
					{
						Role: "user",
						Content: llm.MessageContent{
							Content: lo.ToPtr("Describe this image"),
						},
					},
				},
				Modalities: []string{"text", "image"},
			},
			validate: func(t *testing.T, result *GenerateContentRequest) {
				t.Helper()
				require.NotNil(t, result.GenerationConfig)
				require.Equal(t, []string{"text", "image"}, result.GenerationConfig.ResponseModalities)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertLLMToGeminiRequest(tt.input)
			tt.validate(t, result)
		})
	}
}

// =============================================================================
// Basic Tests for convertGeminiToLLMResponse
// =============================================================================

func TestConvertGeminiToLLMResponse_Basic(t *testing.T) {
	tests := []struct {
		name     string
		input    *GenerateContentResponse
		validate func(t *testing.T, result *llm.Response)
	}{
		{
			name: "simple response",
			input: &GenerateContentResponse{
				ResponseID:    "resp_123",
				ModelVersion:  "gemini-2.5-flash",
				Candidates: []*Candidate{
					{
						Index: 0,
						Content: &Content{
							Role: "model",
							Parts: []*Part{
								{Text: "Hello!"},
							},
						},
						FinishReason: "STOP",
					},
				},
			},
			validate: func(t *testing.T, result *llm.Response) {
				t.Helper()
				require.Equal(t, "resp_123", result.ID)
				require.Equal(t, "gemini-2.5-flash", result.Model)
				require.Equal(t, "chat.completion", result.Object)
				require.Len(t, result.Choices, 1)
				require.Equal(t, "assistant", result.Choices[0].Message.Role)
				require.Equal(t, "Hello!", *result.Choices[0].Message.Content.Content)
				require.Equal(t, "stop", *result.Choices[0].FinishReason)
			},
		},
		{
			name: "response with no ID",
			input: &GenerateContentResponse{
				ModelVersion: "gemini-2.5-flash",
				Candidates: []*Candidate{
					{
						Index: 0,
						Content: &Content{
							Role: "model",
							Parts: []*Part{
								{Text: "Response"},
							},
						},
					},
				},
			},
			validate: func(t *testing.T, result *llm.Response) {
				t.Helper()
				require.NotEmpty(t, result.ID)
				require.Contains(t, result.ID, "chatcmpl-")
			},
		},
		{
			name: "response with thinking",
			input: &GenerateContentResponse{
				ResponseID:   "resp_think",
				ModelVersion: "gemini-2.5-flash",
				Candidates: []*Candidate{
					{
						Index: 0,
						Content: &Content{
							Role: "model",
							Parts: []*Part{
								{Text: "Let me think...", Thought: true},
								{Text: "The answer is 42"},
							},
						},
					},
				},
			},
			validate: func(t *testing.T, result *llm.Response) {
				t.Helper()
				require.Len(t, result.Choices, 1)
				require.Equal(t, "assistant", result.Choices[0].Message.Role)
				require.NotNil(t, result.Choices[0].Message.ReasoningContent)
				require.Equal(t, "Let me think...", *result.Choices[0].Message.ReasoningContent)
				require.Equal(t, "The answer is 42", *result.Choices[0].Message.Content.Content)
			},
		},
		{
			name: "response with tool calls",
			input: &GenerateContentResponse{
				ResponseID:   "resp_tool",
				ModelVersion: "gemini-2.5-flash",
				Candidates: []*Candidate{
					{
						Index: 0,
						Content: &Content{
							Role: "model",
							Parts: []*Part{
								{Text: "I'll check the weather"},
								{
									FunctionCall: &FunctionCall{
										ID:   "call_001",
										Name: "get_weather",
										Args: map[string]any{"location": "NYC"},
									},
								},
							},
						},
					},
				},
			},
			validate: func(t *testing.T, result *llm.Response) {
				t.Helper()
				require.Len(t, result.Choices, 1)
				require.Len(t, result.Choices[0].Message.ToolCalls, 1)
				require.Equal(t, "call_001", result.Choices[0].Message.ToolCalls[0].ID)
				require.Equal(t, "get_weather", result.Choices[0].Message.ToolCalls[0].Function.Name)
				require.Contains(t, result.Choices[0].Message.ToolCalls[0].Function.Arguments, "location")
				require.Contains(t, result.Choices[0].Message.ToolCalls[0].Function.Arguments, "NYC")
			},
		},
		{
			name: "response with usage metadata",
			input: &GenerateContentResponse{
				ResponseID:   "resp_usage",
				ModelVersion: "gemini-2.5-flash",
				Candidates: []*Candidate{
					{
						Index: 0,
						Content: &Content{
							Role: "model",
							Parts: []*Part{
								{Text: "Response"},
							},
						},
					},
				},
				UsageMetadata: &UsageMetadata{
					PromptTokenCount:        100,
					CandidatesTokenCount:    50,
					TotalTokenCount:         150,
					CachedContentTokenCount: int64(20),
					ThoughtsTokenCount:      int64(30),
				},
			},
			validate: func(t *testing.T, result *llm.Response) {
				t.Helper()
				require.NotNil(t, result.Usage)
				require.Equal(t, 100, result.Usage.PromptTokens)
				require.Equal(t, 50, result.Usage.CompletionTokens)
				require.Equal(t, 150, result.Usage.TotalTokens)
				require.Equal(t, 20, result.Usage.PromptTokensDetails.CachedTokens)
				require.Equal(t, 30, result.Usage.CompletionTokensDetails.ReasoningTokens)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertGeminiToLLMResponse(tt.input, false)
			tt.validate(t, result)
		})
	}
}

func TestConvertGeminiToLLMResponse_Streaming(t *testing.T) {
	tests := []struct {
		name     string
		input    *GenerateContentResponse
		validate func(t *testing.T, result *llm.Response)
	}{
		{
			name: "streaming response with delta",
			input: &GenerateContentResponse{
				ResponseID:   "resp_stream",
				ModelVersion: "gemini-2.5-flash",
				Candidates: []*Candidate{
					{
						Index: 0,
						Content: &Content{
							Role: "model",
							Parts: []*Part{
								{Text: "Streaming content"},
							},
						},
					},
				},
			},
			validate: func(t *testing.T, result *llm.Response) {
				t.Helper()
				require.Equal(t, "chat.completion.chunk", result.Object)
				require.Nil(t, result.Choices[0].Message)
				require.NotNil(t, result.Choices[0].Delta)
				require.Equal(t, "assistant", result.Choices[0].Delta.Role)
				require.Equal(t, "Streaming content", *result.Choices[0].Delta.Content.Content)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertGeminiToLLMResponse(tt.input, true)
			tt.validate(t, result)
		})
	}
}

func TestConvertGeminiToLLMResponse_FinishReasons(t *testing.T) {
	finishReasons := map[string]string{
		"STOP":     "stop",
		"MAX_TOKENS": "length",
		"SAFETY":   "content_filter",
		"OTHER":    "stop", // Unknown reasons default to "stop"
	}

	for geminiReason, expectedLLMReason := range finishReasons {
		t.Run("finish_reason_"+geminiReason, func(t *testing.T) {
			input := &GenerateContentResponse{
				ResponseID:   "resp_finish",
				ModelVersion: "gemini-2.5-flash",
				Candidates: []*Candidate{
					{
						Index: 0,
						Content: &Content{
							Role: "model",
							Parts: []*Part{
								{Text: "Test"},
							},
						},
						FinishReason: geminiReason,
					},
				},
			}

			result := convertGeminiToLLMResponse(input, false)
			require.Equal(t, expectedLLMReason, *result.Choices[0].FinishReason)
		})
	}
}

func TestConvertGeminiToLLMResponse_Images(t *testing.T) {
	tests := []struct {
		name     string
		input    *GenerateContentResponse
		validate func(t *testing.T, result *llm.Response)
	}{
		{
			name: "response with inline data (image)",
			input: &GenerateContentResponse{
				ResponseID:   "resp_image",
				ModelVersion: "gemini-2.5-flash",
				Candidates: []*Candidate{
					{
						Index: 0,
						Content: &Content{
							Role: "model",
							Parts: []*Part{
								{
									InlineData: &Blob{
										MIMEType: "image/jpeg",
										Data:     "base64imagedata",
									},
								},
							},
						},
					},
				},
			},
			validate: func(t *testing.T, result *llm.Response) {
				t.Helper()
				require.Len(t, result.Choices, 1)
				require.Len(t, result.Choices[0].Message.Content.MultipleContent, 1)
				require.Equal(t, "image_url", result.Choices[0].Message.Content.MultipleContent[0].Type)
				require.Equal(t, "data:image/jpeg;base64,base64imagedata", result.Choices[0].Message.Content.MultipleContent[0].ImageURL.URL)
			},
		},
		{
			name: "response with file data (image)",
			input: &GenerateContentResponse{
				ResponseID:   "resp_file",
				ModelVersion: "gemini-2.5-flash",
				Candidates: []*Candidate{
					{
						Index: 0,
						Content: &Content{
							Role: "model",
							Parts: []*Part{
								{
									FileData: &FileData{
										MIMEType: "image/png",
										FileURI:  "gs://bucket/image.png",
									},
								},
							},
						},
					},
				},
			},
			validate: func(t *testing.T, result *llm.Response) {
				t.Helper()
				require.Len(t, result.Choices, 1)
				require.Len(t, result.Choices[0].Message.Content.MultipleContent, 1)
				require.Equal(t, "image_url", result.Choices[0].Message.Content.MultipleContent[0].Type)
				require.Equal(t, "gs://bucket/image.png", result.Choices[0].Message.Content.MultipleContent[0].ImageURL.URL)
			},
		},
		{
			name: "response with text and image",
			input: &GenerateContentResponse{
				ResponseID:   "resp_mixed",
				ModelVersion: "gemini-2.5-flash",
				Candidates: []*Candidate{
					{
						Index: 0,
						Content: &Content{
							Role: "model",
							Parts: []*Part{
								{Text: "Here's an image:"},
								{
									InlineData: &Blob{
										MIMEType: "image/jpeg",
										Data:     "base64data",
									},
								},
							},
						},
					},
				},
			},
			validate: func(t *testing.T, result *llm.Response) {
				t.Helper()
				require.Len(t, result.Choices, 1)
				require.Len(t, result.Choices[0].Message.Content.MultipleContent, 2)
				require.Equal(t, "text", result.Choices[0].Message.Content.MultipleContent[0].Type)
				require.Equal(t, "image_url", result.Choices[0].Message.Content.MultipleContent[1].Type)
				require.Equal(t, "Here's an image:", *result.Choices[0].Message.Content.MultipleContent[0].Text)
				require.Equal(t, "data:image/jpeg;base64,base64data", result.Choices[0].Message.Content.MultipleContent[1].ImageURL.URL)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertGeminiToLLMResponse(tt.input, false)
			tt.validate(t, result)
		})
	}
}

func TestConvertGeminiToLLMResponse_MultipleCandidates(t *testing.T) {
	input := &GenerateContentResponse{
		ResponseID:   "resp_multi",
		ModelVersion: "gemini-2.5-flash",
		Candidates: []*Candidate{
			{
				Index: 0,
				Content: &Content{
					Role: "model",
					Parts: []*Part{
						{Text: "First response"},
					},
				},
			},
			{
				Index: 1,
				Content: &Content{
					Role: "model",
					Parts: []*Part{
						{Text: "Second response"},
					},
				},
			},
			{
				Index: 2,
				Content: &Content{
					Role: "model",
					Parts: []*Part{
						{Text: "Third response"},
					},
				},
			},
		},
	}

	result := convertGeminiToLLMResponse(input, false)

	require.Len(t, result.Choices, 3)
	require.Equal(t, 0, result.Choices[0].Index)
	require.Equal(t, "First response", *result.Choices[0].Message.Content.Content)
	require.Equal(t, 1, result.Choices[1].Index)
	require.Equal(t, "Second response", *result.Choices[1].Message.Content.Content)
	require.Equal(t, 2, result.Choices[2].Index)
	require.Equal(t, "Third response", *result.Choices[2].Message.Content.Content)
}

func TestConvertGeminiToLLMResponse_GroundingMetadata(t *testing.T) {
	input := &GenerateContentResponse{
		ResponseID:   "resp_ground",
		ModelVersion: "gemini-2.5-flash",
		Candidates: []*Candidate{
			{
				Index: 0,
				Content: &Content{
					Role: "model",
					Parts: []*Part{
						{Text: "Grounded response"},
					},
				},
				GroundingMetadata: &GroundingMetadata{
					GroundingChunks: []*GroundingChunk{
						{
							Web: &GroundingChunkWeb{
								URI:   "https://example.com",
								Title: "Example Source",
							},
						},
					},
				},
			},
		},
	}

	result := convertGeminiToLLMResponse(input, false)

	require.Len(t, result.Choices, 1)
	require.NotNil(t, result.Choices[0].TransformerMetadata)
	groundingMetadata, ok := result.Choices[0].TransformerMetadata[TransformerMetadataKeyGroundingMetadata]
	require.True(t, ok)
	require.NotNil(t, groundingMetadata)
	sources := groundingMetadata.(map[string]any)["sources"].([]map[string]any)
	require.Len(t, sources, 1)
	require.Equal(t, "Example Source", sources[0]["title"])
}

// =============================================================================
// Testdata Tests
// =============================================================================

func TestConvertLLMToGeminiRequest_Testdata(t *testing.T) {
	testCases := []struct {
		name         string
		llmFile      string
		geminiFile   string
		validateFunc func(t *testing.T, llmReq *llm.Request, geminiReq *GenerateContentRequest)
	}{
		{
			name:       "simple request",
			llmFile:    "llm-simple.request.json",
			geminiFile: "gemini-simple.request.json",
			validateFunc: func(t *testing.T, llmReq *llm.Request, geminiReq *GenerateContentRequest) {
				t.Helper()
				require.Equal(t, llmReq.Model, "gemini-2.5-flash")
				require.Len(t, geminiReq.Contents, 1)
				require.Equal(t, "user", geminiReq.Contents[0].Role)
				require.Len(t, geminiReq.Contents[0].Parts, 1)
				require.Equal(t, "Output 1-20, 5 each line", geminiReq.Contents[0].Parts[0].Text)
				require.Equal(t, int64(4096), geminiReq.GenerationConfig.MaxOutputTokens)
				require.Equal(t, "low", geminiReq.GenerationConfig.ThinkingConfig.ThinkingLevel)
			},
		},
		{
			name:       "tools request",
			llmFile:    "llm-tools.request.json",
			geminiFile: "gemini-tools.request.json",
			validateFunc: func(t *testing.T, llmReq *llm.Request, geminiReq *GenerateContentRequest) {
				t.Helper()
				require.Len(t, geminiReq.Contents, 1)
				require.Equal(t, "user", geminiReq.Contents[0].Role)
				require.Equal(t, "What is the weather in San Francisco, CA?", geminiReq.Contents[0].Parts[0].Text)
				require.Len(t, geminiReq.Tools, 2)
				require.Equal(t, "get_coordinates", geminiReq.Tools[0].FunctionDeclarations[0].Name)
				require.Equal(t, "get_weather", geminiReq.Tools[1].FunctionDeclarations[0].Name)
			},
		},
		{
			name:       "thinking request",
			llmFile:    "llm-thinking.request.json",
			geminiFile: "gemini-thinking.request.json",
			validateFunc: func(t *testing.T, llmReq *llm.Request, geminiReq *GenerateContentRequest) {
				t.Helper()
				require.Len(t, geminiReq.Contents, 3)
				require.Equal(t, "user", geminiReq.Contents[0].Role)
				require.Equal(t, "assistant", geminiReq.Contents[1].Role)
				require.True(t, geminiReq.Contents[1].Parts[0].Thought)
				require.Contains(t, geminiReq.Contents[1].Parts[0].Text, "25 * 47")
				require.Equal(t, "user", geminiReq.Contents[2].Role)
				require.Equal(t, "high", geminiReq.GenerationConfig.ThinkingConfig.ThinkingLevel) // ThinkingLevel "high" takes priority
			},
		},
		{
			name:       "parallel multiple tools request",
			llmFile:    "llm-parallel_multiple_tool.request.json",
			geminiFile: "gemini-parallel_multiple_tool.request.json",
			validateFunc: func(t *testing.T, llmReq *llm.Request, geminiReq *GenerateContentRequest) {
				t.Helper()
				require.Len(t, geminiReq.Contents, 1)
				require.Equal(t, "user", geminiReq.Contents[0].Role)
				require.Len(t, geminiReq.Tools, 2)
				require.Equal(t, "get_weather", geminiReq.Tools[0].FunctionDeclarations[0].Name)
				require.Equal(t, "get_coordinates", geminiReq.Tools[1].FunctionDeclarations[0].Name)
			},
		},
	}

	dir := filepath.Join("testdata")

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			llmReq := loadLLMRequestFromFile(t, filepath.Join(dir, tc.llmFile))
			expectedGeminiReq := loadGeminiRequestFromFile(t, filepath.Join(dir, tc.geminiFile))

			result := convertLLMToGeminiRequest(llmReq)

			// Use expectedGeminiReq to avoid "declared and not used" error
			_ = expectedGeminiReq
			tc.validateFunc(t, llmReq, result)

			// Compare the generated request with the expected one
			// TODO: xtest.CompareGolden function not implemented yet
			// xtest.CompareGolden(t, expectedGeminiReq, result)
		})
	}
}

func TestConvertGeminiToLLMResponse_Testdata(t *testing.T) {
	testCases := []struct {
		name         string
		llmFile      string
		geminiFile   string
		validateFunc func(t *testing.T, llmResp *llm.Response, geminiResp *GenerateContentResponse)
	}{
		{
			name:       "simple response",
			llmFile:    "llm-simple.response.json",
			geminiFile: "gemini-simple.response.json",
			validateFunc: func(t *testing.T, llmResp *llm.Response, geminiResp *GenerateContentResponse) {
				t.Helper()
				require.Equal(t, llmResp.ID, geminiResp.ResponseID)
				require.Equal(t, llmResp.Model, geminiResp.ModelVersion)
				require.Len(t, geminiResp.Candidates, 1)
				require.Equal(t, "model", geminiResp.Candidates[0].Content.Role)
				require.Len(t, geminiResp.Candidates[0].Content.Parts, 1)
			},
		},
		{
			name:       "tools response",
			llmFile:    "llm-tools.response.json",
			geminiFile: "gemini-tools.response.json",
			validateFunc: func(t *testing.T, llmResp *llm.Response, geminiResp *GenerateContentResponse) {
				t.Helper()
				require.Len(t, geminiResp.Candidates, 1)
				require.Len(t, geminiResp.Candidates[0].Content.Parts, 3)
				// Text
				require.Equal(t, "I'll get the weather for you.", geminiResp.Candidates[0].Content.Parts[0].Text)
				// Tool call
				require.NotNil(t, geminiResp.Candidates[0].Content.Parts[1].FunctionCall)
				require.Equal(t, "get_coordinates", geminiResp.Candidates[0].Content.Parts[1].FunctionCall.Name)
				// Text
				require.Equal(t, "Now let me check the weather.", geminiResp.Candidates[0].Content.Parts[2].Text)
			},
		},
		{
			name:       "thinking response",
			llmFile:    "llm-thinking.response.json",
			geminiFile: "gemini-thinking.response.json",
			validateFunc: func(t *testing.T, llmResp *llm.Response, geminiResp *GenerateContentResponse) {
				t.Helper()
				require.Len(t, geminiResp.Candidates, 1)
				require.Len(t, geminiResp.Candidates[0].Content.Parts, 2)
				// Thoughts
				require.True(t, geminiResp.Candidates[0].Content.Parts[0].Thought)
				require.Contains(t, geminiResp.Candidates[0].Content.Parts[0].Text, "25 * 47")
				// Response
				require.False(t, geminiResp.Candidates[0].Content.Parts[1].Thought)
				require.Equal(t, "1175", geminiResp.Candidates[0].Content.Parts[1].Text)
			},
		},
		{
			name:       "parallel multiple tools response",
			llmFile:    "llm-parallel_multiple_tool.response.json",
			geminiFile: "gemini-parallel_multiple_tool.response.json",
			validateFunc: func(t *testing.T, llmResp *llm.Response, geminiResp *GenerateContentResponse) {
				t.Helper()
				require.Len(t, geminiResp.Candidates, 1)
				require.Len(t, geminiResp.Candidates[0].Content.Parts, 3)
				// Text
				require.Equal(t, "I'll help you get the weather and coordinates for San Francisco.", geminiResp.Candidates[0].Content.Parts[0].Text)
				// Tool call 1
				require.NotNil(t, geminiResp.Candidates[0].Content.Parts[1].FunctionCall)
				require.Equal(t, "get_weather", geminiResp.Candidates[0].Content.Parts[1].FunctionCall.Name)
				// Tool call 2
				require.NotNil(t, geminiResp.Candidates[0].Content.Parts[2].FunctionCall)
				require.Equal(t, "get_coordinates", geminiResp.Candidates[0].Content.Parts[2].FunctionCall.Name)
			},
		},
	}

	dir := filepath.Join("testdata")

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			llmResp := loadLLMResponseFromFile(t, filepath.Join(dir, tc.llmFile))
			expectedGeminiResp := loadGeminiResponseFromFile(t, filepath.Join(dir, tc.geminiFile))

			result := convertLLMToGeminiResponse(llmResp, false)

			// Use expectedGeminiResp to avoid "declared and not used" error
			_ = expectedGeminiResp
			tc.validateFunc(t, llmResp, result)

			// Compare the generated response with the expected one
			// TODO: xtest.CompareGolden function not implemented yet
			// xtest.CompareGolden(t, expectedGeminiResp, result)
		})
	}
}

func TestConvertGeminiToLLMResponseWithState_ToolCallIndex(t *testing.T) {
	input := &GenerateContentResponse{
		ResponseID:   "resp_index",
		ModelVersion: "gemini-2.5-flash",
		Candidates: []*Candidate{
			{
				Index: 0,
				Content: &Content{
					Role: "model",
					Parts: []*Part{
						{Text: "I'll call these functions"},
						{
							FunctionCall: &FunctionCall{
								ID:   "call_001",
								Name: "func1",
								Args: map[string]any{},
							},
						},
						{
							FunctionCall: &FunctionCall{
								ID:   "call_002",
								Name: "func2",
								Args: map[string]any{},
							},
						},
					},
				},
			},
		},
	}

	// Test with offset 5
	result, nextIndex := convertGeminiToLLMResponseWithState(input, false, 5)

	require.Len(t, result.Choices, 1)
	require.Len(t, result.Choices[0].Message.ToolCalls, 2)
	require.Equal(t, "call_001", result.Choices[0].Message.ToolCalls[0].ID)
	require.Equal(t, "call_002", result.Choices[0].Message.ToolCalls[1].ID)
	require.Equal(t, 7, nextIndex) // Should return 5 + 2 = 7
}

// Test helper functions

func loadLLMRequestFromFile(t *testing.T, path string) *llm.Request {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var req llm.Request
	err = json.Unmarshal(data, &req)
	require.NoError(t, err)

	return &req
}



// Test for ThoughtSignature handling
func TestConvertLLMMessageToGeminiContent_ThoughtSignature(t *testing.T) {
	tests := []struct {
		name     string
		input    *llm.Message
		validate func(t *testing.T, result *Content)
	}{
		{
			name: "message with tool calls but no thought signature",
			input: &llm.Message{
				Role: "assistant",
				Content: llm.MessageContent{
					Content: lo.ToPtr("I'll check the weather"),
				},
				ToolCalls: []llm.ToolCall{
					{
						ID:   "call_001",
						Type: "function",
						Function: llm.FunctionCall{
							Name:      "get_weather",
							Arguments: `{"location":"NYC"}`,
						},
					},
				},
			},
			validate: func(t *testing.T, result *Content) {
				t.Helper()
				require.Len(t, result.Parts, 2)
				require.Equal(t, "I'll check the weather", result.Parts[0].Text)
				require.NotNil(t, result.Parts[1].FunctionCall)
				require.Equal(t, "context_engineering_is_the_way_to_go", result.Parts[1].ThoughtSignature)
			},
		},
		{
			name: "message with tool calls and existing thought signature",
			input: &llm.Message{
				Role: "assistant",
				Content: llm.MessageContent{
					Content: lo.ToPtr("I'll check the weather"),
				},
				ToolCalls: []llm.ToolCall{
					{
						ID:   "call_001",
						Type: "function",
						Function: llm.FunctionCall{
							Name:      "get_weather",
							Arguments: `{"location":"NYC"}`,
						},
					},
				},
				RedactedReasoningContent: shared.EncodeGeminiThoughtSignature(lo.ToPtr("custom_signature")),
			},
			validate: func(t *testing.T, result *Content) {
				t.Helper()
				require.Len(t, result.Parts, 2)
				require.Equal(t, "I'll check the weather", result.Parts[0].Text)
				require.NotNil(t, result.Parts[1].FunctionCall)
				require.Equal(t, "custom_signature", result.Parts[1].ThoughtSignature)
			},
		},
		{
			name: "message with multiple tool calls",
			input: &llm.Message{
				Role: "assistant",
				Content: llm.MessageContent{
					Content: lo.ToPtr("I'll check both"),
				},
				ToolCalls: []llm.ToolCall{
					{
						ID:   "call_001",
						Type: "function",
						Function: llm.FunctionCall{
							Name:      "get_weather",
							Arguments: `{"location":"NYC"}`,
						},
					},
					{
						ID:   "call_002",
						Type: "function",
						Function: llm.FunctionCall{
							Name:      "get_time",
							Arguments: `{"timezone":"EST"}`,
						},
					},
				},
			},
			validate: func(t *testing.T, result *Content) {
				t.Helper()
				require.Len(t, result.Parts, 3)
				require.Equal(t, "I'll check both", result.Parts[0].Text)
				require.NotNil(t, result.Parts[1].FunctionCall)
				require.Equal(t, "call_001", result.Parts[1].FunctionCall.ID)
				require.Equal(t, "context_engineering_is_the_way_to_go", result.Parts[1].ThoughtSignature)
				require.NotNil(t, result.Parts[2].FunctionCall)
				require.Equal(t, "call_002", result.Parts[2].FunctionCall.ID)
				require.Equal(t, "", result.Parts[2].ThoughtSignature) // Only first function call gets signature
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertLLMMessageToGeminiContent(tt.input)
			tt.validate(t, result)
		})
	}
}

// Test for ExtraBody parsing
func TestConvertLLMToGeminiRequest_ExtraBody(t *testing.T) {
	tests := []struct {
		name     string
		input    *llm.Request
		validate func(t *testing.T, result *GenerateContentRequest)
	}{
		{
			name: "request with ExtraBody integer budget",
			input: &llm.Request{
				Messages: []llm.Message{
					{
						Role: "user",
						Content: llm.MessageContent{
							Content: lo.ToPtr("Test"),
						},
					},
				},
				ExtraBody: json.RawMessage(`{"google":{"thinking_config":{"thinking_budget":8192,"include_thoughts":true}}}`),
			},
			validate: func(t *testing.T, result *GenerateContentRequest) {
				t.Helper()
				require.NotNil(t, result.GenerationConfig)
				require.NotNil(t, result.GenerationConfig.ThinkingConfig)
				require.Equal(t, int64(8192), *result.GenerationConfig.ThinkingConfig.ThinkingBudget)
				require.True(t, result.GenerationConfig.ThinkingConfig.IncludeThoughts)
				require.Equal(t, "", result.GenerationConfig.ThinkingConfig.ThinkingLevel)
			},
		},
		{
			name: "request with ExtraBody invalid JSON",
			input: &llm.Request{
				Messages: []llm.Message{
					{
						Role: "user",
						Content: llm.MessageContent{
							Content: lo.ToPtr("Test"),
						},
					},
				},
				ExtraBody: json.RawMessage(`invalid json`),
			},
			validate: func(t *testing.T, result *GenerateContentRequest) {
				t.Helper()
				require.Nil(t, result.GenerationConfig)
			},
		},
		{
			name: "request with ExtraBody no google field",
			input: &llm.Request{
				Messages: []llm.Message{
					{
						Role: "user",
						Content: llm.MessageContent{
							Content: lo.ToPtr("Test"),
						},
					},
				},
				ExtraBody: json.RawMessage(`{"other_field":"value"}`),
			},
			validate: func(t *testing.T, result *GenerateContentRequest) {
				t.Helper()
				require.Nil(t, result.GenerationConfig)
			},
		},
		{
			name: "request with ExtraBody no thinking config",
			input: &llm.Request{
				Messages: []llm.Message{
					{
						Role: "user",
						Content: llm.MessageContent{
							Content: lo.ToPtr("Test"),
						},
					},
				},
				ExtraBody: json.RawMessage(`{"google":{"other_config":{}}}`),
			},
			validate: func(t *testing.T, result *GenerateContentRequest) {
				t.Helper()
				require.Nil(t, result.GenerationConfig)
			},
		},
		{
			name: "request with ExtraBody null thinking config",
			input: &llm.Request{
				Messages: []llm.Message{
					{
						Role: "user",
						Content: llm.MessageContent{
							Content: lo.ToPtr("Test"),
						},
					},
				},
				ExtraBody: json.RawMessage(`{"google":{"thinking_config":null}}`),
			},
			validate: func(t *testing.T, result *GenerateContentRequest) {
				t.Helper()
				require.Nil(t, result.GenerationConfig)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertLLMToGeminiRequest(tt.input)
			tt.validate(t, result)
		})
	}
}

// Test for convertLLMToGeminiRequestWithConfig
func TestConvertLLMToGeminiRequestWithConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    *llm.Request
		config   *Config
		validate func(t *testing.T, result *GenerateContentRequest)
	}{
		{
			name: "request with non-standard reasoning effort and config",
			input: &llm.Request{
				ReasoningEffort: "custom_effort",
				Messages: []llm.Message{
					{
						Role: "user",
						Content: llm.MessageContent{
							Content: lo.ToPtr("Test"),
						},
					},
				},
			},
			config: &Config{},
			validate: func(t *testing.T, result *GenerateContentRequest) {
				t.Helper()
				require.NotNil(t, result.GenerationConfig)
				require.NotNil(t, result.GenerationConfig.ThinkingConfig)
				require.Equal(t, "custom_effort", result.GenerationConfig.ThinkingConfig.ThinkingLevel)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertLLMToGeminiRequestWithConfig(tt.input, tt.config)
			tt.validate(t, result)
		})
	}
}