package service

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/relaykit/dto"
	kitutil "github.com/QuantumNous/new-api/relaykit/relayconvert/kitutil"
	"github.com/QuantumNous/new-api/relaykit/types"

	"github.com/stretchr/testify/require"
)

func TestExtractLastUserPromptOpenAIKeepsOnlyLatestUser(t *testing.T) {
	req := &dto.GeneralOpenAIRequest{
		Messages: []dto.Message{
			{Role: "system", Content: "you are a helper"},
			{Role: "user", Content: "first question"},
			{Role: "assistant", Content: "first answer"},
			{Role: "user", Content: "latest question"},
		},
	}
	snap := ExtractLastUserPrompt(req, types.RelayFormatOpenAI)
	require.Equal(t, "latest question", snap.LastUserText)
	require.Equal(t, 4, snap.MessageCount)
}

func TestExtractLastUserPromptOpenAIReadsTextParts(t *testing.T) {
	req := &dto.GeneralOpenAIRequest{
		Messages: []dto.Message{
			{
				Role: "user",
				Content: []any{
					map[string]any{"type": "image_url", "image_url": map[string]any{"url": "data:image/png;base64,AAA"}},
					map[string]any{"type": "text", "text": "what is in this image?"},
				},
			},
		},
	}
	snap := ExtractLastUserPrompt(req, types.RelayFormatOpenAI)
	require.Equal(t, "what is in this image?", snap.LastUserText)
	require.NotContains(t, snap.LastUserText, "base64")
}

func TestExtractLastUserPromptClaudeAndGemini(t *testing.T) {
	claude := &dto.ClaudeRequest{
		Messages: []dto.ClaudeMessage{
			{Role: "user", Content: "old"},
			{Role: "assistant", Content: "reply"},
			{Role: "user", Content: "new claude question"},
		},
	}
	gemini := &dto.GeminiChatRequest{
		Contents: []dto.GeminiChatContent{
			{Role: "user", Parts: []dto.GeminiPart{{Text: "old"}}},
			{Role: "model", Parts: []dto.GeminiPart{{Text: "reply"}}},
			{Role: "user", Parts: []dto.GeminiPart{{Text: "new gemini question"}}},
		},
	}
	require.Equal(t, "new claude question", ExtractLastUserPrompt(claude, types.RelayFormatClaude).LastUserText)
	require.Equal(t, "new gemini question", ExtractLastUserPrompt(gemini, types.RelayFormatGemini).LastUserText)
}

func TestExtractLastUserPromptImageAndResponses(t *testing.T) {
	image := &dto.ImageRequest{Prompt: "draw a cat"}
	require.Equal(t, "draw a cat", ExtractLastUserPrompt(image, types.RelayFormatOpenAIImage).LastUserText)

	input, err := kitutil.Marshal("hello responses")
	require.NoError(t, err)
	responses := &dto.OpenAIResponsesRequest{Input: input}
	require.Equal(t, "hello responses", ExtractLastUserPrompt(responses, types.RelayFormatOpenAIResponses).LastUserText)
}

func TestExtractLastUserPromptTruncates(t *testing.T) {
	req := &dto.GeneralOpenAIRequest{
		Messages: []dto.Message{
			{Role: "user", Content: strings.Repeat("问", promptAuditMaxRunes+20)},
		},
	}
	snap := ExtractLastUserPrompt(req, types.RelayFormatOpenAI)
	require.Equal(t, promptAuditMaxRunes+1, len([]rune(snap.LastUserText)))
	require.True(t, strings.HasSuffix(snap.LastUserText, "…"))
}
