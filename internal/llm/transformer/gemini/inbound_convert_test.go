package gemini

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/looplj/axonhub/internal/llm"
	geminioai "github.com/looplj/axonhub/internal/llm/transformer/gemini/openai"
)

// =============================================================================
// Basic Tests for convertGeminiToLLMRequest
// =============================================================================

func TestConvertGeminiToLLMRequest_Basic(t *testing.T) {
	tests := []struct {
		name     string
		input    *GenerateContentRequest
		validate func(t *testing.T, result *llm.Request)
	}{
		{
			name: "simple text request",
			input: &GenerateContentRequest{
				Contents: []*Content{
					{
						Role: "user",
						Parts: []*Part{
							{Text: "Hello, Gemini!"},
						},
					},
				},
				GenerationConfig: &GenerationConfig{
					MaxOutputTokens: 1024,
				},
			},
			validate: func(t *testing.T, result *llm.Request) {
				t.Helper()
				require.NotNil(t, result)
				require.Equal(t, llm.APIFormatGeminiContents, result.RawAPIFormat)
				require.Len(t, result.Messages, 1)
				require.Equal(t, "user", result.Messages[0].Role)
				require.Equal(t, "Hello, Gemini!", *result.Messages[0].Content.Content)
				require.Equal(t, int64(1024), *result.MaxTokens)
			},
		},
		{
			name: "request with system instruction",
			input: &GenerateContentRequest{
				SystemInstruction: &Content{
					Parts: []*Part{
						{Text: "You are a helpful assistant."},
					},
				},
				Contents: []*Content{
					{
						Role: "user",
						Parts: []*Part{
							{Text: "Hello!"},
						},
					},
				},
			},
			validate: func(t *testing.T, result *llm.Request) {
				t.Helper()
				require.Len(t, result.Messages, 2)
				require.Equal(t, "system", result.Messages[0].Role)
				require.Equal(t, "You are a helpful assistant.", *result.Messages[0].Content.Content)
				require.Equal(t, "user", result.Messages[1].Role)
			},
		},
		{
			name: "request with generation config",
			input: &GenerateContentRequest{
				Contents: []*Content{
					{
						Role: "user",
						Parts: []*Part{
							{Text: "Test"},
						},
					},
				},
				GenerationConfig: &GenerationConfig{
					MaxOutputTokens:  2048,
					Temperature:      lo.ToPtr(0.7),
					TopP:             lo.ToPtr(0.9),
					PresencePenalty:  lo.ToPtr(0.5),
					FrequencyPenalty: lo.ToPtr(0.3),
					Seed:             lo.ToPtr(int64(42)),
					StopSequences:    []string{"END", "STOP"},
				},
			},
			validate: func(t *testing.T, result *llm.Request) {
				t.Helper()
				require.Equal(t, int64(2048), *result.MaxTokens)
				require.InDelta(t, 0.7, *result.Temperature, 0.01)
				require.InDelta(t, 0.9, *result.TopP, 0.01)
				require.InDelta(t, 0.5, *result.PresencePenalty, 0.01)
				require.InDelta(t, 0.3, *result.FrequencyPenalty, 0.01)
				require.Equal(t, int64(42), *result.Seed)
				require.NotNil(t, result.Stop)
				require.Equal(t, []string{"END", "STOP"}, result.Stop.MultipleStop)
			},
		},
		{
			name: "request with single stop sequence",
			input: &GenerateContentRequest{
				Contents: []*Content{
					{
						Role: "user",
						Parts: []*Part{
							{Text: "Test"},
						},
					},
				},
				GenerationConfig: &GenerationConfig{
					StopSequences: []string{"END"},
				},
			},
			validate: func(t *testing.T, result *llm.Request) {
				t.Helper()
				require.NotNil(t, result.Stop)
				require.Equal(t, "END", *result.Stop.Stop)
			},
		},
		{
			name: "request with thinking config",
			input: &GenerateContentRequest{
				Contents: []*Content{
					{
						Role: "user",
						Parts: []*Part{
							{Text: "Solve this problem"},
						},
					},
				},
				GenerationConfig: &GenerationConfig{
					MaxOutputTokens: 4096,
					ThinkingConfig: &ThinkingConfig{
						IncludeThoughts: true,
						ThinkingBudget:  lo.ToPtr(int64(8192)),
					},
				},
			},
			validate: func(t *testing.T, result *llm.Request) {
				t.Helper()
				require.Equal(t, "medium", result.ReasoningEffort)
				require.NotEmpty(t, result.ExtraBody)

				var extra geminioai.ExtraBody

				err := json.Unmarshal(result.ExtraBody, &extra)
				require.NoError(t, err)
				require.NotNil(t, extra.Google)
				require.NotNil(t, extra.Google.ThinkingConfig)
				require.True(t, extra.Google.ThinkingConfig.IncludeThoughts)
				require.Empty(t, extra.Google.ThinkingConfig.ThinkingLevel)
				require.NotNil(t, extra.Google.ThinkingConfig.ThinkingBudget)
				require.NotNil(t, extra.Google.ThinkingConfig.ThinkingBudget.IntValue)
				require.Equal(t, 8192, *extra.Google.ThinkingConfig.ThinkingBudget.IntValue)
			},
		},
		{
			name: "request with thinking config low budget",
			input: &GenerateContentRequest{
				Contents: []*Content{
					{
						Role: "user",
						Parts: []*Part{
							{Text: "Quick question"},
						},
					},
				},
				GenerationConfig: &GenerationConfig{
					ThinkingConfig: &ThinkingConfig{
						IncludeThoughts: true,
						ThinkingBudget:  lo.ToPtr(int64(512)),
					},
				},
			},
			validate: func(t *testing.T, result *llm.Request) {
				t.Helper()
				require.Equal(t, "low", result.ReasoningEffort)
			},
		},
		{
			name: "request with thinking config high budget",
			input: &GenerateContentRequest{
				Contents: []*Content{
					{
						Role: "user",
						Parts: []*Part{
							{Text: "Complex problem"},
						},
					},
				},
				GenerationConfig: &GenerationConfig{
					ThinkingConfig: &ThinkingConfig{
						IncludeThoughts: true,
						ThinkingBudget:  lo.ToPtr(int64(32768)),
					},
				},
			},
			validate: func(t *testing.T, result *llm.Request) {
				t.Helper()
				require.Equal(t, "high", result.ReasoningEffort)
			},
		},
		{
			name: "request with thinking config no budget",
			input: &GenerateContentRequest{
				Contents: []*Content{
					{
						Role: "user",
						Parts: []*Part{
							{Text: "Question"},
						},
					},
				},
				GenerationConfig: &GenerationConfig{
					ThinkingConfig: &ThinkingConfig{
						IncludeThoughts: true,
					},
				},
			},
			validate: func(t *testing.T, result *llm.Request) {
				t.Helper()
				require.Equal(t, "medium", result.ReasoningEffort)
			},
		},
		{
			name: "request with thinking config and budget preservation",
			input: &GenerateContentRequest{
				Contents: []*Content{
					{
						Role: "user",
						Parts: []*Part{
							{Text: "Question"},
						},
					},
				},
				GenerationConfig: &GenerationConfig{
					ThinkingConfig: &ThinkingConfig{
						IncludeThoughts: true,
						ThinkingBudget:  lo.ToPtr(int64(5000)),
					},
				},
			},
			validate: func(t *testing.T, result *llm.Request) {
				t.Helper()
				require.Equal(t, "medium", result.ReasoningEffort)
				require.NotNil(t, result.ReasoningBudget)
				require.Equal(t, int64(5000), *result.ReasoningBudget)
			},
		},
		{
			name: "request with thinking level priority - high level",
			input: &GenerateContentRequest{
				Contents: []*Content{
					{
						Role: "user",
						Parts: []*Part{
							{Text: "Complex question"},
						},
					},
				},
				GenerationConfig: &GenerationConfig{
					ThinkingConfig: &ThinkingConfig{
						IncludeThoughts: true,
						ThinkingLevel:   "high",
						ThinkingBudget:  lo.ToPtr(int64(1024)), // Budget is low, but level should take priority
					},
				},
			},
			validate: func(t *testing.T, result *llm.Request) {
				t.Helper()
				require.Equal(t, "high", result.ReasoningEffort) // Should use level, not budget
				require.NotNil(t, result.ReasoningBudget)
				require.Equal(t, int64(1024), *result.ReasoningBudget) // Budget should be preserved
				require.NotEmpty(t, result.ExtraBody)

				var extra geminioai.ExtraBody

				err := json.Unmarshal(result.ExtraBody, &extra)
				require.NoError(t, err)
				require.NotNil(t, extra.Google)
				require.NotNil(t, extra.Google.ThinkingConfig)
				require.True(t, extra.Google.ThinkingConfig.IncludeThoughts)
				require.Equal(t, "high", extra.Google.ThinkingConfig.ThinkingLevel)
				require.NotNil(t, extra.Google.ThinkingConfig.ThinkingBudget)
				require.NotNil(t, extra.Google.ThinkingConfig.ThinkingBudget.IntValue)
				require.Equal(t, 1024, *extra.Google.ThinkingConfig.ThinkingBudget.IntValue)
			},
		},
		{
			name: "request with thinking level priority - low level",
			input: &GenerateContentRequest{
				Contents: []*Content{
					{
						Role: "user",
						Parts: []*Part{
							{Text: "Simple question"},
						},
					},
				},
				GenerationConfig: &GenerationConfig{
					ThinkingConfig: &ThinkingConfig{
						IncludeThoughts: true,
						ThinkingLevel:   "low",
						ThinkingBudget:  lo.ToPtr(int64(32768)), // Budget is high, but level should take priority
					},
				},
			},
			validate: func(t *testing.T, result *llm.Request) {
				t.Helper()
				require.Equal(t, "low", result.ReasoningEffort) // Should use level, not budget
				require.NotNil(t, result.ReasoningBudget)
				require.Equal(t, int64(32768), *result.ReasoningBudget) // Budget should be preserved
			},
		},
		{
			name: "request with thinking level only - no budget",
			input: &GenerateContentRequest{
				Contents: []*Content{
					{
						Role: "user",
						Parts: []*Part{
							{Text: "Question"},
						},
					},
				},
				GenerationConfig: &GenerationConfig{
					ThinkingConfig: &ThinkingConfig{
						IncludeThoughts: true,
						ThinkingLevel:   "high",
					},
				},
			},
			validate: func(t *testing.T, result *llm.Request) {
				t.Helper()
				require.Equal(t, "high", result.ReasoningEffort)
				require.Nil(t, result.ReasoningBudget) // No budget provided
				require.NotEmpty(t, result.ExtraBody)

				var extra geminioai.ExtraBody

				err := json.Unmarshal(result.ExtraBody, &extra)
				require.NoError(t, err)
				require.NotNil(t, extra.Google)
				require.NotNil(t, extra.Google.ThinkingConfig)
				require.True(t, extra.Google.ThinkingConfig.IncludeThoughts)
				require.Equal(t, "high", extra.Google.ThinkingConfig.ThinkingLevel)
				require.Nil(t, extra.Google.ThinkingConfig.ThinkingBudget)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertGeminiToLLMRequest(tt.input)
			require.NoError(t, err)
			tt.validate(t, result)
		})
	}
}

