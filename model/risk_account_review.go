package model

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

const (
	RiskAccountPolicyVersion = "account-link-v1"

	RiskAccountReviewKindHistorical   = "historical_cluster"
	RiskAccountReviewKindClusterMerge = "cluster_merge"

	RiskAccountReviewStatusPending  = "pending"
	RiskAccountReviewStatusApproved = "approved"
	RiskAccountReviewStatusRejected = "rejected"
	RiskAccountReviewStatusFailed   = "failed"

	RiskRewardHoldStatusFrozen   = "frozen"
	RiskRewardHoldStatusReleased = "released"
)

// RiskAccountReview is the administrator approval boundary for historical
// findings and any strong signal that would demote an established main
// account. MemberIDs contains a JSON array of ordinary user IDs only.
type RiskAccountReview struct {
	ID            int64  `json:"id" gorm:"primaryKey"`
	Fingerprint   string `json:"-" gorm:"type:char(64);not null;uniqueIndex"`
	Kind          string `json:"kind" gorm:"type:varchar(32);not null;index"`
	Status        string `json:"status" gorm:"type:varchar(16);not null;index"`
	ProposedMain  int    `json:"proposed_main_user_id" gorm:"not null;index"`
	MemberIDs     string `json:"-" gorm:"type:text;not null"`
	Evidence      string `json:"evidence" gorm:"type:text;not null"`
	PolicyVersion string `json:"policy_version" gorm:"type:varchar(32);not null"`
	CreatedBy     int    `json:"created_by" gorm:"not null;index"`
	ResolvedBy    int    `json:"resolved_by" gorm:"not null"`
	Resolution    string `json:"resolution" gorm:"type:varchar(255)"`
	CreatedAt     int64  `json:"created_at" gorm:"bigint;not null;index"`
	UpdatedAt     int64  `json:"updated_at" gorm:"bigint;not null;index"`
	ResolvedAt    int64  `json:"resolved_at" gorm:"bigint;not null"`
}

func (RiskAccountReview) TableName() string { return "risk_account_reviews" }

// RiskAccountAction is an immutable audit event for automatic enforcement or
// an administrator recovery action. ActionKey makes repeated scans idempotent.
type RiskAccountAction struct {
	ID            int64  `json:"id" gorm:"primaryKey"`
	ActionKey     string `json:"-" gorm:"type:char(64);not null;uniqueIndex"`
	UserID        int    `json:"user_id" gorm:"not null;index"`
	MainUserID    int    `json:"main_user_id" gorm:"not null;index"`
	ClusterID     string `json:"cluster_id" gorm:"type:varchar(64);index"`
	Action        string `json:"action" gorm:"type:varchar(32);not null;index"`
	Rule          string `json:"rule" gorm:"type:varchar(255);not null"`
	PolicyVersion string `json:"policy_version" gorm:"type:varchar(32);not null"`
	CreatedBy     int    `json:"created_by" gorm:"not null"`
	CreatedAt     int64  `json:"created_at" gorm:"bigint;not null;index"`
}

func (RiskAccountAction) TableName() string { return "risk_account_actions" }

// RiskRewardHold records the promotional credit that was still present when
// a subaccount was disabled. The user remains disabled, so the hold is an
// auditable freeze marker rather than a destructive wallet deduction.
type RiskRewardHold struct {
	ID            int64  `json:"id" gorm:"primaryKey"`
	UserID        int    `json:"user_id" gorm:"not null;index;uniqueIndex:idx_risk_reward_hold,priority:1"`
	ClusterID     string `json:"cluster_id" gorm:"type:varchar(64);not null;index;uniqueIndex:idx_risk_reward_hold,priority:2"`
	FrozenQuota   int    `json:"frozen_quota" gorm:"type:bigint;not null"`
	Status        string `json:"status" gorm:"type:varchar(16);not null;index"`
	PolicyVersion string `json:"policy_version" gorm:"type:varchar(32);not null"`
	CreatedAt     int64  `json:"created_at" gorm:"bigint;not null;index"`
	ResolvedAt    int64  `json:"resolved_at" gorm:"bigint;not null"`
}

func (RiskRewardHold) TableName() string { return "risk_reward_holds" }

type RiskAccountReviewView struct {
	ID            int64               `json:"id"`
	Kind          string              `json:"kind"`
	Status        string              `json:"status"`
	ProposedMain  int                 `json:"proposed_main_user_id"`
	Members       []LinkedAccountView `json:"members"`
	Evidence      string              `json:"evidence"`
	PolicyVersion string              `json:"policy_version"`
	CreatedBy     int                 `json:"created_by"`
	ResolvedBy    int                 `json:"resolved_by"`
	Resolution    string              `json:"resolution"`
	CreatedAt     int64               `json:"created_at"`
	UpdatedAt     int64               `json:"updated_at"`
	ResolvedAt    int64               `json:"resolved_at"`
}

