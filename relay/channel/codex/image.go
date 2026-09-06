package codex

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/relaykit/types"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

const defaultCodexImageChatModel = "gpt-5.6-sol"

func isCodexImageRelayMode(mode int) bool {
	return mode == relayconstant.RelayModeImagesGenerations || mode == relayconstant.RelayModeImagesEdits
}

func isCodexImageAlias(model string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(model)), "gpt-image-")
}

func resolveCodexImageUpstreamModel(info *relaycommon.RelayInfo, request dto.ImageRequest) string {
	model := ""
	if info != nil {
		model = strings.TrimSpace(info.UpstreamModelName)
	}
	if model == "" {
		model = strings.TrimSpace(request.Model)
	}
	if isCodexImageAlias(model) || model == "" {
		return defaultCodexImageChatModel
	}
	return model
}

func mapCodexImageToolSize(size string) string {
	switch strings.ToLower(strings.TrimSpace(size)) {
	case "1536x1024", "1792x1024":
		return "1536x1024"
	case "1024x1536", "1024x1792":
		return "1024x1536"
	default:
		return ""
	}
}

func convertCodexImageRequest(c *gin.Context, info *relaycommon.RelayInfo, request dto.ImageRequest) (dto.OpenAIResponsesRequest, error) {
	if request.N != nil && *request.N > 1 {
		return dto.OpenAIResponsesRequest{}, errors.New("codex channel: image n must be 1")
	}
	if maskPresent(request.Mask) {
		return dto.OpenAIResponsesRequest{}, errors.New("codex channel: image masks are not supported")
	}

	prompt := strings.TrimSpace(request.Prompt)
	if prompt == "" {
		return dto.OpenAIResponsesRequest{}, errors.New("codex channel: prompt is required")
	}

	refs, err := collectCodexReferenceImages(c, request)
	if err != nil {
		return dto.OpenAIResponsesRequest{}, err
	}
	if info != nil && info.RelayMode == relayconstant.RelayModeImagesEdits && len(refs) == 0 {
		return dto.OpenAIResponsesRequest{}, errors.New("codex channel: image is required for edits")
	}

	content := make([]map[string]any, 0, len(refs)+1)
	for _, ref := range refs {
		content = append(content, map[string]any{
			"type":      "input_image",
			"image_url": ref,
		})
	}
	content = append(content, map[string]any{
		"type": "input_text",
		"text": prompt,
	})
	inputBytes, err := common.Marshal([]map[string]any{
		{"role": "user", "content": content},
	})
	if err != nil {
		return dto.OpenAIResponsesRequest{}, err
	}

	tool := map[string]any{"type": dto.BuildInToolImageGeneration}
	if toolSize := mapCodexImageToolSize(request.Size); toolSize != "" {
		tool["size"] = toolSize
	}
	toolsBytes, err := common.Marshal([]map[string]any{tool})
	if err != nil {
		return dto.OpenAIResponsesRequest{}, err
	}
	toolChoiceBytes, err := common.Marshal(map[string]any{"type": dto.BuildInToolImageGeneration})
	if err != nil {
		return dto.OpenAIResponsesRequest{}, err
	}

	stream := true
	return dto.OpenAIResponsesRequest{
		Model:      resolveCodexImageUpstreamModel(info, request),
		Input:      inputBytes,
		Tools:      toolsBytes,
		ToolChoice: toolChoiceBytes,
		Stream:     &stream,
	}, nil
}

func maskPresent(raw json.RawMessage) bool {
	trimmed := strings.TrimSpace(string(raw))
	return trimmed != "" && trimmed != "null"
}

func collectCodexReferenceImages(c *gin.Context, request dto.ImageRequest) ([]string, error) {
	refs := make([]string, 0, 2)
	parsed, err := parseCodexImageField(request.Image)
	if err != nil {
		return nil, err
	}
	refs = append(refs, parsed...)
	parsed, err = parseCodexImageField(request.Images)
	if err != nil {
		return nil, err
	}
	refs = append(refs, parsed...)

	if c == nil || c.Request == nil {
		return refs, nil
	}
	if !strings.Contains(c.Request.Header.Get("Content-Type"), "multipart/form-data") && c.Request.MultipartForm == nil {
		return refs, nil
	}

	form := c.Request.MultipartForm
	if form == nil {
		parsedForm, parseErr := common.ParseMultipartFormReusable(c)
		if parseErr != nil {
			return nil, fmt.Errorf("codex channel: parse image form failed: %w", parseErr)
		}
		form = parsedForm
	}
	if form != nil && len(form.File["mask"]) > 0 {
		return nil, errors.New("codex channel: image masks are not supported")
	}
	for _, header := range multipartImageFiles(form) {
		dataURL, fileErr := multipartFileToDataURL(header)
		if fileErr != nil {
			return nil, fileErr
		}
		refs = append(refs, dataURL)
	}
	return refs, nil
}