func TestConvertGeminiToLLMRequest_Tools(t *testing.T) {
	tests := []struct {
		name     string
		input    *GenerateContentRequest
		validate func(t *testing.T, result *llm.Request)
	}{
		{
			name: "request with tools",
			input: &GenerateContentRequest{
				Contents: []*Content{
					{
						Role: "user",
						Parts: []*Part{
							{Text: "What's the weather?"},
						},
					},
				},
				Tools: []*Tool{
					{
						FunctionDeclarations: []*FunctionDeclaration{
							{
								Name:        "get_weather",
								Description: "Get weather information",
								Parameters:  json.RawMessage(`{"type":"object","properties":{"location":{"type":"string"}}}`),
							},
						},
					},
				},
			},
			validate: func(t *testing.T, result *llm.Request) {
				t.Helper()
				require.Len(t, result.Tools, 1)
				require.Equal(t, "function", result.Tools[0].Type)
				require.Equal(t, "get_weather", result.Tools[0].Function.Name)
				require.Equal(t, "Get weather information", result.Tools[0].Function.Description)
			},
		},
		{
			name: "request with multiple tools",
			input: &GenerateContentRequest{
				Contents: []*Content{
					{
						Role: "user",
						Parts: []*Part{
							{Text: "Help me"},
						},
					},
				},
				Tools: []*Tool{
					{
						FunctionDeclarations: []*FunctionDeclaration{
							{
								Name:        "tool1",
								Description: "First tool",
							},
							{
								Name:        "tool2",
								Description: "Second tool",
							},
						},
					},
				},
			},
			validate: func(t *testing.T, result *llm.Request) {
				t.Helper()
				require.Len(t, result.Tools, 2)
				require.Equal(t, "tool1", result.Tools[0].Function.Name)
				require.Equal(t, "tool2", result.Tools[1].Function.Name)
			},
		},
		{
			name: "request with google search and code execution tools",
			input: &GenerateContentRequest{
				Contents: []*Content{
					{
						Role: "user",
						Parts: []*Part{
							{Text: "Search and run"},
						},
					},
				},
				Tools: []*Tool{
					{GoogleSearch: &GoogleSearch{}},
					{CodeExecution: &CodeExecution{}},
				},
			},
			validate: func(t *testing.T, result *llm.Request) {
				t.Helper()
				require.Len(t, result.Tools, 2)
				require.Equal(t, llm.ToolTypeGoogleSearch, result.Tools[0].Type)
				require.NotNil(t, result.Tools[0].Google)
				require.NotNil(t, result.Tools[0].Google.Search)
				require.Equal(t, llm.ToolTypeGoogleCodeExecution, result.Tools[1].Type)
				require.NotNil(t, result.Tools[1].Google)
				require.NotNil(t, result.Tools[1].Google.CodeExecution)
			},
		},
		{
			name: "request with url context tool",
			input: &GenerateContentRequest{
				Contents: []*Content{
					{
						Role: "user",
						Parts: []*Part{
							{Text: "Fetch URL content"},
						},
					},
				},
				Tools: []*Tool{
					{UrlContext: &UrlContext{}},
				},
			},
			validate: func(t *testing.T, result *llm.Request) {
				t.Helper()
				require.Len(t, result.Tools, 1)
				require.Equal(t, llm.ToolTypeGoogleUrlContext, result.Tools[0].Type)
				require.NotNil(t, result.Tools[0].Google)
				require.NotNil(t, result.Tools[0].Google.UrlContext)
			},
		},
		{
			name: "request with all grounding tools",
			input: &GenerateContentRequest{
				Contents: []*Content{
					{
						Role: "user",
						Parts: []*Part{
							{Text: "Use all tools"},
						},
					},
				},
				Tools: []*Tool{
					{GoogleSearch: &GoogleSearch{}},
					{CodeExecution: &CodeExecution{}},
					{UrlContext: &UrlContext{}},
				},
			},
			validate: func(t *testing.T, result *llm.Request) {
				t.Helper()
				require.Len(t, result.Tools, 3)
				require.Equal(t, llm.ToolTypeGoogleSearch, result.Tools[0].Type)
				require.Equal(t, llm.ToolTypeGoogleCodeExecution, result.Tools[1].Type)
				require.Equal(t, llm.ToolTypeGoogleUrlContext, result.Tools[2].Type)
			},
		},
		{
			name: "request with tool config AUTO",
			input: &GenerateContentRequest{
				Contents: []*Content{
					{
						Role: "user",
						Parts: []*Part{
							{Text: "Test"},
						},
					},
				},
				ToolConfig: &ToolConfig{
					FunctionCallingConfig: &FunctionCallingConfig{
						Mode: "AUTO",
					},
				},
			},
			validate: func(t *testing.T, result *llm.Request) {
				t.Helper()
				require.NotNil(t, result.ToolChoice)
				require.Equal(t, "auto", *result.ToolChoice.ToolChoice)
			},
		},
		{
			name: "request with tool config NONE",
			input: &GenerateContentRequest{
				Contents: []*Content{
					{
						Role: "user",
						Parts: []*Part{
							{Text: "Test"},
						},
					},
				},
				ToolConfig: &ToolConfig{
					FunctionCallingConfig: &FunctionCallingConfig{
						Mode: "NONE",
					},
				},
			},
			validate: func(t *testing.T, result *llm.Request) {
				t.Helper()
				require.NotNil(t, result.ToolChoice)
				require.Equal(t, "none", *result.ToolChoice.ToolChoice)
			},
		},
		{
			name: "request with tool config ANY",
			input: &GenerateContentRequest{
				Contents: []*Content{
					{
						Role: "user",
						Parts: []*Part{
							{Text: "Test"},
						},
					},
				},
				ToolConfig: &ToolConfig{
					FunctionCallingConfig: &FunctionCallingConfig{
						Mode: "ANY",
					},
				},
			},
			validate: func(t *testing.T, result *llm.Request) {
				t.Helper()
				require.NotNil(t, result.ToolChoice)
				require.Equal(t, "required", *result.ToolChoice.ToolChoice)
			},
		},
		{
			name: "request with tool config ANY with specific function",
			input: &GenerateContentRequest{
				Contents: []*Content{
					{
						Role: "user",
						Parts: []*Part{
							{Text: "Test"},
						},
					},
				},
				ToolConfig: &ToolConfig{
					FunctionCallingConfig: &FunctionCallingConfig{
						Mode:                 "ANY",
						AllowedFunctionNames: []string{"specific_function"},
					},
				},
			},
			validate: func(t *testing.T, result *llm.Request) {
				t.Helper()
				require.NotNil(t, result.ToolChoice)
				require.NotNil(t, result.ToolChoice.NamedToolChoice)
				require.Equal(t, "function", result.ToolChoice.NamedToolChoice.Type)
				require.Equal(t, "specific_function", result.ToolChoice.NamedToolChoice.Function.Name)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertGeminiToLLMRequest(tt.input)
			require.NoError(t, err)
			tt.validate(t, result)
		})
	}
}

func TestConvertGeminiContentToLLMMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    *Content
		validate func(t *testing.T, result *llm.Message)
	}{
		{
			name:  "nil content",
			input: nil,
			validate: func(t *testing.T, result *llm.Message) {
				t.Helper()
				require.Nil(t, result)
			},
		},
		{
			name: "empty parts",
			input: &Content{
				Role:  "user",
				Parts: []*Part{},
			},
			validate: func(t *testing.T, result *llm.Message) {
				t.Helper()
				require.Nil(t, result)
			},
		},
		{
			name: "text content",
			input: &Content{
				Role: "user",
				Parts: []*Part{
					{Text: "Hello"},
				},
			},
			validate: func(t *testing.T, result *llm.Message) {
				t.Helper()
				require.NotNil(t, result)
				require.Equal(t, "user", result.Role)
				require.Equal(t, "Hello", *result.Content.Content)
			},
		},
		{
			name: "model role conversion",
			input: &Content{
				Role: "model",
				Parts: []*Part{
					{Text: "Response"},
				},
			},
			validate: func(t *testing.T, result *llm.Message) {
				t.Helper()
				require.Equal(t, "assistant", result.Role)
			},
		},
		{
			name: "thinking content",
			input: &Content{
				Role: "model",
				Parts: []*Part{
					{Text: "Let me think...", Thought: true},
					{Text: "The answer is 42"},
				},
			},
			validate: func(t *testing.T, result *llm.Message) {
				t.Helper()
				require.NotNil(t, result.ReasoningContent)
				require.Equal(t, "Let me think...", *result.ReasoningContent)
				require.Equal(t, "The answer is 42", *result.Content.Content)
			},
		},
		{
			name: "inline data (image)",
			input: &Content{
				Role: "user",
				Parts: []*Part{
					{
						InlineData: &Blob{
							MIMEType: "image/jpeg",
							Data:     "base64data",
						},
					},
				},
			},
			validate: func(t *testing.T, result *llm.Message) {
				t.Helper()
				require.Len(t, result.Content.MultipleContent, 1)
				require.Equal(t, "image_url", result.Content.MultipleContent[0].Type)
				require.Equal(t, "data:image/jpeg;base64,base64data", result.Content.MultipleContent[0].ImageURL.URL)
			},
		},
		{
			name: "file data",
			input: &Content{
				Role: "user",
				Parts: []*Part{
					{
						FileData: &FileData{
							MIMEType: "image/png",
							FileURI:  "gs://bucket/file.png",
						},
					},
				},
			},
			validate: func(t *testing.T, result *llm.Message) {
				t.Helper()
				require.Len(t, result.Content.MultipleContent, 1)
				require.Equal(t, "image_url", result.Content.MultipleContent[0].Type)
				require.Equal(t, "gs://bucket/file.png", result.Content.MultipleContent[0].ImageURL.URL)
			},
		},
		{
			name: "function call",
			input: &Content{
				Role: "model",
				Parts: []*Part{
					{
						FunctionCall: &FunctionCall{
							ID:   "call_123",
							Name: "get_weather",
							Args: map[string]any{"location": "NYC"},
						},
					},
				},
			},
			validate: func(t *testing.T, result *llm.Message) {
				t.Helper()
				require.Len(t, result.ToolCalls, 1)
				require.Equal(t, "call_123", result.ToolCalls[0].ID)
				require.Equal(t, "function", result.ToolCalls[0].Type)
				require.Equal(t, "get_weather", result.ToolCalls[0].Function.Name)
				require.Contains(t, result.ToolCalls[0].Function.Arguments, "NYC")
			},
		},
		{
			name: "function response",
			input: &Content{
				Role: "user",
				Parts: []*Part{
					{
						FunctionResponse: &FunctionResponse{
							ID:       "call_123",
							Name:     "get_weather",
							Response: map[string]any{"temperature": 72},
						},
					},
				},
			},
			validate: func(t *testing.T, result *llm.Message) {
				t.Helper()
				require.Equal(t, "tool", result.Role)
				require.Equal(t, "call_123", *result.ToolCallID)
				require.Contains(t, *result.Content.Content, "72")
			},
		},
		{
			name: "multiple text parts",
			input: &Content{
				Role: "user",
				Parts: []*Part{
					{Text: "First part"},
					{Text: "Second part"},
				},
			},
			validate: func(t *testing.T, result *llm.Message) {
				t.Helper()
				require.Len(t, result.Content.MultipleContent, 2)
				require.Equal(t, "text", result.Content.MultipleContent[0].Type)
				require.Equal(t, "First part", *result.Content.MultipleContent[0].Text)
				require.Equal(t, "Second part", *result.Content.MultipleContent[1].Text)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := convertGeminiContentToLLMMessage(tt.input, nil)
			require.NoError(t, err)
			tt.validate(t, result)
		})
	}
}

