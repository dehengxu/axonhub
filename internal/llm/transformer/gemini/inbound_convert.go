package gemini

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/samber/lo"

	"github.com/looplj/axonhub/internal/llm"
	"github.com/looplj/axonhub/internal/pkg/xjson"
	geminioai "github.com/looplj/axonhub/internal/llm/transformer/gemini/openai"
	"github.com/looplj/axonhub/internal/llm/transformer/shared"
)

// convertGeminiToLLMRequest converts Gemini GenerateContentRequest to unified Request.
func convertGeminiToLLMRequest(geminiReq *GenerateContentRequest) (*llm.Request, error) {
	chatReq := &llm.Request{
		RequestType: llm.RequestTypeChat,
		APIFormat:   llm.APIFormatGeminiContents,
	}

	// Convert generation config
	if geminiReq.GenerationConfig != nil {
		gc := geminiReq.GenerationConfig

		if gc.MaxOutputTokens > 0 {
			chatReq.MaxTokens = lo.ToPtr(gc.MaxOutputTokens)
		}

		if gc.Temperature != nil {
			chatReq.Temperature = lo.ToPtr(*gc.Temperature)
		}

		if gc.TopP != nil {
			chatReq.TopP = lo.ToPtr(*gc.TopP)
		}

		if gc.PresencePenalty != nil {
			chatReq.PresencePenalty = lo.ToPtr(*gc.PresencePenalty)
		}

		if gc.FrequencyPenalty != nil {
			chatReq.FrequencyPenalty = lo.ToPtr(*gc.FrequencyPenalty)
		}

		if gc.Seed != nil {
			chatReq.Seed = lo.ToPtr(*gc.Seed)
		}

		if len(gc.StopSequences) > 0 {
			if len(gc.StopSequences) == 1 {
				chatReq.Stop = &llm.Stop{Stop: &gc.StopSequences[0]}
			} else {
				chatReq.Stop = &llm.Stop{MultipleStop: gc.StopSequences}
			}
		}

		// Convert thinking config to reasoning effort and preserve budget
		// Priority 1: Use ThinkingLevel if provided
		// Priority 2: Convert from ThinkingBudget if provided
		if gc.ThinkingConfig != nil {
			rawExtraBody, err := convertGeminiThinkingConfigToGeminiOpenAIExtraBody(gc.ThinkingConfig)
			if err != nil {
				return nil, err
			}

			chatReq.ExtraBody = rawExtraBody

			if gc.ThinkingConfig.ThinkingLevel != "" {
				// ThinkingLevel has priority - use it directly
				chatReq.ReasoningEffort = strings.ToLower(gc.ThinkingConfig.ThinkingLevel)
				// Gemini "minimal" maps to LLM "low" for consistency.
				if chatReq.ReasoningEffort == "minimal" {
					chatReq.ReasoningEffort = "low"
				}
			} else if gc.ThinkingConfig.ThinkingBudget != nil {
				// No ThinkingLevel, convert from ThinkingBudget
				chatReq.ReasoningEffort = thinkingBudgetToReasoningEffort(*gc.ThinkingConfig.ThinkingBudget)
			} else {
				// No level or budget, use default
				chatReq.ReasoningEffort = "medium"
			}
			// Always preserve the original budget if present
			if gc.ThinkingConfig.ThinkingBudget != nil {
				chatReq.ReasoningBudget = gc.ThinkingConfig.ThinkingBudget
			}
		}

		// Convert responseModalities to modalities
		if len(gc.ResponseModalities) > 0 {
			chatReq.Modalities = convertGeminiModalitiesToLLM(gc.ResponseModalities)
		}
	}

	// Convert system instruction
	messages := make([]llm.Message, 0)

	if geminiReq.SystemInstruction != nil {
		systemText := extractTextFromContent(geminiReq.SystemInstruction)
		if systemText != "" {
			messages = append(messages, llm.Message{
				Role: "system",
				Content: llm.MessageContent{
					Content: &systemText,
				},
			})
		}
	}

	// Convert contents to messages
	for i, content := range geminiReq.Contents {
		msg, err := convertGeminiContentToLLMMessage(content, geminiReq.Contents[:i])
		if err != nil {
			return nil, err
		}

		if msg != nil {
			messages = append(messages, *msg)
		}
	}

	chatReq.Messages = messages

	// Convert tools
	if len(geminiReq.Tools) > 0 {
		tools := make([]llm.Tool, 0)

		for _, tool := range geminiReq.Tools {
			// Handle function declarations
			if tool.FunctionDeclarations != nil {
				for _, fd := range tool.FunctionDeclarations {
					parameters := fd.Parameters
					// The gemini sdk use UPPER case for type, but the unified format use lower case.
					parameters, err := xjson.Transform(parameters, func(s *jsonschema.Schema) {
						s.Type = strings.ToLower(s.Type)
					})
					if err != nil {
						// If transform failed, fallback to the original parameters.
						parameters = fd.Parameters
					}

					llmTool := llm.Tool{
						Type: "function",
						Function: llm.Function{
							Name:        fd.Name,
							Description: fd.Description,
							Parameters:  parameters,
						},
					}
					tools = append(tools, llmTool)
				}
			}

			// Handle Google Search tool
			if tool.GoogleSearch != nil {
				llmTool := llm.Tool{
					Type: llm.ToolTypeGoogleSearch,
					Google: &llm.GoogleTools{
						Search: &llm.GoogleSearch{},
					},
				}
				tools = append(tools, llmTool)
			}

			// Handle Code Execution tool
			if tool.CodeExecution != nil {
				llmTool := llm.Tool{
					Type: llm.ToolTypeGoogleCodeExecution,
					Google: &llm.GoogleTools{
						CodeExecution: &llm.GoogleCodeExecution{},
					},
				}
				tools = append(tools, llmTool)
			}

			// Handle URL Context tool
			if tool.UrlContext != nil {
				llmTool := llm.Tool{
					Type: llm.ToolTypeGoogleUrlContext,
					Google: &llm.GoogleTools{
						UrlContext: &llm.GoogleUrlContext{},
					},
				}
				tools = append(tools, llmTool)
			}
		}

		chatReq.Tools = tools
	}

	// Convert tool config
	if geminiReq.ToolConfig != nil && geminiReq.ToolConfig.FunctionCallingConfig != nil {
		fcc := geminiReq.ToolConfig.FunctionCallingConfig
		chatReq.ToolChoice = convertGeminiFunctionCallingConfigToToolChoice(fcc)
	}

	return chatReq, nil
}