func parseCodexImageField(raw json.RawMessage) ([]string, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return nil, nil
	}
	switch common.GetJsonType(trimmed) {
	case "string":
		var value string
		if err := common.Unmarshal(trimmed, &value); err != nil {
			return nil, fmt.Errorf("codex channel: invalid image field: %w", err)
		}
		if normalized := normalizeCodexImageRef(value); normalized != "" {
			return []string{normalized}, nil
		}
		return nil, nil
	case "array":
		var items []json.RawMessage
		if err := common.Unmarshal(trimmed, &items); err != nil {
			return nil, fmt.Errorf("codex channel: invalid image field: %w", err)
		}
		refs := make([]string, 0, len(items))
		for _, item := range items {
			parsed, err := parseCodexImageField(item)
			if err != nil {
				return nil, err
			}
			refs = append(refs, parsed...)
		}
		return refs, nil
	case "object":
		var obj map[string]any
		if err := common.Unmarshal(trimmed, &obj); err != nil {
			return nil, fmt.Errorf("codex channel: invalid image field: %w", err)
		}
		if url, _ := obj["url"].(string); strings.TrimSpace(url) != "" {
			return []string{normalizeCodexImageRef(url)}, nil
		}
		if url, _ := obj["image_url"].(string); strings.TrimSpace(url) != "" {
			return []string{normalizeCodexImageRef(url)}, nil
		}
		if b64, _ := obj["b64_json"].(string); strings.TrimSpace(b64) != "" {
			return []string{normalizeCodexImageRef(b64)}, nil
		}
		return nil, nil
	default:
		return nil, errors.New("codex channel: invalid image field")
	}
}

func normalizeCodexImageRef(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "data:") || strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value
	}
	return "data:image/png;base64," + value
}

func multipartImageFiles(form *multipart.Form) []*multipart.FileHeader {
	if form == nil || form.File == nil {
		return nil
	}
	if files := form.File["image"]; len(files) > 0 {
		return files
	}
	if files := form.File["image[]"]; len(files) > 0 {
		return files
	}
	var files []*multipart.FileHeader
	for name, headers := range form.File {
		if strings.HasPrefix(name, "image[") {
			files = append(files, headers...)
		}
	}
	return files
}

func multipartFileToDataURL(header *multipart.FileHeader) (string, error) {
	if header == nil {
		return "", errors.New("codex channel: image file is required")
	}
	file, err := header.Open()
	if err != nil {
		return "", fmt.Errorf("codex channel: open image failed: %w", err)
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("codex channel: read image failed: %w", err)
	}
	contentType := header.Header.Get("Content-Type")
	if contentType == "" || contentType == "application/octet-stream" {
		contentType = sniffImageContentType(data)
	}
	return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(data), nil
}

func sniffImageContentType(data []byte) string {
	switch {
	case bytes.HasPrefix(data, []byte{0x89, 'P', 'N', 'G'}):
		return "image/png"
	case bytes.HasPrefix(data, []byte{0xff, 0xd8}):
		return "image/jpeg"
	case bytes.HasPrefix(data, []byte("RIFF")) && bytes.Contains(data[:minInt(16, len(data))], []byte("WEBP")):
		return "image/webp"
	case bytes.HasPrefix(data, []byte("GIF8")):
		return "image/gif"
	default:
		return "image/png"
	}
}

func minInt(a, b int) int {
	return lo.Ternary(a < b, a, b)
}

func handleCodexImageResponse(c *gin.Context, resp *http.Response) (*dto.Usage, *types.NewAPIError) {
	if resp == nil || resp.Body == nil {
		return nil, types.NewOpenAIError(errors.New("codex channel: empty image response"), types.ErrorCodeBadResponse, http.StatusInternalServerError)
	}
	defer service.CloseResponseBodyGracefully(resp)

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, types.NewOpenAIError(err, types.ErrorCodeReadResponseBodyFailed, http.StatusInternalServerError)
	}

	result, usage, parseErr := parseCodexImageStream(raw)
	if parseErr != nil {
		return nil, parseErr
	}
	if strings.TrimSpace(result.B64Json) == "" {
		return nil, types.NewOpenAIError(errors.New("codex channel: no image_generation_call result"), types.ErrorCodeBadResponse, http.StatusBadGateway)
	}

	body, marshalErr := common.Marshal(dto.ImageResponse{
		Created: common.GetTimestamp(),
		Data:    []dto.ImageData{result},
	})
	if marshalErr != nil {
		return nil, types.NewOpenAIError(marshalErr, types.ErrorCodeBadResponseBody, http.StatusInternalServerError)
	}
	c.Writer.Header().Set("Content-Type", "application/json")
	service.IOCopyBytesGracefully(c, nil, body)
	return usage, nil
}