// =============================================================================
// Basic Tests for convertLLMToGeminiResponse
// =============================================================================

func TestConvertLLMToGeminiResponse_Basic(t *testing.T) {
	tests := []struct {
		name     string
		input    *llm.Response
		validate func(t *testing.T, result *GenerateContentResponse)
	}{
		{
			name: "simple response",
			input: &llm.Response{
				ID:    "resp_123",
				Model: "gemini-2.5-flash",
				Choices: []llm.Choice{
					{
						Index: 0,
						Message: &llm.Message{
							Role: "assistant",
							Content: llm.MessageContent{
								Content: lo.ToPtr("Hello!"),
							},
						},
						FinishReason: lo.ToPtr("stop"),
					},
				},
			},
			validate: func(t *testing.T, result *GenerateContentResponse) {
				t.Helper()
				require.Equal(t, "resp_123", result.ResponseID)
				require.Equal(t, "gemini-2.5-flash", result.ModelVersion)
				require.Len(t, result.Candidates, 1)
				require.Equal(t, "model", result.Candidates[0].Content.Role)
				require.Len(t, result.Candidates[0].Content.Parts, 1)
				require.Equal(t, "Hello!", result.Candidates[0].Content.Parts[0].Text)
				require.Equal(t, "STOP", result.Candidates[0].FinishReason)
			},
		},
		{
			name: "response with thinking",
			input: &llm.Response{
				ID:    "resp_think",
				Model: "gemini-2.5-flash",
				Choices: []llm.Choice{
					{
						Index: 0,
						Message: &llm.Message{
							Role:             "assistant",
							ReasoningContent: lo.ToPtr("Let me think..."),
							Content: llm.MessageContent{
								Content: lo.ToPtr("The answer is 42"),
							},
						},
					},
				},
			},
			validate: func(t *testing.T, result *GenerateContentResponse) {
				t.Helper()
				require.Len(t, result.Candidates[0].Content.Parts, 2)
				require.True(t, result.Candidates[0].Content.Parts[0].Thought)
				require.Equal(t, "Let me think...", result.Candidates[0].Content.Parts[0].Text)
				require.False(t, result.Candidates[0].Content.Parts[1].Thought)
				require.Equal(t, "The answer is 42", result.Candidates[0].Content.Parts[1].Text)
			},
		},
		{
			name: "response with tool calls",
			input: &llm.Response{
				ID:    "resp_tool",
				Model: "gemini-2.5-flash",
				Choices: []llm.Choice{
					{
						Index: 0,
						Message: &llm.Message{
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
					},
				},
			},
			validate: func(t *testing.T, result *GenerateContentResponse) {
				t.Helper()
				require.Len(t, result.Candidates[0].Content.Parts, 2)
				require.Equal(t, "I'll check the weather", result.Candidates[0].Content.Parts[0].Text)
				require.NotNil(t, result.Candidates[0].Content.Parts[1].FunctionCall)
				require.Equal(t, "call_001", result.Candidates[0].Content.Parts[1].FunctionCall.ID)
				require.Equal(t, "get_weather", result.Candidates[0].Content.Parts[1].FunctionCall.Name)
				require.Equal(t, "NYC", result.Candidates[0].Content.Parts[1].FunctionCall.Args["location"])
			},
		},
		{
			name: "response with usage",
			input: &llm.Response{
				ID:    "resp_usage",
				Model: "gemini-2.5-flash",
				Choices: []llm.Choice{
					{
						Index: 0,
						Message: &llm.Message{
							Role: "assistant",
							Content: llm.MessageContent{
								Content: lo.ToPtr("Response"),
							},
						},
					},
				},
				Usage: &llm.Usage{
					PromptTokens:     100,
					CompletionTokens: 50,
					TotalTokens:      150,
					PromptTokensDetails: &llm.PromptTokensDetails{
						CachedTokens: 20,
					},
					CompletionTokensDetails: &llm.CompletionTokensDetails{
						ReasoningTokens: 30,
					},
				},
			},
			validate: func(t *testing.T, result *GenerateContentResponse) {
				t.Helper()
				require.NotNil(t, result.UsageMetadata)
				require.Equal(t, int64(100), result.UsageMetadata.PromptTokenCount)
				require.Equal(t, int64(20), result.UsageMetadata.CandidatesTokenCount)
				require.Equal(t, int64(150), result.UsageMetadata.TotalTokenCount)
				require.Equal(t, int64(20), result.UsageMetadata.CachedContentTokenCount)
				require.Equal(t, int64(30), result.UsageMetadata.ThoughtsTokenCount)
			},
		},
		{
			name: "response with multiple content parts",
			input: &llm.Response{
				ID:    "resp_multi",
				Model: "gemini-2.5-flash",
				Choices: []llm.Choice{
					{
						Index: 0,
						Message: &llm.Message{
							Role: "assistant",
							Content: llm.MessageContent{
								MultipleContent: []llm.MessageContentPart{
									{Type: "text", Text: lo.ToPtr("First part")},
									{Type: "text", Text: lo.ToPtr("Second part")},
								},
							},
						},
					},
				},
			},
			validate: func(t *testing.T, result *GenerateContentResponse) {
				t.Helper()
				require.Len(t, result.Candidates[0].Content.Parts, 2)
				require.Equal(t, "First part", result.Candidates[0].Content.Parts[0].Text)
				require.Equal(t, "Second part", result.Candidates[0].Content.Parts[1].Text)
			},
		},
		{
			name: "response with delta instead of message",
			input: &llm.Response{
				ID:    "resp_delta",
				Model: "gemini-2.5-flash",
				Choices: []llm.Choice{
					{
						Index: 0,
						Delta: &llm.Message{
							Role: "assistant",
							Content: llm.MessageContent{
								Content: lo.ToPtr("Streaming content"),
							},
						},
					},
				},
			},
			validate: func(t *testing.T, result *GenerateContentResponse) {
				t.Helper()
				require.Len(t, result.Candidates[0].Content.Parts, 1)
				require.Equal(t, "Streaming content", result.Candidates[0].Content.Parts[0].Text)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertLLMToGeminiResponse(tt.input, false)
			tt.validate(t, result)
		})
	}
}

func TestConvertLLMToGeminiResponse_FinishReasons(t *testing.T) {
	finishReasons := map[string]string{
		"stop":           "STOP",
		"length":         "MAX_TOKENS",
		"content_filter": "SAFETY",
		"tool_calls":     "STOP",
		"unknown":        "STOP",
	}

	for llmReason, expectedGeminiReason := range finishReasons {
		t.Run("finish_reason_"+llmReason, func(t *testing.T) {
			input := &llm.Response{
				ID:    "resp_finish",
				Model: "gemini-2.5-flash",
				Choices: []llm.Choice{
					{
						Index: 0,
						Message: &llm.Message{
							Role: "assistant",
							Content: llm.MessageContent{
								Content: lo.ToPtr("Test"),
							},
						},
						FinishReason: lo.ToPtr(llmReason),
					},
				},
			}

			result := convertLLMToGeminiResponse(input, false)
			require.Equal(t, expectedGeminiReason, result.Candidates[0].FinishReason)
		})
	}
}

// =============================================================================
// Testdata Tests
// =============================================================================

func TestConvertGeminiToLLMRequest_Testdata(t *testing.T) {
	testCases := []struct {
		name         string
		geminiFile   string
		validateFunc func(t *testing.T, result *llm.Request)
	}{
		{
			name:       "simple request",
			geminiFile: "gemini-simple.request.json",
			validateFunc: func(t *testing.T, result *llm.Request) {
				t.Helper()
				require.Len(t, result.Messages, 1)
				require.Equal(t, "user", result.Messages[0].Role)
				require.Equal(t, "Output 1-20, 5 each line", *result.Messages[0].Content.Content)
				require.Equal(t, int64(4096), *result.MaxTokens)
				require.Equal(t, "low", result.ReasoningEffort)
			},
		},
		{
			name:       "tools request",
			geminiFile: "gemini-tools.request.json",
			validateFunc: func(t *testing.T, result *llm.Request) {
				t.Helper()
				require.Len(t, result.Messages, 1)
				require.Equal(t, "What is the weather in San Francisco, CA?", *result.Messages[0].Content.Content)
				require.Len(t, result.Tools, 2)
				require.Equal(t, "get_coordinates", result.Tools[0].Function.Name)
				require.Equal(t, "get_weather", result.Tools[1].Function.Name)
			},
		},
		{
			name:       "thinking request",
			geminiFile: "gemini-thinking.request.json",
			validateFunc: func(t *testing.T, result *llm.Request) {
				t.Helper()
				require.Len(t, result.Messages, 3)
				require.Equal(t, "user", result.Messages[0].Role)
				require.Equal(t, "assistant", result.Messages[1].Role)
				require.NotNil(t, result.Messages[1].ReasoningContent)
				require.Contains(t, *result.Messages[1].ReasoningContent, "25 * 47")
				require.Equal(t, "user", result.Messages[2].Role)
				require.Equal(t, "high", result.ReasoningEffort) // ThinkingLevel "high" takes priority
			},
		},
		{
			name:       "parallel multiple tools request",
			geminiFile: "gemini-parallel_multiple_tool.request.json",
			validateFunc: func(t *testing.T, result *llm.Request) {
				t.Helper()
				require.Len(t, result.Messages, 1)
				require.Equal(t, "user", result.Messages[0].Role)
				require.Len(t, result.Tools, 2)
				require.Equal(t, "get_weather", result.Tools[0].Function.Name)
				require.Equal(t, "get_coordinates", result.Tools[1].Function.Name)
			},
		},
	}

	dir := filepath.Join("testdata")

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			geminiReq := loadGeminiRequestFromFile(t, filepath.Join(dir, tc.geminiFile))

			result, err := convertGeminiToLLMRequest(geminiReq)
			require.NoError(t, err)
			tc.validateFunc(t, result)
		})
	}
}

func TestConvertLLMToGeminiResponse_Testdata(t *testing.T) {
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

func loadGeminiRequestFromFile(t *testing.T, path string) *GenerateContentRequest {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var req GenerateContentRequest
	err = json.Unmarshal(data, &req)
	require.NoError(t, err)

	return &req
}

func loadLLMResponseFromFile(t *testing.T, path string) *llm.Response {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var resp llm.Response
	err = json.Unmarshal(data, &resp)
	require.NoError(t, err)

	return &resp
}

func loadGeminiResponseFromFile(t *testing.T, path string) *GenerateContentResponse {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var resp GenerateContentResponse
	err = json.Unmarshal(data, &resp)
	require.NoError(t, err)

	return &resp
}