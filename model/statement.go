package model

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	StatementSourceUserExport    = "user_export"
	StatementSourceAdmin         = "admin"
	StatementSourceSystemMonthly = "system_monthly"
	StatementComplianceVersion   = "cn-consumption-statement-v2"
	StatementRendererVersion     = "go-pdf-v2"
	statementUsageBackfillKey    = "migration.statement_usage_monthly.v1"
)

var statementDisclaimers = []string{
	"本单据不是税务发票、财政票据、付款收据或完税证明。",
	"是否可用于报销由用户所在单位财务审核；正式发票需另行联系。",
	"钱包充值只统计站内确认成功的人民币充值订单。",
	"当前月账单为截至生成时的阶段性数据，后续调用、充值或兑换不会追溯修改已生成版本。",
}

type StatementUsageMonthly struct {
	UserID        int    `json:"user_id" gorm:"primaryKey;autoIncrement:false;uniqueIndex:ux_statement_usage_monthly,priority:1"`
	MonthStart    int64  `json:"month_start" gorm:"primaryKey;autoIncrement:false;uniqueIndex:ux_statement_usage_monthly,priority:2;index"`
	ModelName     string `json:"model_name" gorm:"primaryKey;autoIncrement:false;type:varchar(191);uniqueIndex:ux_statement_usage_monthly,priority:3"`
	InputTokens   int64  `json:"input_tokens"`
	OutputTokens  int64  `json:"output_tokens"`
	BillingCount  int64  `json:"billing_count"`
	LastUpdatedAt int64  `json:"last_updated_at" gorm:"bigint"`
}

type ConsumptionStatement struct {
	ID               int64             `json:"id"`
	StatementNo      string            `json:"statement_no" gorm:"type:varchar(80);uniqueIndex"`
	UserID           int               `json:"user_id" gorm:"index"`
	UsernameSnapshot string            `json:"username" gorm:"type:varchar(191);index"`
	EmailSnapshot    string            `json:"email" gorm:"type:varchar(191);index"`
	MonthStart       int64             `json:"month_start" gorm:"bigint;index"`
	PeriodStart      int64             `json:"period_start" gorm:"bigint"`
	PeriodEnd        int64             `json:"period_end" gorm:"bigint"`
	Timezone         string            `json:"timezone" gorm:"type:varchar(64)"`
	Source           string            `json:"source" gorm:"type:varchar(32);index"`
	IsFinal          bool              `json:"is_final"`
	FinalKey         *string           `json:"-" gorm:"type:varchar(80);uniqueIndex"`
	GeneratedBy      int               `json:"generated_by" gorm:"index"`
	GeneratedAt      int64             `json:"generated_at" gorm:"bigint;index"`
	SnapshotJSON     string            `json:"-" gorm:"type:text;not null"`
	ContentHash      string            `json:"content_hash" gorm:"type:char(64);index"`
	ComplianceVer    string            `json:"compliance_version" gorm:"type:varchar(64)"`
	RendererVer      string            `json:"renderer_version" gorm:"type:varchar(32)"`
	Snapshot         StatementSnapshot `json:"snapshot" gorm:"-:all"`
}

type StatementIssuer struct {
	Name         string `json:"name"`
	ContactEmail string `json:"contact_email"`
	Address      string `json:"address"`
	Website      string `json:"website"`
}

type StatementRecipient struct {
	UserID         int    `json:"user_id"`
	Username       string `json:"username"`
	Email          string `json:"email"`
	BillingTitle   string `json:"billing_title,omitempty"`
	BillingAddress string `json:"billing_address,omitempty"`
	UserSupplied   bool   `json:"user_supplied"`
}

type StatementTokenItem struct {
	ModelName    string `json:"model_name"`
	InputTokens  int64  `json:"input_tokens"`
	OutputTokens int64  `json:"output_tokens"`
	RecordCount  int64  `json:"record_count"`
}

type StatementRedemptionItem struct {
	RecordID    int    `json:"record_id"`
	Category    string `json:"category"`
	AmountCents int64  `json:"amount_cents"`
	RedeemedAt  int64  `json:"redeemed_at"`
}

