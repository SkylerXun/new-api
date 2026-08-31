package service

import (
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderConsumptionStatementPDFProducesReadableHeader(t *testing.T) {
	statement := &model.ConsumptionStatement{
		StatementNo: "ZD-202608-1-TEST",
		ContentHash: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		Snapshot: model.StatementSnapshot{
			StatementNo: "ZD-202608-1-TEST",
			Issuer:      model.StatementIssuer{Name: "测试站点", ContactEmail: "contact@example.com", Address: "广东省深圳市", Website: "https://example.com"},
			Recipient:   model.StatementRecipient{UserID: 1, Username: "tester", Email: "tester@example.com", BillingTitle: "广东研发部", BillingAddress: "广东省深圳市", UserSupplied: true},
			PeriodStart: 1785513600, PeriodEnd: 1788143716, Timezone: "Asia/Shanghai", GeneratedAt: 1788143716,
			Tokens:               []model.StatementTokenItem{{ModelName: "gpt-test", InputTokens: 1200, OutputTokens: 300, RecordCount: 4}},
			Redemptions:          []model.StatementRedemptionItem{{RecordID: 8, Category: "会员兑换", AmountCents: 1999, RedeemedAt: 1786000000}},
			TopUps:               []model.StatementTopUpItem{{RecordID: 9, AmountCents: 888, CompletedAt: 1786100000}},
			RedemptionTotalCents: 1999, TopUpTotalCents: 888, TotalCents: 2887,
			Disclaimers: []string{"本单据不是税务发票、财政票据、付款收据或完税证明。"},
		},
	}
	pdf, err := RenderConsumptionStatementPDF(statement)
	require.NoError(t, err)
	require.Greater(t, len(pdf), 10_000)
	assert.Equal(t, "%PDF", string(pdf[:4]))
}