func normalizeRiskMemberIDs(memberIDs []int) []int {
	seen := make(map[int]bool, len(memberIDs))
	result := make([]int, 0, len(memberIDs))
	for _, id := range memberIDs {
		if id <= 0 || seen[id] {
			continue
		}
		seen[id] = true
		result = append(result, id)
	}
	sort.Ints(result)
	return result
}

func marshalRiskMemberIDs(memberIDs []int) (string, error) {
	data, err := common.Marshal(normalizeRiskMemberIDs(memberIDs))
	return string(data), err
}

func unmarshalRiskMemberIDs(value string) ([]int, error) {
	var memberIDs []int
	if err := common.UnmarshalJsonStr(value, &memberIDs); err != nil {
		return nil, err
	}
	return normalizeRiskMemberIDs(memberIDs), nil
}

func riskReviewFingerprint(kind string, memberIDs []int) string {
	ids := normalizeRiskMemberIDs(memberIDs)
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, fmt.Sprintf("%d", id))
	}
	return common.GenerateHMAC("risk-review:" + RiskAccountPolicyVersion + ":" + kind + ":" + strings.Join(parts, ","))
}

func CreateRiskAccountReview(kind, evidence string, memberIDs []int, createdBy int) (*RiskAccountReview, bool, error) {
	memberIDs = normalizeRiskMemberIDs(memberIDs)
	if len(memberIDs) < 2 {
		return nil, false, errors.New("at least two risk review members are required")
	}
	var users []User
	if err := DB.Unscoped().Where("id IN ?", memberIDs).Find(&users).Error; err != nil {
		return nil, false, err
	}
	exemptions, err := ListActiveRiskExemptionIDs(time.Now().Unix())
	if err != nil {
		return nil, false, err
	}
	eligible := make([]User, 0, len(users))
	for _, user := range users {
		if !isRiskExemptUser(&user, time.Now().Unix(), exemptions) {
			eligible = append(eligible, user)
		}
	}
	if len(eligible) < 2 {
		return nil, false, nil
	}
	sort.Slice(eligible, func(i, j int) bool { return accountSortLess(eligible[i], eligible[j]) })
	memberIDs = make([]int, 0, len(eligible))
	for _, user := range eligible {
		memberIDs = append(memberIDs, user.Id)
	}
	encoded, err := marshalRiskMemberIDs(memberIDs)
	if err != nil {
		return nil, false, err
	}
	fingerprint := riskReviewFingerprint(kind, memberIDs)
	var existing RiskAccountReview
	if err := DB.Where("fingerprint = ?", fingerprint).First(&existing).Error; err == nil {
		return &existing, false, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}
	now := time.Now().Unix()
	review := &RiskAccountReview{
		Fingerprint:   fingerprint,
		Kind:          kind,
		Status:        RiskAccountReviewStatusPending,
		ProposedMain:  eligible[0].Id,
		MemberIDs:     encoded,
		Evidence:      strings.TrimSpace(evidence),
		PolicyVersion: RiskAccountPolicyVersion,
		CreatedBy:     createdBy,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := DB.Create(review).Error; err != nil {
		if lookupErr := DB.Where("fingerprint = ?", fingerprint).First(&existing).Error; lookupErr == nil {
			return &existing, false, nil
		}
		return nil, false, err
	}
	return review, true, nil
}

func ListRiskAccountReviews(status string, page, pageSize int) ([]RiskAccountReviewView, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	query := DB.Model(&RiskAccountReview{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []RiskAccountReview
	if err := query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	views := make([]RiskAccountReviewView, 0, len(rows))
	for _, row := range rows {
		memberIDs, err := unmarshalRiskMemberIDs(row.MemberIDs)
		if err != nil {
			return nil, 0, err
		}
		var users []User
		if err := DB.Unscoped().Where("id IN ?", memberIDs).Find(&users).Error; err != nil {
			return nil, 0, err
		}
		byID := make(map[int]User, len(users))
		for _, user := range users {
			byID[user.Id] = user
		}
		members := make([]LinkedAccountView, 0, len(memberIDs))
		for _, id := range memberIDs {
			user, ok := byID[id]
			if !ok {
				continue
			}
			relation := "subaccount"
			if id == row.ProposedMain {
				relation = "main"
			}
			members = append(members, LinkedAccountView{ID: user.Id, Username: user.Username, DisplayName: user.DisplayName, Status: linkedAccountStatus(user.Status), CreatedAt: user.CreatedAt, Relation: relation})
		}
		views = append(views, RiskAccountReviewView{
			ID: row.ID, Kind: row.Kind, Status: row.Status, ProposedMain: row.ProposedMain,
			Members: members, Evidence: row.Evidence, PolicyVersion: row.PolicyVersion,
			CreatedBy: row.CreatedBy, ResolvedBy: row.ResolvedBy, Resolution: row.Resolution,
			CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, ResolvedAt: row.ResolvedAt,
		})
	}
	return views, total, nil
}

func ApproveRiskAccountReview(reviewID int64, adminID int, resolution string) error {
	if reviewID <= 0 || adminID <= 0 {
		return errors.New("invalid risk review approval")
	}
	var review RiskAccountReview
	if err := DB.Where("id = ?", reviewID).First(&review).Error; err != nil {
		return err
	}
	if review.Status == RiskAccountReviewStatusApproved {
		return nil
	}
	if review.Status != RiskAccountReviewStatusPending && review.Status != RiskAccountReviewStatusFailed {
		return errors.New("risk review is not pending")
	}
	memberIDs, err := unmarshalRiskMemberIDs(review.MemberIDs)
	if err != nil {
		return err
	}
	var users []User
	if err := DB.Unscoped().Where("id IN ?", memberIDs).Find(&users).Error; err != nil {
		return err
	}
	exemptions, err := ListActiveRiskExemptionIDs(time.Now().Unix())
	if err != nil {
		return err
	}
	eligible := make([]User, 0, len(users))
	for _, user := range users {
		if !isRiskExemptUser(&user, time.Now().Unix(), exemptions) {
			eligible = append(eligible, user)
		}
	}
	if len(eligible) >= 2 {
		var existingLinks []RiskAccountLink
		if err := DB.Where("user_id IN ?", memberIDs).Find(&existingLinks).Error; err != nil {
			return err
		}
		for _, existing := range existingLinks {
			// Historical findings that would demote an established main account
			// first create a dedicated cluster-merge review. Once that explicit
			// review is being approved, allow the recomputed earliest account to
			// become main instead of generating the same review again.
			if review.Kind != RiskAccountReviewKindClusterMerge && existing.Relation == "main" && existing.UserID != eligible[0].Id {
				mergeReview, _, reviewErr := CreateRiskAccountReview(
					RiskAccountReviewKindClusterMerge,
					"historical approval would demote existing main account",
					memberIDs,
					adminID,
				)
				if reviewErr != nil {
					return reviewErr
				}
				message := "cluster merge requires a separate administrator review"
				if mergeReview != nil {
					message = fmt.Sprintf("cluster merge review %d requires approval", mergeReview.ID)
				}
				_ = DB.Model(&RiskAccountReview{}).Where("id = ?", review.ID).Updates(map[string]any{
					"status": RiskAccountReviewStatusFailed, "resolution": message, "updated_at": time.Now().Unix(),
				}).Error
				return errors.New(message)
			}
		}
		if err := enforceRiskAccountGroup(eligible, review.Evidence); err != nil {
			_ = DB.Model(&RiskAccountReview{}).Where("id = ?", review.ID).Updates(map[string]any{
				"status": RiskAccountReviewStatusFailed, "resolution": err.Error(), "updated_at": time.Now().Unix(),
			}).Error
			return err
		}
	}
	now := time.Now().Unix()
	return DB.Model(&RiskAccountReview{}).Where("id = ?", review.ID).Updates(map[string]any{
		"status": RiskAccountReviewStatusApproved, "resolved_by": adminID,
		"resolution": strings.TrimSpace(resolution), "resolved_at": now, "updated_at": now,
	}).Error
}

func RejectRiskAccountReview(reviewID int64, adminID int, resolution string) error {
	if reviewID <= 0 || adminID <= 0 {
		return errors.New("invalid risk review rejection")
	}
	now := time.Now().Unix()
	result := DB.Model(&RiskAccountReview{}).
		Where("id = ? AND status IN ?", reviewID, []string{RiskAccountReviewStatusPending, RiskAccountReviewStatusFailed}).
		Updates(map[string]any{
			"status": RiskAccountReviewStatusRejected, "resolved_by": adminID,
			"resolution": strings.TrimSpace(resolution), "resolved_at": now, "updated_at": now,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		var review RiskAccountReview
		if err := DB.Where("id = ?", reviewID).First(&review).Error; err != nil {
			return err
		}
		if review.Status != RiskAccountReviewStatusRejected {
			return errors.New("risk review is not pending")
		}
	}
	return nil
}
