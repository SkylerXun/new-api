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

// RiskExemption is an account-level opt-out from account-linking enforcement.
// Administrator accounts are exempt by role and do not need a row here.
type RiskExemption struct {
	UserID    int    `json:"user_id" gorm:"primaryKey"`
	Reason    string `json:"reason" gorm:"type:varchar(255);not null;default:''"`
	CreatedBy int    `json:"created_by" gorm:"not null;index"`
	CreatedAt int64  `json:"created_at" gorm:"bigint;not null;index"`
	ExpiresAt int64  `json:"expires_at" gorm:"bigint;not null;default:0;index"`
	Enabled   bool   `json:"enabled" gorm:"not null"`
}

func (RiskExemption) TableName() string { return "risk_exemptions" }

// RiskAccountLink materializes the current account cluster. Every ordinary
// account in a cluster has one row, including the main account itself.
type RiskAccountLink struct {
	UserID        int    `json:"user_id" gorm:"primaryKey"`
	MainUserID    int    `json:"main_user_id" gorm:"not null;index"`
	ClusterID     string `json:"cluster_id" gorm:"type:varchar(64);not null;index"`
	Relation      string `json:"relation" gorm:"type:varchar(16);not null"`
	Evidence      string `json:"evidence" gorm:"type:text"`
	PolicyVersion string `json:"policy_version" gorm:"type:varchar(32);not null"`
	DecisionAt    int64  `json:"decision_at" gorm:"bigint;not null;index"`
	CreatedAt     int64  `json:"created_at" gorm:"bigint;not null;index"`
	UpdatedAt     int64  `json:"updated_at" gorm:"bigint;not null;index"`
}

func (RiskAccountLink) TableName() string { return "risk_account_links" }

// RiskAccountObservation stores server-side signals used for linking. Values
// are HMACs so raw IPs and user agents are not copied into the risk tables.
type RiskAccountObservation struct {
	ID            int64  `json:"id" gorm:"primaryKey"`
	UserID        int    `json:"user_id" gorm:"not null;index"`
	IPHash        string `json:"ip_hash" gorm:"type:char(64);index"`
	UserAgentHash string `json:"user_agent_hash" gorm:"type:char(64);index"`
	DeviceHash    string `json:"device_hash" gorm:"type:char(64);index"`
	ObservedAt    int64  `json:"observed_at" gorm:"bigint;not null;index"`
}

func (RiskAccountObservation) TableName() string { return "risk_account_observations" }

