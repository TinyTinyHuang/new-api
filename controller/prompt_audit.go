package controller

import (
	"errors"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetPromptAudits(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	var blocked *bool
	if raw := strings.TrimSpace(c.Query("blocked")); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			common.ApiErrorMsg(c, "invalid blocked filter")
			return
		}
		blocked = &value
	}
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	audits, total, err := model.GetPromptAudits(model.PromptAuditQuery{
		Username:       c.Query("username"),
		TokenName:      c.Query("token_name"),
		ModelName:      c.Query("model_name"),
		Keyword:        c.Query("keyword"),
		RequestId:      c.Query("request_id"),
		Blocked:        blocked,
		StartTimestamp: startTimestamp,
		EndTimestamp:   endTimestamp,
		StartIdx:       pageInfo.GetStartIdx(),
		Num:            pageInfo.GetPageSize(),
	})
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(audits)
	common.ApiSuccess(c, pageInfo)
}

func GetPromptAuditByRequestId(c *gin.Context) {
	requestId := strings.TrimSpace(c.Query("request_id"))
	if requestId == "" {
		common.ApiErrorMsg(c, "request_id is required")
		return
	}
	audit, err := model.GetPromptAuditByRequestId(requestId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiSuccess(c, nil)
			return
		}
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, audit)
}
