package service

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relaykit/dto"
	kitutil "github.com/QuantumNous/new-api/relaykit/relayconvert/kitutil"
	"github.com/QuantumNous/new-api/relaykit/types"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/gin-gonic/gin"
)

const promptAuditMaxRunes = 8000
const promptAuditNonTextPlaceholder = "[non-text content]"

// PromptSnapshot is the employee-side text extracted from a relay request.
type PromptSnapshot struct {
	LastUserText string
	MessageCount int
	RelayFormat  string
}

// CapturePromptAudit records the last user message of a relay request.
// Safe to call on blocked requests; write failures never affect the user request.
func CapturePromptAudit(c *gin.Context, request dto.Request, relayInfo *relaycommon.RelayInfo, blocked bool, blockedWords []string) {
	if request == nil || relayInfo == nil {
		return
	}
	if relayInfo.IsChannelTest || relayInfo.UserId == 0 {
		return
	}
	snap := ExtractLastUserPrompt(request, relayInfo.RelayFormat)
	if snap.LastUserText == "" && snap.MessageCount == 0 && !blocked {
		return
	}
	if snap.LastUserText == "" {
		snap.LastUserText = promptAuditNonTextPlaceholder
	}

	audit := &model.PromptAudit{
		RequestId:    relayInfo.RequestId,
		UserId:       relayInfo.UserId,
		CreatedAt:    common.GetTimestamp(),
		Username:     common.GetContextKeyString(c, constant.ContextKeyUserName),
		TokenName:    c.GetString("token_name"),
		TokenId:      relayInfo.TokenId,
		ModelName:    relayInfo.OriginModelName,
		Group:        relayInfo.UsingGroup,
		Ip:           c.ClientIP(),
		RelayFormat:  snap.RelayFormat,
		LastUserText: snap.LastUserText,
		MessageCount: snap.MessageCount,
		Blocked:      blocked,
		BlockedWords: strings.Join(blockedWords, ", "),
	}
	if audit.RequestId == "" {
		audit.RequestId = common.GetContextKeyString(c, common.RequestIdKey)
	}
	if audit.Username == "" {
		audit.Username = c.GetString("username")
	}

	gopool.Go(func() {
		if err := model.CreatePromptAudit(audit); err != nil {
			common.SysLog("failed to record prompt audit: " + err.Error())
		}
	})
}

// ExtractLastUserPrompt returns only the latest user turn, not full history.
func ExtractLastUserPrompt(request dto.Request, relayFormat types.RelayFormat) PromptSnapshot {
	snap := PromptSnapshot{RelayFormat: string(relayFormat)}
	if request == nil {
		return snap
	}
	switch req := request.(type) {
	case *dto.GeneralOpenAIRequest:
		snap.MessageCount = len(req.Messages)
		snap.LastUserText = lastOpenAIUserText(req)
	case *dto.ClaudeRequest:
		snap.MessageCount = len(req.Messages)
		snap.LastUserText = lastClaudeUserText(req)
	case *dto.GeminiChatRequest:
		snap.MessageCount = len(req.Contents)
		snap.LastUserText = lastGeminiUserText(req)
	case *dto.OpenAIResponsesRequest:
		text, count := lastResponsesUserText(req)
		snap.LastUserText = text
		snap.MessageCount = count
	case *dto.ImageRequest:
		snap.LastUserText = strings.TrimSpace(req.Prompt)
		if snap.LastUserText != "" {
			snap.MessageCount = 1
		}
	case *dto.AudioRequest:
		snap.LastUserText = strings.TrimSpace(req.Input)
		if snap.LastUserText != "" {
			snap.MessageCount = 1
		}
	case *dto.EmbeddingRequest:
		inputs := req.ParseInput()
		snap.MessageCount = len(inputs)
		if n := len(inputs); n > 0 {
			snap.LastUserText = strings.TrimSpace(inputs[n-1])
		}
	case *dto.RerankRequest:
		snap.LastUserText = strings.TrimSpace(req.Query)
		if snap.LastUserText != "" {
			snap.MessageCount = 1
		}
	}
	snap.LastUserText = truncateRunes(strings.TrimSpace(snap.LastUserText), promptAuditMaxRunes)
	return snap
}