func convertGeminiThinkingConfigToGeminiOpenAIExtraBody(thinkingConfig *ThinkingConfig) (json.RawMessage, error) {
	if thinkingConfig == nil {
		return nil, nil
	}

	extraThinkingConfig := &geminioai.ThinkingConfig{
		IncludeThoughts: thinkingConfig.IncludeThoughts,
	}
	if thinkingConfig.ThinkingBudget != nil {
		budget := int(*thinkingConfig.ThinkingBudget)
		extraThinkingConfig.ThinkingBudget = geminioai.NewThinkingBudgetInt(budget)
	}

	if thinkingConfig.ThinkingLevel != "" {
		level := strings.ToLower(thinkingConfig.ThinkingLevel)
		if level == "minimal" {
			level = "low"
		}

		extraThinkingConfig.ThinkingLevel = level
	}

	extraBody := &geminioai.ExtraBody{
		Google: &geminioai.GoogleExtraBody{
			ThinkingConfig: extraThinkingConfig,
		},
	}

	return json.Marshal(extraBody)
}

// convertLLMToGeminiResponse converts LLM Response to Gemini GenerateContentResponse.
func convertLLMToGeminiResponse(llmResp *llm.Response, isStream bool) *GenerateContentResponse {
	geminiResp := &GenerateContentResponse{
		ResponseID:   llmResp.ID,
		ModelVersion: llmResp.Model,
		Candidates:   make([]*Candidate, 0),
	}

	// Convert choices to candidates
	for _, choice := range llmResp.Choices {
		candidate := convertLLMChoiceToGeminiCandidate(choice, isStream)
		geminiResp.Candidates = append(geminiResp.Candidates, &candidate)
	}

	// Convert usage information
	if llmResp.Usage != nil {
		geminiResp.UsageMetadata = &UsageMetadata{
			PromptTokenCount:        llmResp.Usage.PromptTokens,
			CandidatesTokenCount:    llmResp.Usage.CompletionTokens,
			TotalTokenCount:         llmResp.Usage.TotalTokens,
		}

		if llmResp.Usage.PromptTokensDetails.CachedTokens > 0 {
			geminiResp.UsageMetadata.CachedContentTokenCount = int64(llmResp.Usage.PromptTokensDetails.CachedTokens)
		}
		if llmResp.Usage.CompletionTokensDetails.ReasoningTokens > 0 {
			geminiResp.UsageMetadata.ThoughtsTokenCount = int64(llmResp.Usage.CompletionTokensDetails.ReasoningTokens)
		}
	}

	return geminiResp
}

