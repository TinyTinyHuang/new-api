package model

import (
	"fmt"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupPromptAuditTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	previousDB, previousLogDB := DB, LOG_DB
	previousRedisEnabled := common.RedisEnabled
	previousMainDatabaseType, previousLogDatabaseType := common.MainDatabaseType(), common.LogDatabaseType()
	common.RedisEnabled = false
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	DB, LOG_DB = db, db
	require.NoError(t, db.AutoMigrate(&PromptAudit{}))

	t.Cleanup(func() {
		DB, LOG_DB = previousDB, previousLogDB
		common.RedisEnabled = previousRedisEnabled
		common.SetDatabaseTypes(previousMainDatabaseType, previousLogDatabaseType)
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func TestGetPromptAuditFilterOptionsReturnsDistinctUsernamesAndTokens(t *testing.T) {
	db := setupPromptAuditTestDB(t)
	require.NoError(t, db.Create([]*PromptAudit{
		{RequestId: "req-1", Username: "alice", TokenName: "ops-key", LastUserText: "hello"},
		{RequestId: "req-2", Username: "bob", TokenName: "ops-key", LastUserText: "world"},
		{RequestId: "req-3", Username: "alice", TokenName: "dev-key", LastUserText: "again"},
		{RequestId: "req-4", Username: "", TokenName: "", LastUserText: "ignored"},
	}).Error)

	options, err := GetPromptAuditFilterOptions()
	require.NoError(t, err)
	assert.Equal(t, []string{"alice", "bob"}, options.Usernames)
	assert.Equal(t, []string{"dev-key", "ops-key"}, options.TokenNames)
}

func TestGetPromptAuditsFiltersByUsernameAndTokenName(t *testing.T) {
	db := setupPromptAuditTestDB(t)
	require.NoError(t, db.Create([]*PromptAudit{
		{RequestId: "req-1", Username: "alice", TokenName: "ops-key", LastUserText: "one", CreatedAt: 3},
		{RequestId: "req-2", Username: "alice", TokenName: "dev-key", LastUserText: "two", CreatedAt: 2},
		{RequestId: "req-3", Username: "bob", TokenName: "ops-key", LastUserText: "three", CreatedAt: 1},
	}).Error)

	audits, total, err := GetPromptAudits(PromptAuditQuery{
		Username:  "alice",
		TokenName: "ops-key",
		Num:       20,
	})
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, audits, 1)
	assert.Equal(t, "req-1", audits[0].RequestId)
}