type StatementTopUpItem struct {
	RecordID    int   `json:"record_id"`
	AmountCents int64 `json:"amount_cents"`
	CompletedAt int64 `json:"completed_at"`
}

type StatementWarnings struct {
	UnpricedRedemptions   int64 `json:"unpriced_redemptions"`
	UnknownCurrencyTopUps int64 `json:"unknown_currency_topups"`
}

type StatementSnapshot struct {
	StatementNo          string                    `json:"statement_no"`
	Issuer               StatementIssuer           `json:"issuer"`
	Recipient            StatementRecipient        `json:"recipient"`
	PeriodStart          int64                     `json:"period_start"`
	PeriodEnd            int64                     `json:"period_end"`
	Timezone             string                    `json:"timezone"`
	Source               string                    `json:"source"`
	IsFinal              bool                      `json:"is_final"`
	GeneratedAt          int64                     `json:"generated_at"`
	GeneratedBy          int                       `json:"generated_by"`
	Tokens               []StatementTokenItem      `json:"tokens"`
	Redemptions          []StatementRedemptionItem `json:"redemptions"`
	TopUps               []StatementTopUpItem      `json:"topups"`
	RedemptionTotalCents int64                     `json:"redemption_total_cents"`
	TopUpTotalCents      int64                     `json:"topup_total_cents"`
	TotalCents           int64                     `json:"total_cents"`
	Warnings             StatementWarnings         `json:"warnings"`
	Disclaimers          []string                  `json:"disclaimers"`
	ComplianceVersion    string                    `json:"compliance_version"`
}

type StatementGenerateInput struct {
	UserID         int
	Month          string
	Source         string
	GeneratedBy    int
	BillingTitle   string
	BillingAddress string
	Now            time.Time
}

func shanghaiLocation() *time.Location {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	return location
}

func StatementMonthBounds(month string, now time.Time) (time.Time, time.Time, bool, error) {
	location := shanghaiLocation()
	if now.IsZero() {
		now = time.Now()
	}
	now = now.In(location)
	month = strings.TrimSpace(month)
	if month == "" {
		month = now.Format("2006-01")
	}
	start, err := time.ParseInLocation("2006-01", month, location)
	if err != nil {
		return time.Time{}, time.Time{}, false, errors.New("月份格式必须为 YYYY-MM")
	}
	currentStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, location)
	if start.After(currentStart) {
		return time.Time{}, time.Time{}, false, errors.New("不能生成未来月份账单")
	}
	isCurrent := start.Equal(currentStart)
	end := start.AddDate(0, 1, 0)
	if isCurrent {
		end = now
	}
	return start, end, isCurrent, nil
}

func normalizeStatementModel(modelName string) string {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return "未标注模型"
	}
	if len([]rune(modelName)) > 191 {
		return string([]rune(modelName)[:191])
	}
	return modelName
}

func IncrementStatementUsage(userID int, createdAt int64, modelName string, inputTokens int, outputTokens int) error {
	if DB == nil || userID <= 0 || createdAt <= 0 {
		return nil
	}
	created := time.Unix(createdAt, 0).In(shanghaiLocation())
	monthStart := time.Date(created.Year(), created.Month(), 1, 0, 0, 0, 0, shanghaiLocation()).Unix()
	row := StatementUsageMonthly{
		UserID: userID, MonthStart: monthStart, ModelName: normalizeStatementModel(modelName),
		InputTokens: int64(max(inputTokens, 0)), OutputTokens: int64(max(outputTokens, 0)), BillingCount: 1,
		LastUpdatedAt: common.GetTimestamp(),
	}
	return DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "month_start"}, {Name: "model_name"}},
		DoUpdates: clause.Assignments(map[string]any{
			"input_tokens":    gorm.Expr("input_tokens + ?", row.InputTokens),
			"output_tokens":   gorm.Expr("output_tokens + ?", row.OutputTokens),
			"billing_count":   gorm.Expr("billing_count + 1"),
			"last_updated_at": row.LastUpdatedAt,
		}),
	}).Create(&row).Error
}