func lastOpenAIUserText(req *dto.GeneralOpenAIRequest) string {
	if req == nil {
		return ""
	}
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if strings.EqualFold(req.Messages[i].Role, "user") {
			return openaiMessageText(&req.Messages[i])
		}
	}
	if req.Prompt == nil {
		return ""
	}
	switch v := req.Prompt.(type) {
	case string:
		return v
	case []any:
		for i := len(v) - 1; i >= 0; i-- {
			if s, ok := v[i].(string); ok && strings.TrimSpace(s) != "" {
				return s
			}
		}
	default:
		return fmt.Sprintf("%v", v)
	}
	return ""
}

func openaiMessageText(message *dto.Message) string {
	if message == nil {
		return ""
	}
	parts := message.ParseContent()
	texts := make([]string, 0, len(parts))
	for _, part := range parts {
		if part.Type == dto.ContentTypeText && strings.TrimSpace(part.Text) != "" {
			texts = append(texts, part.Text)
		}
	}
	return strings.TrimSpace(strings.Join(texts, "\n"))
}

func lastClaudeUserText(req *dto.ClaudeRequest) string {
	if req == nil {
		return ""
	}
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if strings.EqualFold(req.Messages[i].Role, "user") {
			return strings.TrimSpace(req.Messages[i].GetStringContent())
		}
	}
	return strings.TrimSpace(req.Prompt)
}

func lastGeminiUserText(req *dto.GeminiChatRequest) string {
	if req == nil {
		return ""
	}
	for i := len(req.Contents) - 1; i >= 0; i-- {
		content := req.Contents[i]
		if content.Role != "" && !strings.EqualFold(content.Role, "user") {
			continue
		}
		texts := make([]string, 0, len(content.Parts))
		for _, part := range content.Parts {
			if part.Thought {
				continue
			}
			if strings.TrimSpace(part.Text) != "" {
				texts = append(texts, part.Text)
			}
		}
		joined := strings.TrimSpace(strings.Join(texts, "\n"))
		if joined != "" || strings.EqualFold(content.Role, "user") {
			return joined
		}
	}
	return ""
}

func lastResponsesUserText(req *dto.OpenAIResponsesRequest) (string, int) {
	if req == nil || len(req.Input) == 0 {
		return "", 0
	}
	if kitutil.GetJsonType(req.Input) == "string" {
		var text string
		_ = kitutil.Unmarshal(req.Input, &text)
		text = strings.TrimSpace(text)
		if text == "" {
			return "", 0
		}
		return text, 1
	}
	if kitutil.GetJsonType(req.Input) != "array" {
		return "", 0
	}
	var items []dto.Input
	if err := kitutil.Unmarshal(req.Input, &items); err != nil || len(items) == 0 {
		inputs := req.ParseInput()
		for i := len(inputs) - 1; i >= 0; i-- {
			if inputs[i].Type == "input_text" && strings.TrimSpace(inputs[i].Text) != "" {
				return strings.TrimSpace(inputs[i].Text), len(inputs)
			}
		}
		return "", len(inputs)
	}
	for i := len(items) - 1; i >= 0; i-- {
		item := items[i]
		if item.Role != "" && !strings.EqualFold(item.Role, "user") {
			continue
		}
		text := responsesInputText(item)
		if text != "" || strings.EqualFold(item.Role, "user") {
			return text, len(items)
		}
	}
	return "", len(items)
}

func responsesInputText(item dto.Input) string {
	if len(item.Content) == 0 {
		return ""
	}
	if kitutil.GetJsonType(item.Content) == "string" {
		var text string
		_ = kitutil.Unmarshal(item.Content, &text)
		return strings.TrimSpace(text)
	}
	if kitutil.GetJsonType(item.Content) != "array" {
		return ""
	}
	var parts []map[string]any
	if err := kitutil.Unmarshal(item.Content, &parts); err != nil {
		return ""
	}
	texts := make([]string, 0, len(parts))
	for _, part := range parts {
		typeVal, _ := part["type"].(string)
		if typeVal != "" && typeVal != "input_text" && typeVal != "text" && typeVal != "output_text" {
			continue
		}
		if text, ok := part["text"].(string); ok && strings.TrimSpace(text) != "" {
			texts = append(texts, text)
		}
	}
	return strings.TrimSpace(strings.Join(texts, "\n"))
}

func truncateRunes(s string, max int) string {
	if max <= 0 || s == "" {
		return s
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}