// convertLLMChoiceToGeminiCandidate converts LLM Choice to Gemini Candidate.
func convertLLMChoiceToGeminiCandidate(choice llm.Choice, isStream bool) Candidate {
	candidate := Candidate{
		Index:        int64(choice.Index),
		FinishReason: convertLLMFinishReasonToGemini(choice.FinishReason),
	}

	// Handle streaming vs non-streaming content
	if isStream && choice.Delta != nil {
		// For streaming, use delta content
		content := convertLLMMessageToGeminiContent(&llm.Message{
			Role:    choice.Delta.Role,
			Content: choice.Delta.Content,
		})
		if content != nil {
			candidate.Content = content
		}
	} else if choice.Message != nil {
		// For non-streaming, use message content
		content := convertLLMMessageToGeminiContent(choice.Message)
		if content != nil {
			candidate.Content = content
		}
	}

	// Handle transformer metadata for GroundingMetadata
	if choice.TransformerMetadata != nil {
		if groundingMetadata, exists := choice.TransformerMetadata[TransformerMetadataKeyGroundingMetadata]; exists {
			if metadata, ok := groundingMetadata.(*GroundingMetadata); ok {
				candidate.GroundingMetadata = metadata
			}
		}
	}

	return candidate
}

// convertGeminiContentToLLMMessage converts Gemini Content to LLM Message.
func convertGeminiContentToLLMMessage(content *Content, previousContents []*Content) (*llm.Message, error) {
	if content == nil || len(content.Parts) == 0 {
		return nil, nil
	}

	msg := &llm.Message{
		Role:    convertGeminiRoleToLLMRole(content.Role),
		Content: llm.MessageContent{},
	}

	var textParts []string
	var reasoningContent *string
	var multipleContent []llm.MessageContentPart
	var toolCalls []llm.ToolCall

	for _, part := range content.Parts {
		if part.Text != "" {
			if part.Thought {
				// Add to reasoning content
				if reasoningContent == nil {
					reasoningContent = lo.ToPtr("")
				}
				*reasoningContent += part.Text
			} else {
				// Regular text content
				textParts = append(textParts, part.Text)
			}
		}

		// Handle inline data (images)
		if part.InlineData != nil {
			multipleContent = append(multipleContent, llm.MessageContentPart{
				Type: "image_url",
				ImageURL: &llm.ImageURL{
					URL: fmt.Sprintf("data:%s;base64,%s", part.InlineData.MIMEType, part.InlineData.Data),
				},
			})
		}

		// Handle file data
		if part.FileData != nil {
			multipleContent = append(multipleContent, llm.MessageContentPart{
				Type: "image_url",
				ImageURL: &llm.ImageURL{
					URL: part.FileData.FileURI,
				},
			})
		}

		// Handle function calls
		if part.FunctionCall != nil {
			args, err := json.Marshal(part.FunctionCall.Args)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal function call args: %w", err)
			}

			// Handle ThoughtSignature if present
			if len(part.ThoughtSignature) > 0 {
				signatureStr := string(part.ThoughtSignature)
				encoded := shared.EncodeGeminiThoughtSignature(&signatureStr)
				if encoded != nil {
					msg.RedactedReasoningContent = encoded
				}
			}

			toolCalls = append(toolCalls, llm.ToolCall{
				ID:   part.FunctionCall.ID,
				Type: "function",
				Function: llm.FunctionCall{
					Name:      part.FunctionCall.Name,
					Arguments: string(args),
				},
			})
		}

		// Handle function responses
		if part.FunctionResponse != nil {
			response, err := json.Marshal(part.FunctionResponse.Response)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal function response: %w", err)
			}

			msg.Content = llm.MessageContent{
				Content: lo.ToPtr(string(response)),
			}
			msg.Role = "tool"
			msg.ToolCallID = lo.ToPtr(part.FunctionResponse.ID)
			msg.ToolCallName = lo.ToPtr(part.FunctionResponse.Name)

			// Find function name from previous contents if not present
			if part.FunctionResponse.Name == "" {
				for _, prevContent := range previousContents {
					for _, prevPart := range prevContent.Parts {
						if prevPart.FunctionCall != nil && prevPart.FunctionCall.ID == part.FunctionResponse.ID {
							msg.ToolCallName = lo.ToPtr(prevPart.FunctionCall.Name)
							break
						}
					}
				}
			}

			return msg, nil
		}
	}

	// Set text content
	if len(textParts) > 0 {
		if len(multipleContent) > 0 {
			// Mix of text and other content types
			for _, text := range textParts {
				multipleContent = append([]llm.MessageContentPart{
					{Type: "text", Text: lo.ToPtr(text)},
				}, multipleContent...)
			}
			msg.Content.MultipleContent = multipleContent
		} else {
			// Text only
			msg.Content.Content = lo.ToPtr(strings.Join(textParts, ""))
		}
	} else if len(multipleContent) > 0 {
		// Non-text content only
		msg.Content.MultipleContent = multipleContent
	}

	// Set reasoning content
	if reasoningContent != nil {
		msg.ReasoningContent = reasoningContent
	}

	// Set tool calls
	if len(toolCalls) > 0 {
		msg.ToolCalls = toolCalls
	}

	return msg, nil
}