type statementLogAggregate struct {
	UserID       int
	MonthStart   int64
	ModelName    string
	InputTokens  int64
	OutputTokens int64
	BillingCount int64
}

func reconcileStatementAggregateRow(row statementLogAggregate) error {
	row.ModelName = normalizeStatementModel(row.ModelName)
	seed := StatementUsageMonthly{UserID: row.UserID, MonthStart: row.MonthStart, ModelName: row.ModelName}
	if err := DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&seed).Error; err != nil {
		return err
	}
	return DB.Model(&StatementUsageMonthly{}).
		Where("user_id = ? AND month_start = ? AND model_name = ?", row.UserID, row.MonthStart, row.ModelName).
		Updates(map[string]any{
			"input_tokens":    gorm.Expr("CASE WHEN input_tokens < ? THEN ? ELSE input_tokens END", row.InputTokens, row.InputTokens),
			"output_tokens":   gorm.Expr("CASE WHEN output_tokens < ? THEN ? ELSE output_tokens END", row.OutputTokens, row.OutputTokens),
			"billing_count":   gorm.Expr("CASE WHEN billing_count < ? THEN ? ELSE billing_count END", row.BillingCount, row.BillingCount),
			"last_updated_at": common.GetTimestamp(),
		}).Error
}

func ReconcileStatementUsage(userID int, start, end time.Time) error {
	if LOG_DB == nil || userID <= 0 {
		return nil
	}
	var rows []struct {
		ModelName    string
		InputTokens  int64
		OutputTokens int64
		BillingCount int64
	}
	err := LOG_DB.Model(&Log{}).
		Select("model_name, SUM(prompt_tokens) AS input_tokens, SUM(completion_tokens) AS output_tokens, COUNT(*) AS billing_count").
		Where("user_id = ? AND type = ? AND created_at >= ? AND created_at < ?", userID, LogTypeConsume, start.Unix(), end.Unix()).
		Group("model_name").Scan(&rows).Error
	if err != nil {
		return err
	}
	for _, item := range rows {
		if err := reconcileStatementAggregateRow(statementLogAggregate{
			UserID: userID, MonthStart: start.Unix(), ModelName: item.ModelName,
			InputTokens: item.InputTokens, OutputTokens: item.OutputTokens, BillingCount: item.BillingCount,
		}); err != nil {
			return err
		}
	}
	return nil
}

// BackfillStatementUsageFromLogs is idempotent: it only raises persisted
// totals to the values still observable in the log database, so log cleanup
// can never reduce a durable monthly aggregate.
func BackfillStatementUsageFromLogs() error {
	if LOG_DB == nil {
		return nil
	}
	rows, err := LOG_DB.Model(&Log{}).
		Select("user_id, created_at, model_name, prompt_tokens, completion_tokens").
		Where("type = ?", LogTypeConsume).Rows()
	if err != nil {
		return err
	}
	defer rows.Close()
	aggregates := make(map[string]*statementLogAggregate)
	for rows.Next() {
		var userID int
		var createdAt int64
		var modelName string
		var inputTokens int64
		var outputTokens int64
		if err := rows.Scan(&userID, &createdAt, &modelName, &inputTokens, &outputTokens); err != nil {
			return err
		}
		created := time.Unix(createdAt, 0).In(shanghaiLocation())
		monthStart := time.Date(created.Year(), created.Month(), 1, 0, 0, 0, 0, shanghaiLocation()).Unix()
		modelName = normalizeStatementModel(modelName)
		key := fmt.Sprintf("%d:%d:%s", userID, monthStart, modelName)
		aggregate := aggregates[key]
		if aggregate == nil {
			aggregate = &statementLogAggregate{UserID: userID, MonthStart: monthStart, ModelName: modelName}
			aggregates[key] = aggregate
		}
		aggregate.InputTokens += inputTokens
		aggregate.OutputTokens += outputTokens
		aggregate.BillingCount++
	}
	for _, aggregate := range aggregates {
		if err := reconcileStatementAggregateRow(*aggregate); err != nil {
			return err
		}
	}
	return rows.Err()
}

