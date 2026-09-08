package controller

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestDisabledLinkedSubaccountErrorIncludesMainUsername(t *testing.T) {
	previousDB := model.DB
	previousLogDB := model.LOG_DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	model.DB, model.LOG_DB = db, db
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.RiskExemption{}, &model.RiskAccountLink{}))
	t.Cleanup(func() { model.DB, model.LOG_DB = previousDB, previousLogDB })

	now := time.Now().Unix()
	main := &model.User{Username: "controller-main", Password: "unused", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, AffCode: "controller-main-aff", CreatedAt: now - 10}
	sub := &model.User{Username: "controller-sub", Password: "unused", Role: common.RoleCommonUser, Status: common.UserStatusDisabled, AffCode: "controller-sub-aff", CreatedAt: now}
	require.NoError(t, db.Create(main).Error)
	require.NoError(t, db.Create(sub).Error)
	require.NoError(t, db.Create(&model.RiskAccountLink{UserID: main.Id, MainUserID: main.Id, ClusterID: "cluster-test", Relation: "main", PolicyVersion: model.RiskAccountPolicyVersion, DecisionAt: now, CreatedAt: now, UpdatedAt: now}).Error)
	require.NoError(t, db.Create(&model.RiskAccountLink{UserID: sub.Id, MainUserID: main.Id, ClusterID: "cluster-test", Relation: "subaccount", PolicyVersion: model.RiskAccountPolicyVersion, DecisionAt: now, CreatedAt: now, UpdatedAt: now}).Error)

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	writeAccountBannedError(ctx, sub)

	assert.Contains(t, recorder.Body.String(), main.Username)
	assert.Contains(t, recorder.Body.String(), "linked_subaccount_banned")
}
