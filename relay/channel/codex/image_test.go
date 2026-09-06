package codex

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestMapCodexImageToolSize(t *testing.T) {
	assert.Equal(t, "", mapCodexImageToolSize(""))
	assert.Equal(t, "", mapCodexImageToolSize("1024x1024"))
	assert.Equal(t, "", mapCodexImageToolSize("512x512"))
	assert.Equal(t, "1536x1024", mapCodexImageToolSize("1536x1024"))
	assert.Equal(t, "1536x1024", mapCodexImageToolSize("1792x1024"))
	assert.Equal(t, "1024x1536", mapCodexImageToolSize("1024x1536"))
	assert.Equal(t, "1024x1536", mapCodexImageToolSize("1024x1792"))
}

func TestConvertImageRequestTextToImage(t *testing.T) {
	adaptor := &Adaptor{}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeCodex,
			UpstreamModelName: "gpt-image-1",
		},
		RelayMode: relayconstant.RelayModeImagesGenerations,
	}

	converted, err := adaptor.ConvertImageRequest(nil, info, dto.ImageRequest{
		Model:  "gpt-image-1",
		Prompt: "a red square",
		N:      lo.ToPtr(uint(1)),
		Size:   "1024x1024",
	})
	require.NoError(t, err)

	request, ok := converted.(dto.OpenAIResponsesRequest)
	require.True(t, ok)
	assert.Equal(t, "gpt-5.6-sol", request.Model)
	require.NotNil(t, request.Stream)
	assert.True(t, *request.Stream)
	assert.Equal(t, "false", string(request.Store))
	assert.Equal(t, "array", common.GetJsonType(request.Input))
	assert.Equal(t, "input_text", gjson.GetBytes(request.Input, "0.content.0.type").String())
	assert.Equal(t, "a red square", gjson.GetBytes(request.Input, "0.content.0.text").String())
	assert.Equal(t, "image_generation", gjson.GetBytes(request.Tools, "0.type").String())
	assert.False(t, gjson.GetBytes(request.Tools, "0.size").Exists())
	assert.Equal(t, "image_generation", gjson.GetBytes(request.ToolChoice, "type").String())
}

func TestConvertImageRequestPassesLandscapeSize(t *testing.T) {
	adaptor := &Adaptor{}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeCodex,
			UpstreamModelName: "gpt-5.6-sol",
		},
		RelayMode: relayconstant.RelayModeImagesGenerations,
	}

	converted, err := adaptor.ConvertImageRequest(nil, info, dto.ImageRequest{
		Model:  "gpt-5.6-sol",
		Prompt: "navy landscape",
		Size:   "1536x1024",
	})
	require.NoError(t, err)
	request := converted.(dto.OpenAIResponsesRequest)
	assert.Equal(t, "gpt-5.6-sol", request.Model)
	assert.Equal(t, "1536x1024", gjson.GetBytes(request.Tools, "0.size").String())
}

func TestConvertImageRequestRejectsNAndMask(t *testing.T) {
	adaptor := &Adaptor{}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeCodex},
		RelayMode:   relayconstant.RelayModeImagesGenerations,
	}

	_, err := adaptor.ConvertImageRequest(nil, info, dto.ImageRequest{
		Model:  "gpt-image-1",
		Prompt: "cat",
		N:      lo.ToPtr(uint(2)),
	})
	require.ErrorContains(t, err, "n must be 1")

	_, err = adaptor.ConvertImageRequest(nil, info, dto.ImageRequest{
		Model:  "gpt-image-1",
		Prompt: "cat",
		Mask:   json.RawMessage(`"data:image/png;base64,AAA"`),
	})
	require.ErrorContains(t, err, "masks are not supported")
}

func TestConvertImageRequestEditsAddsInputImage(t *testing.T) {
	adaptor := &Adaptor{}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:       constant.ChannelTypeCodex,
			UpstreamModelName: "gpt-image-2",
		},
		RelayMode: relayconstant.RelayModeImagesEdits,
	}

	converted, err := adaptor.ConvertImageRequest(nil, info, dto.ImageRequest{
		Model:  "gpt-image-2",
		Prompt: "add a white circle",
		Image:  json.RawMessage(`"data:image/png;base64,AAAA"`),
	})
	require.NoError(t, err)
	request := converted.(dto.OpenAIResponsesRequest)
	assert.Equal(t, "gpt-5.6-sol", request.Model)
	assert.Equal(t, "input_image", gjson.GetBytes(request.Input, "0.content.0.type").String())
	assert.Equal(t, "data:image/png;base64,AAAA", gjson.GetBytes(request.Input, "0.content.0.image_url").String())
	assert.Equal(t, "input_text", gjson.GetBytes(request.Input, "0.content.1.type").String())
}