func BackfillStatementUsageFromLogsOnce() error {
	var marker Option
	err := DB.Where("key = ?", statementUsageBackfillKey).First(&marker).Error
	if err == nil {
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if err := BackfillStatementUsageFromLogs(); err != nil {
		return err
	}
	return DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&Option{
		Key: statementUsageBackfillKey, Value: "completed",
	}).Error
}

func validateStatementCustomerFields(title, address string) (string, string, error) {
	title = strings.TrimSpace(title)
	address = strings.TrimSpace(address)
	if len([]rune(title)) > 120 {
		return "", "", errors.New("对账抬头最多 120 个字符")
	}
	if len([]rune(address)) > 300 {
		return "", "", errors.New("联系地址最多 300 个字符")
	}
	return title, address, nil
}

func buildStatementSnapshot(user *User, statementNo string, start, end time.Time, input StatementGenerateInput) (StatementSnapshot, error) {
	if user == nil {
		return StatementSnapshot{}, errors.New("用户不存在")
	}
	title, address, err := validateStatementCustomerFields(input.BillingTitle, input.BillingAddress)
	if err != nil {
		return StatementSnapshot{}, err
	}
	if err := ReconcileStatementUsage(user.Id, start, end); err != nil {
		return StatementSnapshot{}, err
	}
	var usage []StatementUsageMonthly
	if err := DB.Where("user_id = ? AND month_start = ?", user.Id, start.Unix()).Order("model_name asc").Find(&usage).Error; err != nil {
		return StatementSnapshot{}, err
	}
	tokens := make([]StatementTokenItem, 0, len(usage))
	for _, row := range usage {
		tokens = append(tokens, StatementTokenItem{ModelName: row.ModelName, InputTokens: row.InputTokens, OutputTokens: row.OutputTokens, RecordCount: row.BillingCount})
	}

	var redemptions []Redemption
	if err := DB.Unscoped().Where("used_user_id = ? AND status = ? AND redeemed_time >= ? AND redeemed_time < ?", user.Id, common.RedemptionCodeStatusUsed, start.Unix(), end.Unix()).Order("redeemed_time asc, id asc").Find(&redemptions).Error; err != nil {
		return StatementSnapshot{}, err
	}
	redemptionItems := make([]StatementRedemptionItem, 0, len(redemptions))
	var redemptionTotal int64
	var unpriced int64
	for _, redemption := range redemptions {
		if redemption.CategoryPricedAt == 0 || redemption.CategoryID == 0 {
			unpriced++
			continue
		}
		redemptionItems = append(redemptionItems, StatementRedemptionItem{RecordID: redemption.Id, Category: redemption.CategoryNameSnapshot, AmountCents: redemption.CategoryPriceCents, RedeemedAt: redemption.RedeemedTime})
		redemptionTotal += redemption.CategoryPriceCents
	}

	var topUps []TopUp
	baseTopUp := DB.Where("user_id = ? AND status = ? AND amount > ? AND complete_time >= ? AND complete_time < ?", user.Id, common.TopUpStatusSuccess, 0, start.Unix(), end.Unix())
	if err := baseTopUp.Where("paid_currency = ? AND paid_amount_minor > ?", "CNY", 0).Order("complete_time asc, id asc").Find(&topUps).Error; err != nil {
		return StatementSnapshot{}, err
	}
	topUpItems := make([]StatementTopUpItem, 0, len(topUps))
	var topUpTotal int64
	for _, topUp := range topUps {
		topUpItems = append(topUpItems, StatementTopUpItem{RecordID: topUp.Id, AmountCents: topUp.PaidAmountMinor, CompletedAt: topUp.CompleteTime})
		topUpTotal += topUp.PaidAmountMinor
	}
	var unknownCurrency int64
	if err := DB.Model(&TopUp{}).
		Where("user_id = ? AND status = ? AND amount > ? AND complete_time >= ? AND complete_time < ?", user.Id, common.TopUpStatusSuccess, 0, start.Unix(), end.Unix()).
		Where("paid_currency = ?", "").Count(&unknownCurrency).Error; err != nil {
		return StatementSnapshot{}, err
	}

	settings := system_setting.GetStatementSettings()
	website := strings.TrimSpace(system_setting.ServerAddress)
	generatedAt := input.Now
	if generatedAt.IsZero() {
		generatedAt = time.Now()
	}
	return StatementSnapshot{
		StatementNo: statementNo,
		// The statement identity follows the same system name configured for the
		// site. Contact details remain separately configurable for accounting
		// enquiries, but the PDF title must not drift from the deployed site name.
		Issuer:      StatementIssuer{Name: common.SystemName, ContactEmail: settings.ContactEmail, Address: settings.IssuerAddress, Website: website},
		Recipient:   StatementRecipient{UserID: user.Id, Username: user.Username, Email: user.Email, BillingTitle: title, BillingAddress: address, UserSupplied: title != "" || address != ""},
		PeriodStart: start.Unix(), PeriodEnd: end.Unix(), Timezone: "Asia/Shanghai", Source: input.Source,
		IsFinal: input.Source == StatementSourceSystemMonthly, GeneratedAt: generatedAt.Unix(), GeneratedBy: input.GeneratedBy,
		Tokens: tokens, Redemptions: redemptionItems, TopUps: topUpItems,
		RedemptionTotalCents: redemptionTotal, TopUpTotalCents: topUpTotal, TotalCents: redemptionTotal + topUpTotal,
		Warnings:    StatementWarnings{UnpricedRedemptions: unpriced, UnknownCurrencyTopUps: unknownCurrency},
		Disclaimers: append([]string(nil), statementDisclaimers...), ComplianceVersion: StatementComplianceVersion,
	}, nil
}

