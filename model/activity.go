package model

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

const (
	ActivityKeyNewUserRedeemBonus = "new_user_redeem_bonus"

	ActivityGrantSourceRedeem            = "redeem"
	ActivityGrantSourceAllUsersGrant     = "all_users_grant"
	ActivityGrantSourceCampaignClaim     = "campaign_claim"
	ActivityGrantSourceCampaignImmediate = "campaign_immediate"

	ActivityGrantSourceRefClaim     = "claim"
	ActivityGrantSourceRefImmediate = "immediate"
	ActivityGrantSourceRefAllUsers  = "all_users"

	ActivityCampaignTypeClaimable = "claimable"
	ActivityCampaignTypeImmediate = "immediate"

	ActivityCampaignAudienceAll      = "all"
	ActivityCampaignAudienceSelected = "selected"

	ActivityCampaignStatusActive    = "active"
	ActivityCampaignStatusQueued    = "queued"
	ActivityCampaignStatusRunning   = "running"
	ActivityCampaignStatusCompleted = "completed"
	ActivityCampaignStatusFailed    = "failed"
	ActivityCampaignStatusClosed    = "closed"
)

var (
	ErrActivityCampaignNotFound      = errors.New("activity campaign not found")
	ErrActivityCampaignUnavailable   = errors.New("activity campaign is unavailable")
	ErrActivityCampaignNotClaimable  = errors.New("activity campaign is not claimable")
	ErrActivityCampaignTaskActive    = errors.New("activity campaign task is still active")
	ErrActivityCampaignTaskMismatch  = errors.New("activity campaign task mismatch")
	ErrActivityCampaignInvalidStatus = errors.New("activity campaign has an invalid status")
)

// ActivityGrant is the durable activity-credit ledger. Its unique key keeps
// each source event idempotent without preventing a later, distinct source
// from granting the same activity to the same user.
type ActivityGrant struct {
	Id          int64  `json:"id"`
	ActivityKey string `json:"activity_key" gorm:"type:varchar(128);not null;uniqueIndex:idx_activity_grant_source,priority:1;index"`
	UserId      int    `json:"user_id" gorm:"not null;uniqueIndex:idx_activity_grant_source,priority:2;index"`
	SourceType  string `json:"source_type" gorm:"type:varchar(64);not null"`
	SourceRef   string `json:"source_ref" gorm:"type:varchar(191);not null;uniqueIndex:idx_activity_grant_source,priority:3;index"`
	Quota       int    `json:"quota" gorm:"type:int;not null"`
	CreatedAt   int64  `json:"created_at" gorm:"bigint;not null;index"`
}

// ActivityCampaign is the operator-managed activity definition. Claimable
// campaigns are redeemed by an authenticated user; immediate campaigns are
// fulfilled by a background all-user grant task.
type ActivityCampaign struct {
	Id                 int64  `json:"id"`
	ActivityKey        string `json:"activity_key" gorm:"type:varchar(128);not null;uniqueIndex:idx_activity_campaign_key"`
	Type               string `json:"type" gorm:"type:varchar(32);not null;index"`
	AudienceType       string `json:"audience_type" gorm:"type:varchar(32);index"`
	Status             string `json:"status" gorm:"type:varchar(32);not null;index"`
	Title              string `json:"title" gorm:"type:varchar(128);not null"`
	Description        string `json:"description" gorm:"type:text"`
	Reason             string `json:"reason" gorm:"type:varchar(255)"`
	AmountUSD          string `json:"amount_usd" gorm:"type:varchar(64);not null;default:''"`
	Quota              int    `json:"quota" gorm:"type:int;not null"`
	StartsAt           int64  `json:"starts_at" gorm:"bigint;not null;index"`
	EndsAt             int64  `json:"ends_at" gorm:"bigint;not null;index"`
	RecipientMaxUserID int    `json:"recipient_max_user_id" gorm:"type:int;not null;default:0"`
	RecipientCount     int64  `json:"recipient_count" gorm:"bigint;not null;default:0"`
	TaskID             string `json:"task_id,omitempty" gorm:"type:varchar(64);index"`
	FailureReason      string `json:"failure_reason,omitempty" gorm:"type:text"`
	CreatedBy          int    `json:"created_by" gorm:"not null;index"`
	ClosedBy           int    `json:"closed_by,omitempty" gorm:"not null;default:0"`
	ClosedAt           int64  `json:"closed_at,omitempty" gorm:"bigint;not null;default:0"`
	CreatedAt          int64  `json:"created_at" gorm:"bigint;not null;index"`
	UpdatedAt          int64  `json:"updated_at" gorm:"bigint;not null;index"`
	GrantedCount       int64  `json:"granted_count" gorm:"-"`
}

