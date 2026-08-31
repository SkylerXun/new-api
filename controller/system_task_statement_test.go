package controller

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
)

func TestStatementMonthlyCloseSchedulesAtShanghaiMonthBoundary(t *testing.T) {
	location := time.FixedZone("Asia/Shanghai", 8*60*60)
	handler := statementMonthlyCloseHandler{}
	now := time.Date(2026, time.September, 1, 0, 0, 15, 0, location).Unix()

	previousMonthRun := &model.SystemTask{
		Status:    model.SystemTaskStatusSucceeded,
		UpdatedAt: time.Date(2026, time.August, 31, 23, 59, 0, 0, location).Unix(),
	}
	assert.True(t, handler.ShouldSchedule(now, previousMonthRun))

	currentMonthRun := &model.SystemTask{
		Status:    model.SystemTaskStatusSucceeded,
		UpdatedAt: time.Date(2026, time.September, 1, 0, 0, 10, 0, location).Unix(),
	}
	assert.False(t, handler.ShouldSchedule(now, currentMonthRun))

	failedRun := &model.SystemTask{
		Status:    model.SystemTaskStatusFailed,
		UpdatedAt: now - int64((6 * time.Minute).Seconds()),
	}
	assert.True(t, handler.ShouldSchedule(now, failedRun))
}