func GenerateConsumptionStatement(input StatementGenerateInput) (*ConsumptionStatement, error) {
	if input.UserID <= 0 {
		return nil, errors.New("用户 ID 无效")
	}
	if input.Source != StatementSourceUserExport && input.Source != StatementSourceAdmin && input.Source != StatementSourceSystemMonthly {
		return nil, errors.New("账单来源无效")
	}
	start, end, _, err := StatementMonthBounds(input.Month, input.Now)
	if err != nil {
		return nil, err
	}
	if input.Source == StatementSourceSystemMonthly && !end.Equal(start.AddDate(0, 1, 0)) {
		return nil, errors.New("系统月结只能生成已结束月份")
	}
	var user User
	if err := DB.Select("id", "username", "email").Where("id = ?", input.UserID).First(&user).Error; err != nil {
		return nil, err
	}
	if input.Now.IsZero() {
		input.Now = time.Now()
	}
	random, err := common.GenerateRandomCharsKey(8)
	if err != nil {
		return nil, err
	}
	statementNo := fmt.Sprintf("ZD-%s-%d-%s", start.Format("200601"), user.Id, strings.ToUpper(random))
	snapshot, err := buildStatementSnapshot(&user, statementNo, start, end, input)
	if err != nil {
		return nil, err
	}
	snapshotBytes, err := common.Marshal(snapshot)
	if err != nil {
		return nil, err
	}
	hash := fmt.Sprintf("%x", sha256.Sum256(snapshotBytes))
	statement := &ConsumptionStatement{
		StatementNo: statementNo, UserID: user.Id, UsernameSnapshot: user.Username, EmailSnapshot: user.Email,
		MonthStart: start.Unix(), PeriodStart: start.Unix(), PeriodEnd: end.Unix(), Timezone: "Asia/Shanghai",
		Source: input.Source, IsFinal: input.Source == StatementSourceSystemMonthly, GeneratedBy: input.GeneratedBy,
		GeneratedAt: input.Now.Unix(), SnapshotJSON: string(snapshotBytes), ContentHash: hash,
		ComplianceVer: StatementComplianceVersion, RendererVer: StatementRendererVersion, Snapshot: snapshot,
	}
	if statement.IsFinal {
		key := fmt.Sprintf("%d:%d", user.Id, start.Unix())
		statement.FinalKey = &key
	}
	if err := DB.Create(statement).Error; err != nil {
		if statement.IsFinal {
			var existing ConsumptionStatement
			if lookupErr := DB.Where("final_key = ?", *statement.FinalKey).First(&existing).Error; lookupErr == nil {
				if decodeErr := common.UnmarshalJsonStr(existing.SnapshotJSON, &existing.Snapshot); decodeErr != nil {
					return nil, decodeErr
				}
				return &existing, nil
			}
		}
		return nil, err
	}
	return statement, nil
}