func TestConvertImageRequestEditsRequiresImage(t *testing.T) {
	adaptor := &Adaptor{}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeCodex},
		RelayMode:   relayconstant.RelayModeImagesEdits,
	}
	_, err := adaptor.ConvertImageRequest(nil, info, dto.ImageRequest{
		Model:  "gpt-image-1",
		Prompt: "edit this",
	})
	require.ErrorContains(t, err, "image is required")
}

func TestGetRequestURLImagesUsesCodexResponses(t *testing.T) {
	adaptor := &Adaptor{}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelType:    constant.ChannelTypeCodex,
			ChannelBaseUrl: "https://chatgpt.com",
		},
		RelayMode: relayconstant.RelayModeImagesGenerations,
	}
	url, err := adaptor.GetRequestURL(info)
	require.NoError(t, err)
	assert.Equal(t, "https://chatgpt.com/backend-api/codex/responses", url)
}

func TestDoResponseImagesCollectsSSE(t *testing.T) {
	oldMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(oldMode) })

	sse := "" +
		"event: response.output_item.done\n" +
		"data: {\"type\":\"response.output_item.done\",\"item\":{\"type\":\"image_generation_call\",\"status\":\"completed\",\"result\":\"iVBORw0KGgo\",\"revised_prompt\":\"solid red square\"}}\n\n" +
		"event: response.completed\n" +
		"data: {\"type\":\"response.completed\",\"response\":{\"status\":\"completed\",\"error\":null,\"usage\":{\"input_tokens\":10,\"output_tokens\":20,\"total_tokens\":30},\"tool_usage\":{\"image_gen\":{\"input_tokens\":50,\"output_tokens\":229,\"total_tokens\":279,\"input_tokens_details\":{\"image_tokens\":16,\"text_tokens\":34},\"output_tokens_details\":{\"image_tokens\":229}}}}}\n\n"

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(sse)),
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeCodex},
		RelayMode:   relayconstant.RelayModeImagesGenerations,
		IsStream:    true,
	}

	usageAny, apiErr := (&Adaptor{}).DoResponse(c, resp, info)
	require.Nil(t, apiErr)
	usage, ok := usageAny.(*dto.Usage)
	require.True(t, ok)
	require.NotNil(t, usage)
	assert.Equal(t, 50, usage.PromptTokens)
	assert.Equal(t, 229, usage.CompletionTokens)
	assert.Equal(t, 16, usage.PromptTokensDetails.ImageTokens)
	assert.Equal(t, 229, usage.CompletionTokenDetails.ImageTokens)

	var image dto.ImageResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &image))
	require.Len(t, image.Data, 1)
	assert.Equal(t, "iVBORw0KGgo", image.Data[0].B64Json)
	assert.Equal(t, "solid red square", image.Data[0].RevisedPrompt)
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
}

func TestDoResponseImagesStreamError(t *testing.T) {
	oldMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(oldMode) })

	sse := "event: response.failed\n" +
		"data: {\"type\":\"response.failed\",\"response\":{\"status\":\"failed\",\"error\":{\"type\":\"service_unavailable_error\",\"message\":\"Our servers are currently overloaded. Please try again later.\"}}}\n\n"

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(sse)),
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{ChannelType: constant.ChannelTypeCodex},
		RelayMode:   relayconstant.RelayModeImagesGenerations,
	}

	usage, apiErr := (&Adaptor{}).DoResponse(c, resp, info)
	require.Nil(t, usage)
	require.NotNil(t, apiErr)
	assert.Contains(t, apiErr.Error(), "overloaded")
}

func TestGetEndpointTypesCodexIncludesImageGeneration(t *testing.T) {
	types := common.GetEndpointTypesByChannelType(constant.ChannelTypeCodex, "gpt-5.6-sol")
	assert.Contains(t, types, constant.EndpointTypeImageGeneration)
	assert.Contains(t, types, constant.EndpointTypeOpenAIResponse)
}

func TestIsImageGenerationModelIncludesGptImage2(t *testing.T) {
	assert.True(t, common.IsImageGenerationModel("gpt-image-2"))
	assert.True(t, common.IsImageGenerationModel("gpt-image-1"))
}