type ActivityCampaignGrantDetail struct {
	Id          int64  `json:"id"`
	UserId      int    `json:"user_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Quota       int    `json:"quota"`
	GrantedAt   int64  `json:"granted_at"`
}

// ActivityCampaignRecipient freezes the explicit audience of a selected-user
// campaign. A user can occur at most once per campaign.
type ActivityCampaignRecipient struct {
	Id         int64 `json:"id"`
	CampaignID int64 `json:"campaign_id" gorm:"not null;uniqueIndex:idx_activity_campaign_recipient,priority:1;index"`
	UserID     int   `json:"user_id" gorm:"not null;uniqueIndex:idx_activity_campaign_recipient,priority:2;index"`
	CreatedAt  int64 `json:"created_at" gorm:"bigint;not null;index"`
}

func (ActivityCampaignRecipient) TableName() string {
	return "activity_campaign_recipients"
}

func (ActivityCampaign) TableName() string {
	return "activity_campaigns"
}

func (grant *ActivityGrant) BeforeCreate(_ *gorm.DB) error {
	if grant.CreatedAt == 0 {
		grant.CreatedAt = common.GetTimestamp()
	}
	return nil
}

func (campaign *ActivityCampaign) BeforeCreate(_ *gorm.DB) error {
	now := common.GetTimestamp()
	if campaign.CreatedAt == 0 {
		campaign.CreatedAt = now
	}
	if campaign.UpdatedAt == 0 {
		campaign.UpdatedAt = now
	}
	return nil
}

func GetActivityGrantForUser(userId int, activityKey string) (*ActivityGrant, error) {
	if userId <= 0 || strings.TrimSpace(activityKey) == "" {
		return nil, errors.New("invalid activity grant lookup")
	}
	var grant ActivityGrant
	err := DB.Where("activity_key = ? AND user_id = ?", strings.TrimSpace(activityKey), userId).
		Order("created_at desc, id desc").
		First(&grant).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &grant, nil
}

func GetActivityGrantForUserSource(userId int, activityKey string, sourceRef string) (*ActivityGrant, error) {
	if userId <= 0 || strings.TrimSpace(activityKey) == "" || strings.TrimSpace(sourceRef) == "" {
		return nil, errors.New("invalid activity grant lookup")
	}
	return getActivityGrantForUserSource(DB, userId, strings.TrimSpace(activityKey), strings.TrimSpace(sourceRef))
}

func getActivityGrantForUserSource(tx *gorm.DB, userId int, activityKey string, sourceRef string) (*ActivityGrant, error) {
	var grant ActivityGrant
	err := tx.Where("activity_key = ? AND user_id = ? AND source_ref = ?", activityKey, userId, sourceRef).
		First(&grant).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &grant, nil
}

// GrantActivityQuotaTx atomically records one source credit and increases the
// user's wallet only when that exact source has not already granted it.
// Callers must commit or roll back tx with their source event before updating
// the quota cache.
func GrantActivityQuotaTx(tx *gorm.DB, userId int, activityKey string, sourceType string, sourceRef string, quota int) (bool, error) {
	activityKey = strings.TrimSpace(activityKey)
	sourceType = strings.TrimSpace(sourceType)
	sourceRef = strings.TrimSpace(sourceRef)
	if tx == nil || userId <= 0 || activityKey == "" || sourceType == "" || sourceRef == "" || quota <= 0 {
		return false, errors.New("invalid activity grant")
	}
	if utf8.RuneCountInString(activityKey) > 128 || utf8.RuneCountInString(sourceType) > 64 || utf8.RuneCountInString(sourceRef) > 191 {
		return false, errors.New("activity grant metadata is too long")
	}

	// Serialize grants for one user before checking the ledger. This avoids
	// relying on dialect-specific conflict RowsAffected behavior, notably MySQL
	// with CLIENT_FOUND_ROWS enabled.
	var user User
	if err := lockForUpdate(tx).Select("id").Where("id = ?", userId).First(&user).Error; err != nil {
		return false, err
	}
	existing, lookupErr := getActivityGrantForUserSource(tx, userId, activityKey, sourceRef)
	if lookupErr != nil {
		return false, lookupErr
	}
	if existing != nil {
		return false, nil
	}

	grant := ActivityGrant{
		ActivityKey: activityKey,
		UserId:      userId,
		SourceType:  sourceType,
		SourceRef:   sourceRef,
		Quota:       quota,
	}
	result := tx.Create(&grant)
	if result.Error != nil {
		return false, result.Error
	}

	result = tx.Model(&User{}).Where("id = ? AND quota <= ?", userId, common.MaxQuota-quota).
		Update("quota", gorm.Expr("quota + ?", quota))
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected != 1 {
		return false, gorm.ErrRecordNotFound
	}
	return true, nil
}

// GrantActivityQuota is the transaction-owning form for standalone activity
// credits. It synchronizes the Redis-backed quota cache after a successful
// commit; use GrantActivityQuotaTx when the credit must join another event's
// transaction such as redemption.
func GrantActivityQuota(ctx context.Context, userId int, activityKey string, sourceType string, sourceRef string, quota int) (bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	granted := false
	err := DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		granted, err = GrantActivityQuotaTx(tx, userId, activityKey, sourceType, sourceRef, quota)
		return err
	})
	if err != nil || !granted {
		return granted, err
	}
	syncCreditUserQuotaCache(userId, quota, "activity grant")
	return true, nil
}

// GrantActivityQuotaBatch grants one activity credit to every supplied user.
// Each user gets an independent transaction so a deleted user or a wallet at
// the quota ceiling is skipped without rolling back credits already committed
// for the rest of the batch. The ledger makes retries safe.
func GrantActivityQuotaBatch(ctx context.Context, userIds []int, activityKey string, sourceType string, sourceRef string, quota int) (granted int, skipped int, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(userIds) == 0 {
		return 0, 0, nil
	}

	for _, userId := range userIds {
		if err := ctx.Err(); err != nil {
			return granted, skipped, err
		}
		didGrant, grantErr := GrantActivityQuota(ctx, userId, activityKey, sourceType, sourceRef, quota)
		if errors.Is(grantErr, gorm.ErrRecordNotFound) {
			skipped++
			continue
		}
		if grantErr != nil {
			return granted, skipped, grantErr
		}
		if didGrant {
			granted++
		} else {
			skipped++
		}
	}
	return granted, skipped, nil
}

func CreateActivityCampaign(ctx context.Context, campaign *ActivityCampaign) error {
	return CreateActivityCampaignWithRecipients(ctx, campaign, nil)
}

// CreateActivityCampaignWithRecipients creates the campaign and, for a
// selected audience, its frozen recipient rows atomically.
func CreateActivityCampaignWithRecipients(ctx context.Context, campaign *ActivityCampaign, userIDs []int) error {
	if campaign == nil {
		return errors.New("activity campaign is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := normalizeNewActivityCampaign(campaign); err != nil {
		return err
	}
	if campaign.AudienceType == "" {
		campaign.AudienceType = ActivityCampaignAudienceAll
	}
	userIDs = uniquePositiveInts(userIDs)
	if campaign.AudienceType == ActivityCampaignAudienceSelected {
		if campaign.Type != ActivityCampaignTypeClaimable || len(userIDs) == 0 {
			return errors.New("selected activity campaigns require at least one recipient and must be claimable")
		}
	} else if campaign.AudienceType != ActivityCampaignAudienceAll || len(userIDs) > 0 {
		return errors.New("activity campaign audience is invalid")
	}

	return DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if campaign.AudienceType == ActivityCampaignAudienceSelected {
			var count int64
			if err := tx.Model(&User{}).Where("id IN ? AND status = ?", userIDs, common.UserStatusEnabled).Count(&count).Error; err != nil {
				return err
			}
			if count != int64(len(userIDs)) {
				return errors.New("selected activity recipients must be enabled users")
			}
			campaign.RecipientMaxUserID = 0
			campaign.RecipientCount = int64(len(userIDs))
		} else {
			maxUserID, recipientCount, err := getActivityCampaignTargetSnapshotTx(tx)
			if err != nil {
				return err
			}
			campaign.RecipientMaxUserID = maxUserID
			campaign.RecipientCount = recipientCount
		}
		if err := tx.Create(campaign).Error; err != nil {
			return err
		}
		if campaign.AudienceType == ActivityCampaignAudienceSelected {
			recipients := make([]ActivityCampaignRecipient, 0, len(userIDs))
			createdAt := common.GetTimestamp()
			for _, userID := range userIDs {
				recipients = append(recipients, ActivityCampaignRecipient{CampaignID: campaign.Id, UserID: userID, CreatedAt: createdAt})
			}
			if err := tx.Create(&recipients).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func uniquePositiveInts(values []int) []int {
	seen := make(map[int]struct{}, len(values))
	result := make([]int, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

// GetActivityCampaignTargetSnapshot freezes the account ID frontier when a
// campaign is published. Unlike immediate-grant execution, the snapshot does
// not filter by current status: an existing account may be re-enabled before
// it claims, while accounts created after the frontier can never participate.
func GetActivityCampaignTargetSnapshot(ctx context.Context) (maxUserID int, total int64, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	query := DB.WithContext(ctx).Model(&User{})
	if err = query.Count(&total).Error; err != nil {
		return 0, 0, err
	}
	if total == 0 {
		return 0, 0, nil
	}

	var lastUser User
	if err = query.Select("id").Order("id desc").First(&lastUser).Error; err != nil {
		return 0, 0, err
	}
	return lastUser.Id, total, nil
}

func getActivityCampaignTargetSnapshotTx(tx *gorm.DB) (maxUserID int, total int64, err error) {
	query := tx.Model(&User{})
	if err = query.Count(&total).Error; err != nil || total == 0 {
		return 0, total, err
	}
	var lastUser User
	if err = query.Select("id").Order("id desc").First(&lastUser).Error; err != nil {
		return 0, 0, err
	}
	return lastUser.Id, total, nil
}

func GetActivityCampaignByKey(ctx context.Context, activityKey string) (*ActivityCampaign, error) {
	activityKey = strings.TrimSpace(activityKey)
	if activityKey == "" {
		return nil, errors.New("activity campaign key is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var campaign ActivityCampaign
	err := DB.WithContext(ctx).Where("activity_key = ?", activityKey).First(&campaign).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &campaign, nil
}

func GetActivityCampaignByID(ctx context.Context, id int64) (*ActivityCampaign, error) {
	if id <= 0 {
		return nil, errors.New("activity campaign id is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var campaign ActivityCampaign
	err := DB.WithContext(ctx).Where("id = ?", id).First(&campaign).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &campaign, nil
}

func ListActivityCampaigns(ctx context.Context, limit int) ([]*ActivityCampaign, error) {
	campaigns, _, err := ListActivityCampaignsPage(ctx, 0, limit)
	return campaigns, err
}

// ListActivityCampaignsPage returns campaigns in reverse creation order. The
// cursor is the last campaign ID from the preceding page, keeping pagination
// stable without exposing a database-specific offset query.
func ListActivityCampaignsPage(ctx context.Context, cursor int64, limit int) ([]*ActivityCampaign, int64, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if cursor < 0 {
		return nil, 0, errors.New("activity campaign cursor is invalid")
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	var campaigns []*ActivityCampaign
	query := DB.WithContext(ctx).Order("id desc")
	if cursor > 0 {
		query = query.Where("id < ?", cursor)
	}
	if err := query.Limit(limit + 1).Find(&campaigns).Error; err != nil {
		return nil, 0, err
	}
	nextCursor := int64(0)
	if len(campaigns) > limit {
		nextCursor = campaigns[limit-1].Id
		campaigns = campaigns[:limit]
	}
	if err := populateActivityCampaignGrantedCounts(ctx, campaigns); err != nil {
		return nil, 0, err
	}
	return campaigns, nextCursor, nil
}

func populateActivityCampaignGrantedCounts(ctx context.Context, campaigns []*ActivityCampaign) error {
	if len(campaigns) == 0 {
		return nil
	}
	type grantCount struct {
		ActivityKey string
		SourceType  string
		SourceRef   string
		Count       int64
	}
	keys := make([]string, 0, len(campaigns))
	for _, campaign := range campaigns {
		keys = append(keys, campaign.ActivityKey)
	}
	var counts []grantCount
	err := DB.WithContext(ctx).Model(&ActivityGrant{}).
		Select("activity_key, source_type, source_ref, COUNT(*) AS count").
		Where("activity_key IN ?", keys).
		Where("(source_type = ? AND source_ref = ?) OR (source_type = ? AND source_ref = ?)", ActivityGrantSourceCampaignClaim, ActivityGrantSourceRefClaim, ActivityGrantSourceCampaignImmediate, ActivityGrantSourceRefImmediate).
		Group("activity_key, source_type, source_ref").
		Scan(&counts).Error
	if err != nil {
		return err
	}
	countBySource := make(map[string]int64, len(counts))
	for _, count := range counts {
		countBySource[count.ActivityKey+"\x00"+count.SourceType+"\x00"+count.SourceRef] = count.Count
	}
	for _, campaign := range campaigns {
		sourceType, sourceRef := activityCampaignGrantSource(campaign.Type)
		campaign.GrantedCount = countBySource[campaign.ActivityKey+"\x00"+sourceType+"\x00"+sourceRef]
	}
	return nil
}

func ListActivityCampaignGrantDetailsPage(ctx context.Context, campaign *ActivityCampaign, cursor int64, limit int) ([]ActivityCampaignGrantDetail, int64, error) {
	if campaign == nil {
		return nil, 0, errors.New("activity campaign is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if cursor < 0 {
		return nil, 0, errors.New("activity campaign grant cursor is invalid")
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	sourceType, sourceRef := activityCampaignGrantSource(campaign.Type)
	if sourceType == "" {
		return nil, 0, errors.New("activity campaign type is invalid")
	}

	details := make([]ActivityCampaignGrantDetail, 0, limit+1)
	query := DB.WithContext(ctx).Table("activity_grants AS grants").
		Select("grants.id, grants.user_id, COALESCE(users.username, '') AS username, COALESCE(users.display_name, '') AS display_name, grants.quota, grants.created_at AS granted_at").
		Joins("LEFT JOIN users AS users ON users.id = grants.user_id AND users.deleted_at IS NULL").
		Where("grants.activity_key = ? AND grants.source_type = ? AND grants.source_ref = ?", campaign.ActivityKey, sourceType, sourceRef).
		Order("grants.id DESC")
	if cursor > 0 {
		query = query.Where("grants.id < ?", cursor)
	}
	if err := query.Limit(limit + 1).Scan(&details).Error; err != nil {
		return nil, 0, err
	}
	if len(details) <= limit {
		return details, 0, nil
	}
	nextCursor := details[limit-1].Id
	return details[:limit], nextCursor, nil
}

func activityCampaignGrantSource(campaignType string) (string, string) {
	if campaignType == ActivityCampaignTypeClaimable {
		return ActivityGrantSourceCampaignClaim, ActivityGrantSourceRefClaim
	}
	if campaignType == ActivityCampaignTypeImmediate {
		return ActivityGrantSourceCampaignImmediate, ActivityGrantSourceRefImmediate
	}
	return "", ""
}

func CloseActivityCampaign(ctx context.Context, activityKey string, closedBy int) (*ActivityCampaign, error) {
	activityKey = strings.TrimSpace(activityKey)
	if activityKey == "" || closedBy <= 0 {
		return nil, errors.New("invalid activity campaign close request")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	var closed ActivityCampaign
	err := DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var campaign ActivityCampaign
		if err := lockForUpdate(tx).Where("activity_key = ?", activityKey).First(&campaign).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrActivityCampaignNotFound
			}
			return err
		}
		if campaign.Status == ActivityCampaignStatusClosed {
			closed = campaign
			return nil
		}
		if campaign.Type == ActivityCampaignTypeImmediate && (campaign.Status == ActivityCampaignStatusQueued || campaign.Status == ActivityCampaignStatusRunning) {
			return ErrActivityCampaignTaskActive
		}
		if campaign.Status != ActivityCampaignStatusActive {
			return fmt.Errorf("%w: %s", ErrActivityCampaignInvalidStatus, campaign.Status)
		}

		now := common.GetTimestamp()
		result := tx.Model(&ActivityCampaign{}).Where("id = ? AND status = ?", campaign.Id, ActivityCampaignStatusActive).
			Updates(map[string]any{
				"status":     ActivityCampaignStatusClosed,
				"closed_by":  closedBy,
				"closed_at":  now,
				"updated_at": now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return ErrActivityCampaignInvalidStatus
		}
		campaign.Status = ActivityCampaignStatusClosed
		campaign.ClosedBy = closedBy
		campaign.ClosedAt = now
		campaign.UpdatedAt = now
		closed = campaign
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &closed, nil
}

// AttachActivityCampaignTask associates a newly created immediate task before
// it is picked up by the runner. The runner repeats this association while it
// transitions the campaign to running, making a cross-node pickup safe too.
func AttachActivityCampaignTask(ctx context.Context, campaignID int64, taskID string) error {
	return transitionActivityCampaignTask(ctx, campaignID, taskID,
		[]string{ActivityCampaignStatusQueued, ActivityCampaignStatusRunning}, "", "")
}

func MarkActivityCampaignTaskRunning(ctx context.Context, campaignID int64, taskID string) error {
	return transitionActivityCampaignTask(ctx, campaignID, taskID,
		[]string{ActivityCampaignStatusQueued, ActivityCampaignStatusRunning}, ActivityCampaignStatusRunning, "")
}

func CompleteActivityCampaignTask(ctx context.Context, campaignID int64, taskID string) error {
	return transitionActivityCampaignTask(ctx, campaignID, taskID,
		[]string{ActivityCampaignStatusQueued, ActivityCampaignStatusRunning, ActivityCampaignStatusCompleted}, ActivityCampaignStatusCompleted, "")
}

func FailActivityCampaignTask(ctx context.Context, campaignID int64, taskID string, failureReason string) error {
	return transitionActivityCampaignTask(ctx, campaignID, taskID,
		[]string{ActivityCampaignStatusQueued, ActivityCampaignStatusRunning, ActivityCampaignStatusFailed}, ActivityCampaignStatusFailed, failureReason)
}

// FailActivityCampaignEnqueue records an enqueue failure before an immediate
// campaign has a task ID. This keeps failed creation attempts visible to the
// operator without treating an unrelated active task as this campaign's task.
func FailActivityCampaignEnqueue(ctx context.Context, campaignID int64, failureReason string) error {
	if campaignID <= 0 {
		return errors.New("invalid activity campaign id")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	failureReason = strings.TrimSpace(failureReason)
	if utf8.RuneCountInString(failureReason) > 2000 {
		failureReason = string([]rune(failureReason)[:2000])
	}
	now := common.GetTimestamp()
	result := DB.WithContext(ctx).Model(&ActivityCampaign{}).
		Where("id = ? AND type = ? AND status = ?", campaignID, ActivityCampaignTypeImmediate, ActivityCampaignStatusQueued).
		Updates(map[string]any{
			"status":         ActivityCampaignStatusFailed,
			"failure_reason": failureReason,
			"updated_at":     now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 0 {
		return nil
	}
	var campaign ActivityCampaign
	if err := DB.WithContext(ctx).Where("id = ?", campaignID).First(&campaign).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrActivityCampaignNotFound
		}
		return err
	}
	if campaign.Type == ActivityCampaignTypeImmediate && campaign.Status == ActivityCampaignStatusFailed {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrActivityCampaignInvalidStatus, campaign.Status)
}

func transitionActivityCampaignTask(ctx context.Context, campaignID int64, taskID string, allowedStatuses []string, targetStatus string, failureReason string) error {
	if campaignID <= 0 || strings.TrimSpace(taskID) == "" {
		return errors.New("invalid activity campaign task")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	taskID = strings.TrimSpace(taskID)
	failureReason = strings.TrimSpace(failureReason)
	if utf8.RuneCountInString(failureReason) > 2000 {
		failureReason = string([]rune(failureReason)[:2000])
	}
	now := common.GetTimestamp()
	query := DB.WithContext(ctx).Model(&ActivityCampaign{}).
		Where("id = ? AND type = ? AND status IN ?", campaignID, ActivityCampaignTypeImmediate, allowedStatuses).
		Where("task_id = ? OR task_id = ?", "", taskID)
	updates := map[string]any{
		"task_id":    taskID,
		"updated_at": now,
	}
	if targetStatus != "" {
		updates["status"] = targetStatus
	}
	if targetStatus == ActivityCampaignStatusFailed {
		updates["failure_reason"] = failureReason
	} else if targetStatus == ActivityCampaignStatusQueued || targetStatus == ActivityCampaignStatusRunning || targetStatus == ActivityCampaignStatusCompleted {
		updates["failure_reason"] = ""
	}
	if err := query.Updates(updates).Error; err != nil {
		return err
	}

	var campaign ActivityCampaign
	if err := DB.WithContext(ctx).Where("id = ?", campaignID).First(&campaign).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrActivityCampaignNotFound
		}
		return err
	}
	if campaign.Type != ActivityCampaignTypeImmediate || campaign.TaskID != taskID {
		return ErrActivityCampaignTaskMismatch
	}
	if targetStatus == "" || campaign.Status == targetStatus {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrActivityCampaignInvalidStatus, campaign.Status)
}

// ClaimActivityCampaignQuota verifies campaign availability and performs the
// ledger insert plus quota credit in one transaction. Repeated client requests
// return the existing claim instead of granting twice.
func ClaimActivityCampaignQuota(ctx context.Context, userId int, activityKey string) (*ActivityGrant, bool, error) {
	activityKey = strings.TrimSpace(activityKey)
	if userId <= 0 || activityKey == "" {
		return nil, false, errors.New("invalid activity campaign claim")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	var grant *ActivityGrant
	granted := false
	err := DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var campaign ActivityCampaign
		if err := lockForUpdate(tx).Where("activity_key = ?", activityKey).First(&campaign).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrActivityCampaignNotFound
			}
			return err
		}
		if campaign.Type != ActivityCampaignTypeClaimable {
			return ErrActivityCampaignNotClaimable
		}
		if campaign.Status != ActivityCampaignStatusActive {
			return ErrActivityCampaignUnavailable
		}
		eligible, eligibilityErr := isActivityCampaignUserEligible(tx, &campaign, userId)
		if eligibilityErr != nil {
			return eligibilityErr
		}
		if !eligible {
			return ErrActivityCampaignUnavailable
		}
		now := common.GetTimestamp()
		if now < campaign.StartsAt || campaign.EndsAt <= now {
			return ErrActivityCampaignUnavailable
		}

		var err error
		granted, err = GrantActivityQuotaTx(
			tx,
			userId,
			campaign.ActivityKey,
			ActivityGrantSourceCampaignClaim,
			ActivityGrantSourceRefClaim,
			campaign.Quota,
		)
		if err != nil {
			return err
		}
		grant, err = getActivityGrantForUserSource(tx, userId, campaign.ActivityKey, ActivityGrantSourceRefClaim)
		return err
	})
	if err != nil {
		return nil, false, err
	}
	if granted {
		syncCreditUserQuotaCache(userId, grant.Quota, "activity campaign claim")
	}
	return grant, granted, nil
}

func IsActivityCampaignUserEligible(ctx context.Context, campaign *ActivityCampaign, userId int) (bool, error) {
	if campaign == nil || userId <= 0 {
		return false, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return isActivityCampaignUserEligible(DB.WithContext(ctx), campaign, userId)
}

func isActivityCampaignUserEligible(tx *gorm.DB, campaign *ActivityCampaign, userId int) (bool, error) {
	audienceType := campaign.AudienceType
	if audienceType == "" {
		audienceType = ActivityCampaignAudienceAll
	}
	if audienceType == ActivityCampaignAudienceSelected {
		var count int64
		err := tx.Model(&ActivityCampaignRecipient{}).
			Where("campaign_id = ? AND user_id = ?", campaign.Id, userId).
			Count(&count).Error
		return count == 1, err
	}
	return campaign.RecipientMaxUserID > 0 && userId <= campaign.RecipientMaxUserID, nil
}

func ListActivityCampaignsForUserPage(ctx context.Context, userId int, cursor int64, limit int) ([]*ActivityCampaign, int64, error) {
	if userId <= 0 {
		return nil, 0, errors.New("activity user is invalid")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if cursor < 0 {
		return nil, 0, errors.New("activity campaign cursor is invalid")
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	var campaigns []*ActivityCampaign
	query := DB.WithContext(ctx).
		Where("activity_key IN (?)", DB.WithContext(ctx).Model(&ActivityGrant{}).Select("activity_key").Where("user_id = ?", userId)).
		Order("id desc")
	if cursor > 0 {
		query = query.Where("id < ?", cursor)
	}
	if err := query.Limit(limit + 1).Find(&campaigns).Error; err != nil {
		return nil, 0, err
	}
	if len(campaigns) <= limit {
		return campaigns, 0, nil
	}
	return campaigns[:limit], campaigns[limit-1].Id, nil
}

func HasUnclaimedActivityCampaignForUser(ctx context.Context, userId int, now int64) (bool, error) {
	if userId <= 0 {
		return false, errors.New("activity user is invalid")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	selectedCampaignIds := DB.WithContext(ctx).Model(&ActivityCampaignRecipient{}).
		Select("campaign_id").Where("user_id = ?", userId)
	claimedActivityKeys := DB.WithContext(ctx).Model(&ActivityGrant{}).
		Select("activity_key").
		Where("user_id = ? AND source_type = ? AND source_ref = ?", userId, ActivityGrantSourceCampaignClaim, ActivityGrantSourceRefClaim)
	var count int64
	err := DB.WithContext(ctx).Model(&ActivityCampaign{}).
		Where("type = ? AND status = ? AND starts_at <= ? AND ends_at > ?", ActivityCampaignTypeClaimable, ActivityCampaignStatusActive, now, now).
		Where("((audience_type = ? AND id IN (?)) OR ((audience_type = ? OR audience_type = '') AND recipient_max_user_id >= ?))", ActivityCampaignAudienceSelected, selectedCampaignIds, ActivityCampaignAudienceAll, userId).
		Where("activity_key NOT IN (?)", claimedActivityKeys).
		Limit(1).
		Count(&count).Error
	return count > 0, err
}

// ListActivityGrantsForUserActivityKeys returns grants keyed by activity key
// for the supplied campaigns. A key can have multiple source records, so the
// caller retains the source reference when selecting a user-visible reward.
func ListActivityGrantsForUserActivityKeys(ctx context.Context, userId int, activityKeys []string) (map[string][]ActivityGrant, error) {
	grantsByKey := make(map[string][]ActivityGrant)
	if userId <= 0 || len(activityKeys) == 0 {
		return grantsByKey, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	keys := make([]string, 0, len(activityKeys))
	seen := make(map[string]struct{}, len(activityKeys))
	for _, key := range activityKeys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		keys = append(keys, key)
	}
	if len(keys) == 0 {
		return grantsByKey, nil
	}
	var grants []ActivityGrant
	err := DB.WithContext(ctx).Where("user_id = ? AND activity_key IN ?", userId, keys).
		Order("created_at desc, id desc").
		Find(&grants).Error
	if err != nil {
		return nil, err
	}
	for _, grant := range grants {
		grantsByKey[grant.ActivityKey] = append(grantsByKey[grant.ActivityKey], grant)
	}
	return grantsByKey, nil
}

func ListActivityGrantsForUser(ctx context.Context, userId int) ([]ActivityGrant, error) {
	if userId <= 0 {
		return []ActivityGrant{}, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var grants []ActivityGrant
	err := DB.WithContext(ctx).Where("user_id = ?", userId).Order("created_at desc, id desc").Find(&grants).Error
	return grants, err
}

func ListExistingActivityCampaignKeys(ctx context.Context, activityKeys []string) (map[string]struct{}, error) {
	result := make(map[string]struct{})
	if len(activityKeys) == 0 {
		return result, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var keys []string
	if err := DB.WithContext(ctx).Model(&ActivityCampaign{}).
		Where("activity_key IN ?", activityKeys).Pluck("activity_key", &keys).Error; err != nil {
		return nil, err
	}
	for _, key := range keys {
		result[key] = struct{}{}
	}
	return result, nil
}

// SumActivityGrantQuotaForUser returns the cumulative reward recorded for a
// single durable activity key. It is used by the permanent new-user activity,
// where each redeemed code produces a distinct ledger event.
func SumActivityGrantQuotaForUser(ctx context.Context, userId int, activityKey string) (int64, error) {
	if userId <= 0 || strings.TrimSpace(activityKey) == "" {
		return 0, errors.New("invalid activity grant lookup")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var grants []ActivityGrant
	if err := DB.WithContext(ctx).
		Select("quota").
		Where("user_id = ? AND activity_key = ?", userId, strings.TrimSpace(activityKey)).
		Find(&grants).Error; err != nil {
		return 0, err
	}
	var total int64
	for _, grant := range grants {
		total += int64(grant.Quota)
	}
	return total, nil
}

// GetActivityGrantTargetSnapshot captures the currently enabled user set for
// a bulk grant. GORM's default scope excludes soft-deleted users.
func GetActivityGrantTargetSnapshot(ctx context.Context) (maxUserId int, total int64, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	query := DB.WithContext(ctx).Model(&User{}).Where("status = ?", common.UserStatusEnabled)
	if err = query.Count(&total).Error; err != nil {
		return 0, 0, err
	}
	if total == 0 {
		return 0, 0, nil
	}

	var lastUser User
	if err = query.Select("id").Order("id desc").First(&lastUser).Error; err != nil {
		return 0, 0, err
	}
	return lastUser.Id, total, nil
}

func CountActivityGrantEligibleUsers(ctx context.Context, maxUserId int) (int64, error) {
	if maxUserId <= 0 {
		return 0, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var total int64
	err := DB.WithContext(ctx).Model(&User{}).
		Where("id <= ? AND status = ?", maxUserId, common.UserStatusEnabled).
		Count(&total).Error
	return total, err
}

// ListActivityGrantEligibleUserIds returns one deterministic ID-ordered page
// of currently enabled, non-deleted users inside the enqueue-time ID snapshot.
func ListActivityGrantEligibleUserIds(ctx context.Context, afterUserId int, maxUserId int, limit int) ([]int, error) {
	if maxUserId <= 0 || limit <= 0 {
		return []int{}, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if afterUserId < 0 {
		afterUserId = 0
	}
	var userIds []int
	err := DB.WithContext(ctx).Model(&User{}).
		Where("id > ? AND id <= ? AND status = ?", afterUserId, maxUserId, common.UserStatusEnabled).
		Order("id asc").Limit(limit).Pluck("id", &userIds).Error
	return userIds, err
}

func CountActivityGrants(ctx context.Context, activityKey string) (int64, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var total int64
	err := DB.WithContext(ctx).Model(&ActivityGrant{}).
		Where("activity_key = ?", strings.TrimSpace(activityKey)).
		Count(&total).Error
	return total, err
}

// MigrateActivityGrantIndexes removes the former activity_key + user_id
// uniqueness constraint. Existing rows remain valid under the broader
// source-aware key, and the replacement index is created when needed.
func MigrateActivityGrantIndexes() error {
	migrator := DB.Migrator()
	if !migrator.HasTable(&ActivityGrant{}) {
		return nil
	}
	if migrator.HasIndex(&ActivityGrant{}, "idx_activity_grant_user") {
		if err := migrator.DropIndex(&ActivityGrant{}, "idx_activity_grant_user"); err != nil {
			return fmt.Errorf("drop legacy activity grant index: %w", err)
		}
	}
	if !migrator.HasIndex(&ActivityGrant{}, "idx_activity_grant_source") {
		if err := migrator.CreateIndex(&ActivityGrant{}, "idx_activity_grant_source"); err != nil {
			return fmt.Errorf("create activity grant source index: %w", err)
		}
	}
	return nil
}

func normalizeNewActivityCampaign(campaign *ActivityCampaign) error {
	campaign.ActivityKey = strings.TrimSpace(campaign.ActivityKey)
	campaign.Type = strings.ToLower(strings.TrimSpace(campaign.Type))
	campaign.Status = strings.ToLower(strings.TrimSpace(campaign.Status))
	campaign.Title = strings.TrimSpace(campaign.Title)
	campaign.Description = strings.TrimSpace(campaign.Description)
	campaign.Reason = strings.TrimSpace(campaign.Reason)
	campaign.AmountUSD = strings.TrimSpace(campaign.AmountUSD)
	if campaign.AudienceType == "" {
		campaign.AudienceType = ActivityCampaignAudienceAll
	}
	if campaign.ActivityKey == "" || utf8.RuneCountInString(campaign.ActivityKey) > 128 || !isActivityCampaignKey(campaign.ActivityKey) {
		return errors.New("activity campaign key is invalid")
	}
	if campaign.ActivityKey == ActivityKeyNewUserRedeemBonus {
		return errors.New("activity campaign key is reserved")
	}
	if campaign.Title == "" || utf8.RuneCountInString(campaign.Title) > 128 {
		return errors.New("activity campaign title is invalid")
	}
	if utf8.RuneCountInString(campaign.Description) > 4000 || utf8.RuneCountInString(campaign.Reason) > 255 {
		return errors.New("activity campaign text is too long")
	}
	if campaign.AmountUSD == "" || utf8.RuneCountInString(campaign.AmountUSD) > 64 {
		return errors.New("activity campaign amount is invalid")
	}
	if campaign.Quota <= 0 || campaign.Quota > common.MaxQuota {
		return errors.New("activity campaign quota is invalid")
	}
	if campaign.AudienceType != ActivityCampaignAudienceAll && campaign.AudienceType != ActivityCampaignAudienceSelected {
		return errors.New("activity campaign audience is invalid")
	}
	if campaign.AudienceType == ActivityCampaignAudienceSelected && campaign.Type != ActivityCampaignTypeClaimable {
		return errors.New("selected activity campaigns must be claimable")
	}
	if campaign.CreatedBy <= 0 || campaign.StartsAt < 0 || campaign.EndsAt < 0 {
		return errors.New("activity campaign metadata is invalid")
	}

	switch campaign.Type {
	case ActivityCampaignTypeClaimable:
		if campaign.StartsAt == 0 {
			campaign.StartsAt = common.GetTimestamp()
		}
		if campaign.EndsAt <= campaign.StartsAt {
			return errors.New("claimable activity campaign end time is invalid")
		}
		if campaign.Status == "" {
			campaign.Status = ActivityCampaignStatusActive
		}
		if campaign.Status != ActivityCampaignStatusActive {
			return ErrActivityCampaignInvalidStatus
		}
	case ActivityCampaignTypeImmediate:
		if campaign.Status == "" {
			campaign.Status = ActivityCampaignStatusQueued
		}
		if campaign.Status != ActivityCampaignStatusQueued {
			return ErrActivityCampaignInvalidStatus
		}
	default:
		return errors.New("activity campaign type is invalid")
	}
	return nil
}

func isActivityCampaignKey(key string) bool {
	for _, char := range key {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '_' || char == '-' {
			continue
		}
		return false
	}
	return true
}