func GetConsumptionStatement(id int64) (*ConsumptionStatement, error) {
	var statement ConsumptionStatement
	if err := DB.Where("id = ?", id).First(&statement).Error; err != nil {
		return nil, err
	}
	if err := common.UnmarshalJsonStr(statement.SnapshotJSON, &statement.Snapshot); err != nil {
		return nil, err
	}
	return &statement, nil
}

func GetPreviousMonthSystemStatement(userID int, now time.Time) (*ConsumptionStatement, error) {
	if userID <= 0 {
		return nil, errors.New("用户 ID 无效")
	}
	if now.IsZero() {
		now = time.Now()
	}
	now = now.In(shanghaiLocation())
	currentMonthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, shanghaiLocation())
	previousMonthStart := currentMonthStart.AddDate(0, -1, 0)
	var statement ConsumptionStatement
	if err := DB.Where(
		"user_id = ? AND month_start = ? AND source = ? AND is_final = ?",
		userID, previousMonthStart.Unix(), StatementSourceSystemMonthly, true,
	).Order("generated_at desc, id desc").First(&statement).Error; err != nil {
		return nil, err
	}
	if err := common.UnmarshalJsonStr(statement.SnapshotJSON, &statement.Snapshot); err != nil {
		return nil, err
	}
	return &statement, nil
}

func CanAccessConsumptionStatement(statement *ConsumptionStatement, requesterID, role int, now time.Time) bool {
	if statement == nil {
		return false
	}
	if role >= common.RoleAdminUser {
		return true
	}
	if statement.UserID != requesterID {
		return false
	}
	currentStart, _, _, err := StatementMonthBounds("", now)
	if err != nil {
		return false
	}
	if statement.MonthStart == currentStart.Unix() {
		return true
	}
	previousStart := currentStart.AddDate(0, -1, 0)
	return statement.MonthStart == previousStart.Unix() &&
		statement.Source == StatementSourceSystemMonthly && statement.IsFinal
}

type StatementHistoryFilter struct {
	Month   string
	Keyword string
	Source  string
	Offset  int
	Limit   int
}

func ListConsumptionStatements(filter StatementHistoryFilter) ([]ConsumptionStatement, int64, error) {
	query := DB.Model(&ConsumptionStatement{})
	if strings.TrimSpace(filter.Month) != "" {
		start, _, _, err := StatementMonthBounds(filter.Month, time.Now())
		if err != nil {
			return nil, 0, err
		}
		query = query.Where("month_start = ?", start.Unix())
	}
	if strings.TrimSpace(filter.Source) != "" {
		query = query.Where("source = ?", strings.TrimSpace(filter.Source))
	}
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		pattern, err := sanitizeLikePattern(keyword)
		if err != nil {
			return nil, 0, err
		}
		query = query.Where("statement_no LIKE ? ESCAPE '!' OR username_snapshot LIKE ? ESCAPE '!' OR email_snapshot LIKE ? ESCAPE '!'", pattern, pattern, pattern)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var statements []ConsumptionStatement
	if err := query.Order("generated_at desc, id desc").Offset(max(filter.Offset, 0)).Limit(limit).Find(&statements).Error; err != nil {
		return nil, 0, err
	}
	for i := range statements {
		if err := common.UnmarshalJsonStr(statements[i].SnapshotJSON, &statements[i].Snapshot); err != nil {
			return nil, 0, err
		}
	}
	return statements, total, nil
}