type codexImageGenUsage struct {
	InputTokens        int `json:"input_tokens"`
	OutputTokens       int `json:"output_tokens"`
	TotalTokens        int `json:"total_tokens"`
	InputTokensDetails struct {
		ImageTokens int `json:"image_tokens"`
		TextTokens  int `json:"text_tokens"`
	} `json:"input_tokens_details"`
	OutputTokensDetails struct {
		ImageTokens int `json:"image_tokens"`
	} `json:"output_tokens_details"`
}

func parseCodexImageStream(raw []byte) (dto.ImageData, *dto.Usage, *types.NewAPIError) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return dto.ImageData{}, nil, types.NewOpenAIError(errors.New("codex channel: empty image stream"), types.ErrorCodeBadResponseBody, http.StatusBadGateway)
	}
	if trimmed[0] == '{' {
		var payload struct {
			Error   any    `json:"error"`
			Message string `json:"message"`
		}
		if err := common.Unmarshal(trimmed, &payload); err == nil {
			if oai := dto.GetOpenAIError(payload.Error); oai != nil && oai.Message != "" {
				return dto.ImageData{}, nil, types.WithOpenAIError(*oai, http.StatusBadGateway)
			}
			if payload.Message != "" {
				return dto.ImageData{}, nil, types.NewOpenAIError(errors.New(payload.Message), types.ErrorCodeBadResponse, http.StatusBadGateway)
			}
		}
		return dto.ImageData{}, nil, types.NewOpenAIError(errors.New("codex channel: unexpected JSON image response"), types.ErrorCodeBadResponseBody, http.StatusBadGateway)
	}

	var image dto.ImageData
	usage := &dto.Usage{}
	var streamErr *types.NewAPIError
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" || payload == "[DONE]" {
			continue
		}

		var event struct {
			Type  string `json:"type"`
			Error any    `json:"error"`
			Item  *struct {
				Type          string `json:"type"`
				Status        string `json:"status"`
				Result        string `json:"result"`
				RevisedPrompt string `json:"revised_prompt"`
			} `json:"item"`
			Response *struct {
				Status    string     `json:"status"`
				Error     any        `json:"error"`
				Usage     *dto.Usage `json:"usage"`
				ToolUsage struct {
					ImageGen *codexImageGenUsage `json:"image_gen"`
				} `json:"tool_usage"`
			} `json:"response"`
		}
		if err := common.UnmarshalJsonStr(payload, &event); err != nil {
			continue
		}
		if oai := dto.GetOpenAIError(event.Error); oai != nil && oai.Message != "" {
			streamErr = types.WithOpenAIError(*oai, http.StatusBadGateway)
			continue
		}
		if event.Item != nil && event.Item.Type == dto.ResponsesOutputTypeImageGenerationCall {
			if event.Item.RevisedPrompt != "" {
				image.RevisedPrompt = event.Item.RevisedPrompt
			}
			if event.Item.Result != "" {
				image.B64Json = event.Item.Result
			}
		}
		if event.Response == nil {
			continue
		}
		if oai := dto.GetOpenAIError(event.Response.Error); oai != nil && oai.Message != "" {
			streamErr = types.WithOpenAIError(*oai, http.StatusBadGateway)
		}
		if event.Response.Usage != nil {
			usage = event.Response.Usage
		}
		if event.Response.ToolUsage.ImageGen != nil {
			applyCodexImageGenUsage(usage, event.Response.ToolUsage.ImageGen)
		}
	}
	if streamErr != nil && image.B64Json == "" {
		return dto.ImageData{}, nil, streamErr
	}
	if usage.TotalTokens == 0 {
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}
	return image, usage, nil
}

func applyCodexImageGenUsage(usage *dto.Usage, imageGen *codexImageGenUsage) {
	if usage == nil || imageGen == nil {
		return
	}
	if imageGen.InputTokens > 0 {
		usage.InputTokens = imageGen.InputTokens
		usage.PromptTokens = imageGen.InputTokens
	}
	if imageGen.OutputTokens > 0 {
		usage.OutputTokens = imageGen.OutputTokens
		usage.CompletionTokens = imageGen.OutputTokens
	}
	if imageGen.TotalTokens > 0 {
		usage.TotalTokens = imageGen.TotalTokens
	}
	usage.PromptTokensDetails.ImageTokens = imageGen.InputTokensDetails.ImageTokens
	usage.PromptTokensDetails.TextTokens = imageGen.InputTokensDetails.TextTokens
	usage.CompletionTokenDetails.ImageTokens = imageGen.OutputTokensDetails.ImageTokens
	if usage.InputTokensDetails == nil && (imageGen.InputTokensDetails.ImageTokens > 0 || imageGen.InputTokensDetails.TextTokens > 0) {
		usage.InputTokensDetails = &dto.InputTokenDetails{
			ImageTokens: imageGen.InputTokensDetails.ImageTokens,
			TextTokens:  imageGen.InputTokensDetails.TextTokens,
		}
	}
}
