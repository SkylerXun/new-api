package service

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relaykit/dto"

	"github.com/bytedance/gopkg/util/gopool"
)

const (
	subscriptionResetTickInterval = 1 * time.Minute
	subscriptionResetBatchSize    = 300
	subscriptionCleanupInterval   = 30 * time.Minute
)

var (
	subscriptionResetOnce    sync.Once
	subscriptionResetRunning atomic.Bool
	subscriptionCleanupLast  atomic.Int64
)

func StartSubscriptionQuotaResetTask() {
	subscriptionResetOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		gopool.Go(func() {
			logger.LogInfo(context.Background(), fmt.Sprintf("subscription quota reset task started: tick=%s", subscriptionResetTickInterval))
			ticker := time.NewTicker(subscriptionResetTickInterval)
			defer ticker.Stop()

			runSubscriptionQuotaResetOnce()
			for range ticker.C {
				runSubscriptionQuotaResetOnce()
			}
		})
	})
}

func runSubscriptionQuotaResetOnce() {
	if !subscriptionResetRunning.CompareAndSwap(false, true) {
		return
	}
	defer subscriptionResetRunning.Store(false)

	ctx := context.Background()
	totalReset := 0
	totalExpired := 0
	for {
		n, err := model.ExpireDueSubscriptions(subscriptionResetBatchSize)
		if err != nil {
			logger.LogWarn(ctx, fmt.Sprintf("subscription expire task failed: %v", err))
			return
		}
		if n == 0 {
			break
		}
		totalExpired += n
		if n < subscriptionResetBatchSize {
			break
		}
	}
	sendSubscriptionExpiryReminders(ctx)
	for {
		n, err := model.ResetDueSubscriptions(subscriptionResetBatchSize)
		if err != nil {
			logger.LogWarn(ctx, fmt.Sprintf("subscription quota reset task failed: %v", err))
			return
		}
		if n == 0 {
			break
		}
		totalReset += n
		if n < subscriptionResetBatchSize {
			break
		}
	}
	lastCleanup := time.Unix(subscriptionCleanupLast.Load(), 0)
	if time.Since(lastCleanup) >= subscriptionCleanupInterval {
		if _, err := model.CleanupSubscriptionPreConsumeRecords(7 * 24 * 3600); err == nil {
			subscriptionCleanupLast.Store(time.Now().Unix())
		}
	}
	if common.DebugEnabled && (totalReset > 0 || totalExpired > 0) {
		logger.LogDebug(ctx, "subscription maintenance: reset_count=%d, expired_count=%d", totalReset, totalExpired)
	}
}

func sendSubscriptionExpiryReminders(ctx context.Context) {
	common.OptionMapRWMutex.RLock()
	enabled := common.OptionMap["subscription_expiry_notify_enabled"] != "false"
	days := 1
	if parsed, err := strconv.Atoi(common.OptionMap["subscription_expiry_notify_days"]); err == nil && parsed >= 1 && parsed <= 30 {
		days = parsed
	}
	common.OptionMapRWMutex.RUnlock()
	if !enabled {
		return
	}
	now := common.GetTimestamp()
	windowEnd := now + int64(days*24*60*60)
	windowStart := windowEnd - int64(24*60*60)
	var subs []model.UserSubscription
	if err := model.DB.Where("status = ? AND end_time > ? AND end_time <= ? AND expiry_reminder_sent_at = 0", "active", windowStart, windowEnd).Limit(subscriptionResetBatchSize).Find(&subs).Error; err != nil {
		logger.LogWarn(ctx, fmt.Sprintf("subscription expiry reminder query failed: %v", err))
		return
	}
	for i := range subs {
		sub := &subs[i]
		claimed := model.DB.Model(&model.UserSubscription{}).Where("id = ? AND expiry_reminder_sent_at = 0", sub.Id).Update("expiry_reminder_sent_at", now)
		if claimed.Error != nil || claimed.RowsAffected == 0 {
			continue
		}
		user, err := model.GetUserById(sub.UserId, false)
		if err != nil || user.Email == "" {
			continue
		}
		plan, err := model.GetSubscriptionPlanById(sub.PlanId)
		if err != nil {
			continue
		}
		daysRemaining := (sub.EndTime - now) / int64(24*60*60)
		if err := NotifyUser(user.Id, user.Email, user.GetSetting(), dto.NewNotify(dto.NotifyTypeSubscriptionExpiry, "Subscription expiry reminder", fmt.Sprintf("Your %s subscription expires in approximately %d day(s).", plan.Title, daysRemaining+1), nil)); err != nil {
			logger.LogWarn(ctx, fmt.Sprintf("subscription expiry reminder send failed: user=%d err=%v", user.Id, err))
		}
	}
}