type StatementMonthlySummary struct {
	UserID               int               `json:"user_id"`
	Username             string            `json:"username"`
	Email                string            `json:"email"`
	TokenModelCount      int               `json:"token_model_count"`
	InputTokens          int64             `json:"input_tokens"`
	OutputTokens         int64             `json:"output_tokens"`
	BillingCount         int64             `json:"billing_count"`
	RedemptionTotalCents int64             `json:"redemption_total_cents"`
	TopUpTotalCents      int64             `json:"topup_total_cents"`
	TotalCents           int64             `json:"total_cents"`
	Warnings             StatementWarnings `json:"warnings"`
	VersionCount         int64             `json:"version_count"`
	HasSystemFinal       bool              `json:"has_system_final"`
}

func ListStatementMonthlySummaries(month, keyword string, offset, limit int) ([]StatementMonthlySummary, int64, error) {
	start, end, _, err := StatementMonthBounds(month, time.Now())
	if err != nil {
		return nil, 0, err
	}
	query := DB.Model(&User{}).Select("id", "username", "email")
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		if id, parseErr := strconv.Atoi(keyword); parseErr == nil {
			query = query.Where("id = ?", id)
		} else {
			pattern, patternErr := sanitizeLikePattern(keyword)
			if patternErr != nil {
				return nil, 0, patternErr
			}
			query = query.Where("username LIKE ? ESCAPE '!' OR email LIKE ? ESCAPE '!'", pattern, pattern)
		}
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var users []User
	if err := query.Order("id desc").Offset(max(offset, 0)).Limit(limit).Find(&users).Error; err != nil {
		return nil, 0, err
	}
	result := make([]StatementMonthlySummary, 0, len(users))
	for i := range users {
		preview, buildErr := buildStatementSnapshot(&users[i], "PREVIEW", start, end, StatementGenerateInput{UserID: users[i].Id, Month: month, Source: StatementSourceAdmin, Now: time.Now()})
		if buildErr != nil {
			return nil, 0, buildErr
		}
		summary := StatementMonthlySummary{UserID: users[i].Id, Username: users[i].Username, Email: users[i].Email, TokenModelCount: len(preview.Tokens), RedemptionTotalCents: preview.RedemptionTotalCents, TopUpTotalCents: preview.TopUpTotalCents, TotalCents: preview.TotalCents, Warnings: preview.Warnings}
		for _, item := range preview.Tokens {
			summary.InputTokens += item.InputTokens
			summary.OutputTokens += item.OutputTokens
			summary.BillingCount += item.RecordCount
		}
		if err := DB.Model(&ConsumptionStatement{}).Where("user_id = ? AND month_start = ?", users[i].Id, start.Unix()).Count(&summary.VersionCount).Error; err != nil {
			return nil, 0, err
		}
		var finalCount int64
		if err := DB.Model(&ConsumptionStatement{}).Where("user_id = ? AND month_start = ? AND source = ?", users[i].Id, start.Unix(), StatementSourceSystemMonthly).Count(&finalCount).Error; err != nil {
			return nil, 0, err
		}
		summary.HasSystemFinal = finalCount > 0
		result = append(result, summary)
	}
	return result, total, nil
}

func GenerateClosedMonthStatements(month string, now time.Time) (int, int, error) {
	start, _, isCurrent, err := StatementMonthBounds(month, now)
	if err != nil {
		return 0, 0, err
	}
	if isCurrent {
		return 0, 0, errors.New("系统月结只能处理已结束月份")
	}
	if err := BackfillStatementUsageFromLogsOnce(); err != nil {
		return 0, 0, err
	}
	var ids []int
	if err := DB.Model(&User{}).Order("id asc").Pluck("id", &ids).Error; err != nil {
		return 0, 0, err
	}
	generated := 0
	skipped := 0
	for _, userID := range ids {
		key := fmt.Sprintf("%d:%d", userID, start.Unix())
		var count int64
		if err := DB.Model(&ConsumptionStatement{}).Where("final_key = ?", key).Count(&count).Error; err != nil {
			return generated, skipped, err
		}
		if count > 0 {
			skipped++
			continue
		}
		if _, err := GenerateConsumptionStatement(StatementGenerateInput{UserID: userID, Month: start.Format("2006-01"), Source: StatementSourceSystemMonthly, GeneratedBy: 0, Now: now}); err != nil {
			return generated, skipped, err
		}
		generated++
	}
	return generated, skipped, nil
}
