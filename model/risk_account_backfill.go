package model

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

const (
	RiskBackfillStatusPending        = "pending"
	RiskBackfillStatusRunning        = "running"
	RiskBackfillStatusPauseRequested = "pause_requested"
	RiskBackfillStatusPaused         = "paused"
	RiskBackfillStatusSucceeded      = "succeeded"
	RiskBackfillStatusFailed         = "failed"
)

var ErrRiskBackfillPaused = errors.New("risk account backfill paused")

type RiskAccountBackfillRun struct {
	RunID             string `json:"run_id" gorm:"type:varchar(64);primaryKey"`
	TaskID            string `json:"task_id" gorm:"type:varchar(64);index"`
	Status            string `json:"status" gorm:"type:varchar(24);not null;index"`
	MinUserID         int    `json:"min_user_id" gorm:"not null"`
	MaxUserID         int    `json:"max_user_id" gorm:"not null"`
	CreatedFrom       int64  `json:"created_from" gorm:"bigint;not null"`
	CreatedTo         int64  `json:"created_to" gorm:"bigint;not null"`
	PageSize          int    `json:"page_size" gorm:"not null"`
	LastUserID        int    `json:"last_user_id" gorm:"not null"`
	TotalUsers        int64  `json:"total_users" gorm:"bigint;not null"`
	ProcessedUsers    int64  `json:"processed_users" gorm:"bigint;not null"`
	CandidateGroups   int64  `json:"candidate_groups" gorm:"bigint;not null"`
	CandidateAccounts int64  `json:"candidate_accounts" gorm:"bigint;not null"`
	CreatedBy         int    `json:"created_by" gorm:"not null;index"`
	CreatedAt         int64  `json:"created_at" gorm:"bigint;not null;index"`
	UpdatedAt         int64  `json:"updated_at" gorm:"bigint;not null;index"`
	Error             string `json:"error" gorm:"type:text"`
}

func (RiskAccountBackfillRun) TableName() string { return "risk_account_backfill_runs" }

type RiskBackfillOptions struct {
	MinUserID   int   `json:"min_user_id"`
	MaxUserID   int   `json:"max_user_id"`
	CreatedFrom int64 `json:"created_from"`
	CreatedTo   int64 `json:"created_to"`
	PageSize    int   `json:"page_size"`
}

type RiskBackfillPreviewResult struct {
	CandidateGroups   int64 `json:"candidate_groups"`
	CandidateAccounts int64 `json:"candidate_accounts"`
}

func scopedRiskBackfillUsers(queryOptions RiskBackfillOptions) *gorm.DB {
	query := DB.Unscoped().Model(&User{}).Where("role < ?", common.RoleAdminUser)
	if queryOptions.MinUserID > 0 {
		query = query.Where("id >= ?", queryOptions.MinUserID)
	}
	if queryOptions.MaxUserID > 0 {
		query = query.Where("id <= ?", queryOptions.MaxUserID)
	}
	if queryOptions.CreatedFrom > 0 {
		query = query.Where("created_at >= ?", queryOptions.CreatedFrom)
	}
	if queryOptions.CreatedTo > 0 {
		query = query.Where("created_at <= ?", queryOptions.CreatedTo)
	}
	return query
}

