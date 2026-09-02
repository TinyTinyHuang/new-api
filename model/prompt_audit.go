package model

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PromptAudit stores the last user message of a relay request for internal review.
// CUSTOM: local employee prompt capture; keep this file when merging upstream.
type PromptAudit struct {
	Id           int    `json:"id" gorm:"primaryKey"`
	RequestId    string `json:"request_id" gorm:"type:varchar(64);uniqueIndex;not null;default:''"`
	UserId       int    `json:"user_id" gorm:"index;index:idx_prompt_audits_user_created,priority:1"`
	CreatedAt    int64  `json:"created_at" gorm:"bigint;index;index:idx_prompt_audits_user_created,priority:2"`
	Username     string `json:"username" gorm:"index;default:''"`
	TokenName    string `json:"token_name" gorm:"index;default:''"`
	TokenId      int    `json:"token_id" gorm:"index;default:0"`
	ModelName    string `json:"model_name" gorm:"index;default:''"`
	Group        string `json:"group" gorm:"index;default:''"`
	Ip           string `json:"ip" gorm:"default:''"`
	RelayFormat  string `json:"relay_format" gorm:"type:varchar(64);default:''"`
	LastUserText string `json:"last_user_text" gorm:"type:text"`
	MessageCount int    `json:"message_count" gorm:"default:0"`
	Blocked      bool   `json:"blocked" gorm:"index;default:false"`
	BlockedWords string `json:"blocked_words" gorm:"type:varchar(1024);default:''"`
}

func promptAuditDB() *gorm.DB {
	if LOG_DB != nil && !common.UsingLogDatabase(common.DatabaseTypeClickHouse) {
		return LOG_DB
	}
	return DB
}

func migratePromptAudit() error {
	db := promptAuditDB()
	if db == nil {
		return nil
	}
	return db.AutoMigrate(&PromptAudit{})
}

func CreatePromptAudit(audit *PromptAudit) error {
	if audit == nil {
		return nil
	}
	db := promptAuditDB()
	if db == nil {
		return errors.New("prompt audit database is not initialized")
	}
	if audit.RequestId == "" {
		audit.RequestId = common.NewRequestId()
	}
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "request_id"}},
		DoNothing: true,
	}).Create(audit).Error
}

func GetPromptAuditByRequestId(requestId string) (*PromptAudit, error) {
	if requestId == "" {
		return nil, gorm.ErrRecordNotFound
	}
	db := promptAuditDB()
	if db == nil {
		return nil, errors.New("prompt audit database is not initialized")
	}
	var audit PromptAudit
	err := db.Where("request_id = ?", requestId).First(&audit).Error
	if err != nil {
		return nil, err
	}
	return &audit, nil
}

type PromptAuditQuery struct {
	Username       string
	TokenName      string
	ModelName      string
	Keyword        string
	RequestId      string
	Blocked        *bool
	StartTimestamp int64
	EndTimestamp   int64
	StartIdx       int
	Num            int
}

func GetPromptAudits(query PromptAuditQuery) (audits []*PromptAudit, total int64, err error) {
	db := promptAuditDB()
	if db == nil {
		return nil, 0, errors.New("prompt audit database is not initialized")
	}
	tx := db.Model(&PromptAudit{})
	if query.Username != "" {
		tx = tx.Where("username = ?", query.Username)
	}
	if query.TokenName != "" {
		tx = tx.Where("token_name = ?", query.TokenName)
	}
	if query.ModelName != "" {
		tx = tx.Where("model_name = ?", query.ModelName)
	}
	if query.RequestId != "" {
		tx = tx.Where("request_id = ?", query.RequestId)
	}
	if query.Blocked != nil {
		tx = tx.Where("blocked = ?", *query.Blocked)
	}
	if query.StartTimestamp != 0 {
		tx = tx.Where("created_at >= ?", query.StartTimestamp)
	}
	if query.EndTimestamp != 0 {
		tx = tx.Where("created_at <= ?", query.EndTimestamp)
	}
	if tx, err = applyPromptAuditKeywordFilter(tx, query.Keyword); err != nil {
		return nil, 0, err
	}
	if err = tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if query.Num <= 0 {
		query.Num = common.ItemsPerPage
	}
	err = tx.Order("created_at desc, id desc").Offset(query.StartIdx).Limit(query.Num).Find(&audits).Error
	if err != nil {
		return nil, 0, err
	}
	if audits == nil {
		audits = []*PromptAudit{}
	}
	return audits, total, nil
}

func applyPromptAuditKeywordFilter(tx *gorm.DB, keyword string) (*gorm.DB, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return tx, nil
	}
	hasWildcard := strings.Contains(keyword, "%")
	pattern, err := sanitizeLikePattern(keyword)
	if err != nil {
		return tx, err
	}
	if !hasWildcard {
		pattern = "%" + pattern + "%"
	}
	return tx.Where("last_user_text LIKE ? ESCAPE '!'", pattern), nil
}