type LinkedAccountView struct {
	ID          int    `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Status      string `json:"status"`
	CreatedAt   int64  `json:"created_at"`
	Relation    string `json:"relation"`
}

type AccountLinkageView struct {
	Relation       string              `json:"relation"`
	MainAccount    *LinkedAccountView  `json:"main_account,omitempty"`
	LinkedAccounts []LinkedAccountView `json:"linked_accounts"`
	Total          int                 `json:"total"`
}

func linkedAccountStatus(status int) string {
	if status == common.UserStatusEnabled {
		return "enabled"
	}
	return "disabled"
}

func isRiskExemptUser(user *User, now int64, exemptions map[int]bool) bool {
	if user == nil || user.Role >= common.RoleAdminUser {
		return true
	}
	return exemptions[user.Id]
}

func ListActiveRiskExemptionIDs(now int64) (map[int]bool, error) {
	if now <= 0 {
		now = time.Now().Unix()
	}
	var rows []RiskExemption
	if err := DB.Where("enabled = ? AND (expires_at = ? OR expires_at > ?)", true, 0, now).Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[int]bool, len(rows))
	for _, row := range rows {
		result[row.UserID] = true
	}
	return result, nil
}

func IsRiskExemptUser(userID int) (bool, error) {
	if userID <= 0 {
		return false, errors.New("invalid user id")
	}
	var user User
	if err := DB.Unscoped().Select("id", "role").Where("id = ?", userID).First(&user).Error; err != nil {
		return false, err
	}
	if user.Role >= common.RoleAdminUser {
		return true, nil
	}
	rows, err := ListActiveRiskExemptionIDs(time.Now().Unix())
	return err == nil && rows[userID], err
}

func IsLinkedSubaccount(userID int) bool {
	if userID <= 0 || DB == nil {
		return false
	}
	exempt, err := IsRiskExemptUser(userID)
	if err == nil && exempt {
		return false
	}
	var link RiskAccountLink
	return DB.Where("user_id = ? AND relation = ?", userID, "subaccount").First(&link).Error == nil
}

func UpsertRiskExemption(row *RiskExemption) error {
	if row == nil || row.UserID <= 0 || row.CreatedBy <= 0 {
		return errors.New("invalid risk exemption")
	}
	if row.CreatedAt == 0 {
		row.CreatedAt = time.Now().Unix()
	}
	row.Reason = strings.TrimSpace(row.Reason)
	row.Enabled = true
	return DB.Save(row).Error
}

func DisableRiskExemption(userID int) error {
	if userID <= 0 {
		return errors.New("invalid user id")
	}
	return DB.Model(&RiskExemption{}).Where("user_id = ?", userID).Updates(map[string]any{
		"enabled": false,
	}).Error
}

func ListRiskExemptions() ([]RiskExemption, error) {
	var rows []RiskExemption
	err := DB.Order("created_at desc, user_id desc").Find(&rows).Error
	return rows, err
}

func riskHMAC(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return common.GenerateHMACWithKey([]byte("risk-account-v1:"+common.SessionSecret), strings.ToLower(value))
}

func riskIdentityKeys(user *User) []string {
	if user == nil {
		return nil
	}
	values := []string{
		"email:" + NormalizeEmail(user.Email),
		"github:" + strings.TrimSpace(user.GitHubId),
		"discord:" + strings.TrimSpace(user.DiscordId),
		"oidc:" + strings.TrimSpace(user.OidcId),
		"wechat:" + strings.TrimSpace(user.WeChatId),
		"telegram:" + strings.TrimSpace(user.TelegramId),
		"linuxdo:" + strings.TrimSpace(user.LinuxDOId),
		"stripe_customer:" + strings.TrimSpace(user.StripeCustomer),
	}
	keys := make([]string, 0, len(values))
	for _, value := range values {
		if strings.HasSuffix(value, ":") {
			continue
		}
		keys = append(keys, riskHMAC(value))
	}
	return keys
}

func observeRiskAccount(userID int, ip, userAgent, deviceID string, observedAt int64) error {
	if userID <= 0 {
		return errors.New("invalid user id")
	}
	if observedAt <= 0 {
		observedAt = time.Now().Unix()
	}
	return DB.Create(&RiskAccountObservation{
		UserID:        userID,
		IPHash:        riskHMAC("ip:" + ip),
		UserAgentHash: riskHMAC("ua:" + userAgent),
		DeviceHash:    riskHMAC("device:" + deviceID),
		ObservedAt:    observedAt,
	}).Error
}

func accountSortLess(a, b User) bool {
	if a.DeletedAt.Valid != b.DeletedAt.Valid {
		return !a.DeletedAt.Valid
	}
	if a.CreatedAt != b.CreatedAt {
		return a.CreatedAt < b.CreatedAt
	}
	return a.Id < b.Id
}

func accountClusterID(mainID int, members []User) string {
	ids := make([]int, 0, len(members))
	for _, member := range members {
		ids = append(ids, member.Id)
	}
	sort.Ints(ids)
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, fmt.Sprintf("%d", id))
	}
	return common.GenerateHMAC("cluster:" + fmt.Sprintf("%d:%s", mainID, strings.Join(parts, ",")))[:32]
}

// EvaluateAndEnforceAccountLinkage records current signals and links ordinary
// accounts sharing a strong identity or a device fingerprint. The operation
// is deliberately conservative: IP or User-Agent alone never creates a link.
func EvaluateAndEnforceAccountLinkage(userID int, ip, userAgent, deviceID string) error {
	if userID <= 0 {
		return errors.New("invalid user id")
	}
	var current User
	if err := DB.Unscoped().Where("id = ?", userID).First(&current).Error; err != nil {
		return err
	}
	exemptions, err := ListActiveRiskExemptionIDs(time.Now().Unix())
	if err != nil {
		return err
	}
	if isRiskExemptUser(&current, time.Now().Unix(), exemptions) {
		return nil
	}
	if ip != "" || userAgent != "" || deviceID != "" {
		if err := observeRiskAccount(userID, ip, userAgent, deviceID, time.Now().Unix()); err != nil {
			return err
		}
	}

	var users []User
	if err := DB.Unscoped().Find(&users).Error; err != nil {
		return err
	}
	eligibleUserIDs := make(map[int]bool, len(users))
	identityOwners := map[string][]int{}
	for _, user := range users {
		if isRiskExemptUser(&user, time.Now().Unix(), exemptions) {
			continue
		}
		eligibleUserIDs[user.Id] = true
		for _, key := range riskIdentityKeys(&user) {
			if key != "" {
				identityOwners[key] = append(identityOwners[key], user.Id)
			}
		}
	}
	currentIdentityKeys := riskIdentityKeys(&current)
	var oauthBindings []UserOAuthBinding
	if err := DB.Find(&oauthBindings).Error; err != nil {
		return err
	}
	for _, binding := range oauthBindings {
		// A binding owned by an administrator or an active exemption must
		// never become a bridge into the ordinary-user graph.
		if !eligibleUserIDs[binding.UserId] || strings.TrimSpace(binding.ProviderUserId) == "" {
			continue
		}
		key := riskHMAC(fmt.Sprintf("custom_oauth:%d:%s", binding.ProviderId, binding.ProviderUserId))
		if key == "" {
			continue
		}
		identityOwners[key] = append(identityOwners[key], binding.UserId)
		if binding.UserId == current.Id {
			currentIdentityKeys = append(currentIdentityKeys, key)
		}
	}

	// Exact identity matches are deterministic. A device hash is accepted only
	// when it was supplied by at least two different ordinary accounts.
	memberIDs := map[int]bool{current.Id: true}
	for _, other := range users {
		if other.Id == current.Id || isRiskExemptUser(&other, time.Now().Unix(), exemptions) {
			continue
		}
		matched := false
		for _, key := range currentIdentityKeys {
			if key != "" && containsInt(identityOwners[key], other.Id) {
				matched = true
				break
			}
		}
		if matched {
			memberIDs[other.Id] = true
		}
	}
	if deviceID != "" {
		deviceHash := riskHMAC("device:" + deviceID)
		var observations []RiskAccountObservation
		if err := DB.Where("device_hash = ? AND observed_at > ?", deviceHash, time.Now().Add(-30*24*time.Hour).Unix()).Find(&observations).Error; err != nil {
			return err
		}
		for _, observation := range observations {
			if observation.UserID != current.Id && eligibleUserIDs[observation.UserID] {
				memberIDs[observation.UserID] = true
			}
		}
	}
	if len(memberIDs) < 2 {
		return nil
	}

	// Include every member of any existing cluster touched by the new strong
	// match. A single subaccount seed must expand to the whole cluster.
	var seedLinks []RiskAccountLink
	if err := DB.Where("user_id IN ?", mapKeys(memberIDs)).Find(&seedLinks).Error; err != nil {
		return err
	}
	mainIDs := map[int]bool{}
	for _, link := range seedLinks {
		mainIDs[link.MainUserID] = true
	}
	links := seedLinks
	if len(mainIDs) > 0 {
		if err := DB.Where("main_user_id IN ?", mapKeys(mainIDs)).Find(&links).Error; err != nil {
			return err
		}
	}
	for _, link := range links {
		memberIDs[link.UserID] = true
		memberIDs[link.MainUserID] = true
	}

	memberList := make([]User, 0, len(memberIDs))
	for _, user := range users {
		if memberIDs[user.Id] && !isRiskExemptUser(&user, time.Now().Unix(), exemptions) {
			memberList = append(memberList, user)
		}
	}
	if len(memberList) < 2 {
		return nil
	}
	sort.Slice(memberList, func(i, j int) bool { return accountSortLess(memberList[i], memberList[j]) })
	proposedMainID := memberList[0].Id
	for _, link := range links {
		if link.Relation == "main" && link.UserID != proposedMainID {
			_, _, err := CreateRiskAccountReview(RiskAccountReviewKindClusterMerge, "strong_identity_or_device", mapKeys(memberIDs), 0)
			return err
		}
	}
	return enforceRiskAccountGroup(memberList, "strong_identity_or_device")
}

func enforceRiskAccountGroup(memberList []User, evidence string) error {
	if len(memberList) < 2 {
		return nil
	}
	sort.Slice(memberList, func(i, j int) bool { return accountSortLess(memberList[i], memberList[j]) })
	mainID := memberList[0].Id
	clusterID := accountClusterID(mainID, memberList)
	now := time.Now().Unix()
	err := DB.Transaction(func(tx *gorm.DB) error {
		for _, member := range memberList {
			relation := "subaccount"
			if member.Id == mainID {
				relation = "main"
			}
			row := RiskAccountLink{
				UserID: member.Id, MainUserID: mainID, ClusterID: clusterID,
				Relation: relation, Evidence: evidence, PolicyVersion: RiskAccountPolicyVersion,
				DecisionAt: now, CreatedAt: now, UpdatedAt: now,
			}
			var existing RiskAccountLink
			findErr := tx.Where("user_id = ?", member.Id).First(&existing).Error
			switch {
			case errors.Is(findErr, gorm.ErrRecordNotFound):
				if err := tx.Create(&row).Error; err != nil {
					return err
				}
			case findErr != nil:
				return findErr
			default:
				if err := tx.Model(&RiskAccountLink{}).Where("user_id = ?", member.Id).Updates(map[string]any{
					"main_user_id":   mainID,
					"cluster_id":     clusterID,
					"relation":       relation,
					"evidence":       evidence,
					"policy_version": RiskAccountPolicyVersion,
					"decision_at":    now,
					"updated_at":     now,
				}).Error; err != nil {
					return err
				}
			}
			if member.Id != mainID && member.Status != common.UserStatusDisabled {
				if err := tx.Model(&User{}).Where("id = ? AND role < ?", member.Id, common.RoleAdminUser).Update("status", common.UserStatusDisabled).Error; err != nil {
					return err
				}
			}
			if member.Id != mainID {
				if err := recordRiskRewardHoldTx(tx, member, clusterID); err != nil {
					return err
				}
				if err := recordRiskAccountActionTx(tx, member.Id, mainID, clusterID, "disable_subaccount", evidence, 0); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, member := range memberList {
		if member.Id == mainID {
			continue
		}
		_, _ = RevokeAllUserSessions(member.Id, "risk_account_linked")
		_ = PublishUserAuthCache(member.Id)
	}
	return nil
}

func containsInt(values []int, wanted int) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}

func mapKeys(values map[int]bool) []int {
	result := make([]int, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	return result
}

func GetAccountLinkage(userID int) (*AccountLinkageView, error) {
	if userID <= 0 {
		return nil, errors.New("invalid user id")
	}
	if exempt, err := IsRiskExemptUser(userID); err == nil && exempt {
		return &AccountLinkageView{Relation: "none", LinkedAccounts: []LinkedAccountView{}, Total: 0}, nil
	}
	var link RiskAccountLink
	if err := DB.Where("user_id = ?", userID).First(&link).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &AccountLinkageView{Relation: "none", LinkedAccounts: []LinkedAccountView{}, Total: 0}, nil
		}
		return nil, err
	}
	var members []RiskAccountLink
	if link.MainUserID == userID {
		if err := DB.Where("main_user_id = ?", link.MainUserID).Order("user_id asc").Find(&members).Error; err != nil {
			return nil, err
		}
	} else {
		members = []RiskAccountLink{link}
		// Use a fresh value here. Reusing link would make GORM add the
		// struct's existing primary key (the subaccount ID) to the query,
		// so the main-account lookup could never succeed.
		var mainLink RiskAccountLink
		if err := DB.Where("user_id = ?", link.MainUserID).First(&mainLink).Error; err != nil {
			return nil, err
		}
		members = append(members, mainLink)
		link = mainLink
	}
	ids := make([]int, 0, len(members))
	for _, member := range members {
		ids = append(ids, member.UserID)
	}
	var users []User
	if err := DB.Unscoped().Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	byID := make(map[int]User, len(users))
	exemptions, exemptionErr := ListActiveRiskExemptionIDs(time.Now().Unix())
	if exemptionErr != nil {
		return nil, exemptionErr
	}
	for _, user := range users {
		if isRiskExemptUser(&user, time.Now().Unix(), exemptions) {
			continue
		}
		byID[user.Id] = user
	}
	mainUser, ok := byID[link.MainUserID]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	result := &AccountLinkageView{Relation: "subaccount", MainAccount: &LinkedAccountView{ID: mainUser.Id, Username: mainUser.Username, DisplayName: mainUser.DisplayName, Status: linkedAccountStatus(mainUser.Status), CreatedAt: mainUser.CreatedAt, Relation: "main"}}
	if userID == link.MainUserID {
		result.Relation = "main"
		result.MainAccount = &LinkedAccountView{ID: mainUser.Id, Username: mainUser.Username, DisplayName: mainUser.DisplayName, Status: linkedAccountStatus(mainUser.Status), CreatedAt: mainUser.CreatedAt, Relation: "main"}
		result.LinkedAccounts = make([]LinkedAccountView, 0, len(users))
		for _, member := range members {
			user, ok := byID[member.UserID]
			if !ok {
				continue
			}
			result.LinkedAccounts = append(result.LinkedAccounts, LinkedAccountView{ID: user.Id, Username: user.Username, DisplayName: user.DisplayName, Status: linkedAccountStatus(user.Status), CreatedAt: user.CreatedAt, Relation: member.Relation})
		}
	} else {
		result.LinkedAccounts = []LinkedAccountView{*result.MainAccount}
	}
	result.Total = len(result.LinkedAccounts)
	return result, nil
}