func CreateRiskAccountBackfillRun(options RiskBackfillOptions, createdBy int) (*RiskAccountBackfillRun, error) {
	if createdBy <= 0 || options.MinUserID < 0 || options.MaxUserID < 0 || options.CreatedFrom < 0 || options.CreatedTo < 0 {
		return nil, errors.New("invalid risk backfill options")
	}
	if options.MaxUserID > 0 && options.MinUserID > options.MaxUserID {
		return nil, errors.New("minimum user id cannot exceed maximum user id")
	}
	if options.CreatedTo > 0 && options.CreatedFrom > options.CreatedTo {
		return nil, errors.New("start time cannot exceed end time")
	}
	if options.PageSize < 50 || options.PageSize > 1000 {
		options.PageSize = 200
	}
	if options.MaxUserID == 0 {
		var last User
		err := scopedRiskBackfillUsers(options).Order("id desc").First(&last).Error
		if err == nil {
			options.MaxUserID = last.Id
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	var total int64
	if err := scopedRiskBackfillUsers(options).Count(&total).Error; err != nil {
		return nil, err
	}
	random, err := common.GenerateRandomCharsKey(32)
	if err != nil {
		return nil, err
	}
	now := time.Now().Unix()
	run := &RiskAccountBackfillRun{
		RunID: "riskrun_" + random, Status: RiskBackfillStatusPending,
		MinUserID: options.MinUserID, MaxUserID: options.MaxUserID,
		CreatedFrom: options.CreatedFrom, CreatedTo: options.CreatedTo,
		PageSize: options.PageSize, TotalUsers: total, CreatedBy: createdBy,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := DB.Create(run).Error; err != nil {
		return nil, err
	}
	return run, nil
}

func GetRiskAccountBackfillRun(runID string) (*RiskAccountBackfillRun, error) {
	var run RiskAccountBackfillRun
	if err := DB.Where("run_id = ?", strings.TrimSpace(runID)).First(&run).Error; err != nil {
		return nil, err
	}
	return &run, nil
}

func GetLatestRiskAccountBackfillRun() (*RiskAccountBackfillRun, error) {
	var run RiskAccountBackfillRun
	if err := DB.Order("created_at desc").First(&run).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &run, nil
}

func SetRiskAccountBackfillTask(runID, taskID string) error {
	return DB.Model(&RiskAccountBackfillRun{}).Where("run_id = ?", runID).Updates(map[string]any{
		"task_id": taskID, "updated_at": time.Now().Unix(),
	}).Error
}

func RequestRiskAccountBackfillPause(runID string) error {
	result := DB.Model(&RiskAccountBackfillRun{}).
		Where("run_id = ? AND status IN ?", runID, []string{RiskBackfillStatusPending, RiskBackfillStatusRunning}).
		Updates(map[string]any{"status": RiskBackfillStatusPauseRequested, "updated_at": time.Now().Unix()})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("risk backfill is not running")
	}
	return nil
}

func PrepareRiskAccountBackfillResume(runID string) (*RiskAccountBackfillRun, error) {
	result := DB.Model(&RiskAccountBackfillRun{}).
		Where("run_id = ? AND status IN ?", runID, []string{RiskBackfillStatusPaused, RiskBackfillStatusFailed}).
		Updates(map[string]any{
			"status": RiskBackfillStatusPending, "task_id": "", "last_user_id": 0,
			"processed_users": 0, "error": "", "updated_at": time.Now().Unix(),
		})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, errors.New("risk backfill cannot be resumed")
	}
	return GetRiskAccountBackfillRun(runID)
}

func MarkRiskAccountBackfillStatus(runID, status, errorMessage string) error {
	return DB.Model(&RiskAccountBackfillRun{}).Where("run_id = ?", runID).Updates(map[string]any{
		"status": status, "error": errorMessage, "updated_at": time.Now().Unix(),
	}).Error
}

func UpdateRiskAccountBackfillProgress(runID string, lastUserID int, processed int64) error {
	var run RiskAccountBackfillRun
	if err := DB.Select("status").Where("run_id = ?", runID).First(&run).Error; err != nil {
		return err
	}
	if run.Status == RiskBackfillStatusPauseRequested {
		return ErrRiskBackfillPaused
	}
	return DB.Model(&RiskAccountBackfillRun{}).Where("run_id = ?", runID).Updates(map[string]any{
		"status": RiskBackfillStatusRunning, "last_user_id": lastUserID,
		"processed_users": processed, "updated_at": time.Now().Unix(),
	}).Error
}

type riskBackfillEdge struct {
	a    int
	b    int
	rule string
}

type riskLoginSignal struct {
	userID     int
	observedAt int64
}

func PreviewRiskAccountBackfill(ctx context.Context, run *RiskAccountBackfillRun, progress func(lastUserID int, processed int64) error) (RiskBackfillPreviewResult, error) {
	result := RiskBackfillPreviewResult{}
	if run == nil {
		return result, errors.New("risk backfill run is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	options := RiskBackfillOptions{
		MinUserID: run.MinUserID, MaxUserID: run.MaxUserID,
		CreatedFrom: run.CreatedFrom, CreatedTo: run.CreatedTo, PageSize: run.PageSize,
	}
	exemptions, err := ListActiveRiskExemptionIDs(time.Now().Unix())
	if err != nil {
		return result, err
	}
	users := make([]User, 0)
	afterID := 0
	processed := int64(0)
	for {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		var page []User
		query := scopedRiskBackfillUsers(options).Where("id > ?", afterID).Order("id asc").Limit(options.PageSize)
		if err := query.Find(&page).Error; err != nil {
			return result, err
		}
		if len(page) == 0 {
			break
		}
		for _, user := range page {
			if !isRiskExemptUser(&user, time.Now().Unix(), exemptions) {
				users = append(users, user)
			}
		}
		afterID = page[len(page)-1].Id
		processed += int64(len(page))
		if progress != nil {
			if err := progress(afterID, processed); err != nil {
				return result, err
			}
		}
	}
	if len(users) < 2 {
		return result, nil
	}

	userByID := make(map[int]User, len(users))
	parent := make(map[int]int, len(users))
	for _, user := range users {
		userByID[user.Id] = user
		parent[user.Id] = user.Id
	}
	var find func(int) int
	find = func(id int) int {
		if parent[id] != id {
			parent[id] = find(parent[id])
		}
		return parent[id]
	}
	union := func(a, b int) {
		if _, ok := parent[a]; !ok {
			return
		}
		if _, ok := parent[b]; !ok {
			return
		}
		rootA, rootB := find(a), find(b)
		if rootA != rootB {
			parent[rootB] = rootA
		}
	}
	edges := make([]riskBackfillEdge, 0)
	addEdge := func(a, b int, rule string) {
		if a == b {
			return
		}
		if _, ok := parent[a]; !ok {
			return
		}
		if _, ok := parent[b]; !ok {
			return
		}
		union(a, b)
		edges = append(edges, riskBackfillEdge{a: a, b: b, rule: rule})
	}

	identityOwner := map[string]int{}
	for _, user := range users {
		for _, key := range riskIdentityKeys(&user) {
			if owner, exists := identityOwner[key]; exists {
				addEdge(owner, user.Id, "exact_identity")
			} else {
				identityOwner[key] = user.Id
			}
		}
	}

	// Generic OAuth identities live in their own table. Built-in provider
	// subjects are already covered by riskIdentityKeys above.
	userIDs := make([]int, 0, len(users))
	for _, user := range users {
		userIDs = append(userIDs, user.Id)
	}
	for start := 0; start < len(userIDs); start += 500 {
		end := start + 500
		if end > len(userIDs) {
			end = len(userIDs)
		}
		var bindings []UserOAuthBinding
		if err := DB.Where("user_id IN ?", userIDs[start:end]).Find(&bindings).Error; err != nil {
			return result, err
		}
		for _, binding := range bindings {
			if strings.TrimSpace(binding.ProviderUserId) == "" {
				continue
			}
			key := riskHMAC(fmt.Sprintf("custom_oauth:%d:%s", binding.ProviderId, binding.ProviderUserId))
			if key == "" {
				continue
			}
			if owner, exists := identityOwner[key]; exists {
				addEdge(owner, binding.UserId, "exact_oauth_identity")
			} else {
				identityOwner[key] = binding.UserId
			}
		}
	}

	loginGroups := map[string]map[int]int64{}
	addLoginSignal := func(userID int, ip, userAgent string, observedAt int64) {
		if _, ok := parent[userID]; !ok || strings.TrimSpace(ip) == "" || strings.TrimSpace(userAgent) == "" {
			return
		}
		key := riskHMAC("ip:"+ip) + ":" + riskHMAC("ua:"+userAgent)
		if loginGroups[key] == nil {
			loginGroups[key] = map[int]int64{}
		}
		if previous, ok := loginGroups[key][userID]; !ok || observedAt < previous {
			loginGroups[key][userID] = observedAt
		}
	}
	for start := 0; start < len(userIDs); start += 500 {
		end := start + 500
		if end > len(userIDs) {
			end = len(userIDs)
		}
		var sessions []UserSession
		if err := DB.Where("user_id IN ?", userIDs[start:end]).Find(&sessions).Error; err != nil {
			return result, err
		}
		for _, session := range sessions {
			addLoginSignal(session.UserID, session.IP, session.UserAgent, session.CreatedAt)
		}
		if LOG_DB != nil {
			var logs []Log
			if err := LOG_DB.Where("type = ? AND user_id IN ?", LogTypeLogin, userIDs[start:end]).Find(&logs).Error; err != nil {
				return result, err
			}
			for _, log := range logs {
				var other map[string]any
				if err := common.UnmarshalJsonStr(log.Other, &other); err != nil {
					continue
				}
				userAgent, _ := other["user_agent"].(string)
				addLoginSignal(log.UserId, log.Ip, userAgent, log.CreatedAt)
			}
		}
	}
	for _, group := range loginGroups {
		signals := make([]riskLoginSignal, 0, len(group))
		for userID, observedAt := range group {
			signals = append(signals, riskLoginSignal{userID: userID, observedAt: observedAt})
		}
		sort.Slice(signals, func(i, j int) bool { return signals[i].observedAt < signals[j].observedAt })
		for start := 0; start < len(signals); {
			end := start + 1
			for end < len(signals) && signals[end].observedAt-signals[start].observedAt <= 24*60*60 {
				end++
			}
			if end-start >= 2 {
				for i := start + 1; i < end; i++ {
					addEdge(signals[start].userID, signals[i].userID, "same_ip_user_agent_24h")
				}
			}
			start = end
		}
	}

	// A referral burst is never auto-enforced. During historical preview it is
	// surfaced only when at least three accounts share an inviter, register in
	// a 30-minute window, and all receive a tracked new-user reward.
	rewarded := map[int]bool{}
	for start := 0; start < len(userIDs); start += 500 {
		end := start + 500
		if end > len(userIDs) {
			end = len(userIDs)
		}
		var grantUserIDs []int
		if err := DB.Model(&ActivityGrant{}).
			Where("activity_key = ? AND user_id IN ?", ActivityKeyNewUserRedeemBonus, userIDs[start:end]).
			Distinct("user_id").Pluck("user_id", &grantUserIDs).Error; err != nil {
			return result, err
		}
		for _, userID := range grantUserIDs {
			rewarded[userID] = true
		}
	}
	byInviter := map[int][]User{}
	for _, user := range users {
		if user.InviterId > 0 && rewarded[user.Id] {
			byInviter[user.InviterId] = append(byInviter[user.InviterId], user)
		}
	}
	for _, invitees := range byInviter {
		sort.Slice(invitees, func(i, j int) bool { return accountSortLess(invitees[i], invitees[j]) })
		for start := 0; start < len(invitees); {
			end := start + 1
			for end < len(invitees) && invitees[end].CreatedAt-invitees[start].CreatedAt <= 30*60 {
				end++
			}
			if end-start >= 3 {
				for i := start + 1; i < end; i++ {
					addEdge(invitees[start].Id, invitees[i].Id, "inviter_reward_burst")
				}
			}
			start = end
		}
	}

	components := map[int][]User{}
	for id, user := range userByID {
		components[find(id)] = append(components[find(id)], user)
	}
	for root, members := range components {
		if len(members) < 2 {
			continue
		}
		rules := map[string]bool{}
		for _, edge := range edges {
			if find(edge.a) == root && find(edge.b) == root {
				rules[edge.rule] = true
			}
		}
		ruleList := make([]string, 0, len(rules))
		for rule := range rules {
			ruleList = append(ruleList, rule)
		}
		sort.Strings(ruleList)
		memberIDs := make([]int, 0, len(members))
		for _, member := range members {
			memberIDs = append(memberIDs, member.Id)
		}
		_, created, err := CreateRiskAccountReview(RiskAccountReviewKindHistorical, strings.Join(ruleList, ","), memberIDs, run.CreatedBy)
		if err != nil {
			return result, err
		}
		if created {
			result.CandidateGroups++
			result.CandidateAccounts += int64(len(members))
		}
	}
	return result, nil
}

func CompleteRiskAccountBackfillRun(runID string, result RiskBackfillPreviewResult) error {
	return DB.Model(&RiskAccountBackfillRun{}).Where("run_id = ?", runID).Updates(map[string]any{
		"status": RiskBackfillStatusSucceeded, "candidate_groups": result.CandidateGroups,
		"candidate_accounts": result.CandidateAccounts, "error": "", "updated_at": time.Now().Unix(),
	}).Error
}
