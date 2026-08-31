package service

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/go-pdf/fpdf"
)

//go:embed assets/NotoSansSC-VF.ttf
var statementFont []byte

//go:embed assets/statement-logo.png
var statementLogo []byte

func statementTime(timestamp int64) string {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		location = time.FixedZone("Asia/Shanghai", 8*60*60)
	}
	return time.Unix(timestamp, 0).In(location).Format("2006-01-02 15:04:05")
}

func statementMoney(cents int64) string {
	return fmt.Sprintf("¥%d.%02d", cents/100, cents%100)
}

func statementFitText(pdf *fpdf.Fpdf, value string, maxWidth float64) string {
	value = strings.TrimSpace(value)
	if pdf.GetStringWidth(value) <= maxWidth {
		return value
	}
	runes := []rune(value)
	for len(runes) > 0 {
		candidate := string(runes) + "…"
		if pdf.GetStringWidth(candidate) <= maxWidth {
			return candidate
		}
		runes = runes[:len(runes)-1]
	}
	return "…"
}

func RenderConsumptionStatementPDF(statement *model.ConsumptionStatement) ([]byte, error) {
	logo, imageType := configuredStatementLogo()
	return renderConsumptionStatementPDF(statement, logo, imageType)
}

// RenderConsumptionStatementPDFWithLogo is intended for trusted, already
// downloaded logo bytes (for example a logo cached by an administrator-side
// asset importer). It never fetches a URL and therefore keeps PDF generation
// deterministic and free of SSRF-prone network access.
func RenderConsumptionStatementPDFWithLogo(statement *model.ConsumptionStatement, logo []byte) ([]byte, error) {
	if len(logo) == 0 {
		return RenderConsumptionStatementPDF(statement)
	}
	imageType := statementImageType(logo)
	if imageType == "" {
		logo, imageType = statementLogo, "PNG"
	}
	return renderConsumptionStatementPDF(statement, logo, imageType)
}

// configuredStatementLogo uses the same logo configured in System Information.
// It is resolved once per export, with a small timeout and size cap so a slow
// or unavailable image cannot hold a PDF request indefinitely. The embedded
// logo remains the fallback for empty, invalid, or non-image values.
func configuredStatementLogo() ([]byte, string) {
	logoURL := strings.TrimSpace(common.Logo)
	if logoURL == "" || !(strings.HasPrefix(logoURL, "http://") || strings.HasPrefix(logoURL, "https://")) {
		return statementLogo, "PNG"
	}
	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Get(logoURL)
	if err != nil {
		return statementLogo, "PNG"
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return statementLogo, "PNG"
	}
	limited := io.LimitReader(response.Body, 4<<20)
	data, err := io.ReadAll(limited)
	if err != nil || len(data) == 0 {
		return statementLogo, "PNG"
	}
	imageType := statementImageType(data)
	if imageType == "" {
		return statementLogo, "PNG"
	}
	return data, imageType
}

func statementImageType(data []byte) string {
	contentType := http.DetectContentType(data)
	switch {
	case strings.HasPrefix(contentType, "image/png"):
		return "PNG"
	case strings.HasPrefix(contentType, "image/jpeg"):
		return "JPG"
	default:
		return ""
	}
}

func renderConsumptionStatementPDF(statement *model.ConsumptionStatement, logo []byte, logoType string) ([]byte, error) {
	if statement == nil {
		return nil, fmt.Errorf("statement is required")
	}
	snapshot := statement.Snapshot
	if snapshot.StatementNo == "" {
		return nil, fmt.Errorf("statement snapshot is empty")
	}
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(14, 16, 14)
	pdf.SetAutoPageBreak(true, 22)
	pdf.AddUTF8FontFromBytes("NotoSC", "", statementFont)
	if pdf.Error() != nil {
		return nil, pdf.Error()
	}

	pdf.SetHeaderFunc(func() {
		pdf.SetAlpha(0.075, "Normal")
		pdf.SetTextColor(120, 120, 120)
		pdf.SetFont("NotoSC", "", 30)
		pdf.TransformBegin()
		pdf.TransformRotate(28, 105, 145)
		pdf.Text(43, 140, "非税务发票")
		pdf.Text(48, 160, "仅供消费对账")
		pdf.TransformEnd()
		pdf.SetAlpha(1, "Normal")
		pdf.SetTextColor(32, 32, 32)

		options := fpdf.ImageOptions{ImageType: logoType, ReadDpi: true}
		pdf.RegisterImageOptionsReader("statement-logo", options, bytes.NewReader(logo))
		pdf.ImageOptions("statement-logo", 14, 11, 12, 12, false, options, 0, "")
		pdf.SetFont("NotoSC", "", 13)
		pdf.SetXY(30, 12)
		pdf.CellFormat(100, 7, snapshot.Issuer.Name, "", 0, "L", false, 0, "")
		pdf.SetDrawColor(25, 25, 25)
		pdf.SetLineWidth(0.7)
		pdf.Line(14, 27, 196, 27)
	})

	pdf.SetFooterFunc(func() {
		pdf.SetY(-16)
		pdf.SetDrawColor(190, 190, 190)
		pdf.SetLineWidth(0.2)
		pdf.Line(14, pdf.GetY(), 196, pdf.GetY())
		pdf.SetY(-13)
		pdf.SetFont("NotoSC", "", 7)
		pdf.SetTextColor(100, 100, 100)
		pdf.CellFormat(125, 5, "内容哈希 SHA-256: "+statement.ContentHash, "", 0, "L", false, 0, "")
		pdf.CellFormat(57, 5, fmt.Sprintf("第 %d 页 / {nb}", pdf.PageNo()), "", 0, "R", false, 0, "")
	})
	pdf.AliasNbPages("")
	pdf.AddPage()
	pdf.SetY(34)
	pdf.SetTextColor(24, 24, 24)
	pdf.SetFont("NotoSC", "", 24)
	pdf.CellFormat(182, 12, "消费对账单", "", 1, "L", false, 0, "")
	pdf.Ln(6)

	pdf.SetTextColor(45, 45, 45)
	pdf.SetFont("NotoSC", "", 9)
	leftX, middleX, rightX := 14.0, 76.0, 139.0
	blockY := pdf.GetY()
	pdf.SetXY(leftX, blockY)
	pdf.SetFont("NotoSC", "", 10)
	pdf.CellFormat(56, 6, "出具方", "", 1, "L", false, 0, "")
	pdf.SetFont("NotoSC", "", 8.5)
	pdf.SetX(leftX)
	pdf.MultiCell(56, 5, snapshot.Issuer.Name+"\n"+snapshot.Issuer.ContactEmail+"\n"+snapshot.Issuer.Address+"\n"+snapshot.Issuer.Website, "", "L", false)

	pdf.SetXY(middleX, blockY)
	pdf.SetFont("NotoSC", "", 10)
	pdf.CellFormat(57, 6, "对账对象", "", 1, "L", false, 0, "")
	pdf.SetFont("NotoSC", "", 8.5)
	pdf.SetX(middleX)
	recipient := fmt.Sprintf("用户ID：%d\n账号：%s\n邮箱：%s", snapshot.Recipient.UserID, snapshot.Recipient.Username, snapshot.Recipient.Email)
	if snapshot.Recipient.BillingTitle != "" {
		recipient += "\n对账抬头：" + snapshot.Recipient.BillingTitle
	}
	if snapshot.Recipient.BillingAddress != "" {
		recipient += "\n联系地址：" + snapshot.Recipient.BillingAddress
	}
	pdf.MultiCell(57, 5, recipient, "", "L", false)

	pdf.SetXY(rightX, blockY)
	pdf.SetFont("NotoSC", "", 10)
	pdf.CellFormat(57, 6, "单据信息", "", 1, "L", false, 0, "")
	pdf.SetFont("NotoSC", "", 8.5)
	pdf.SetX(rightX)
	periodLabel := statementTime(snapshot.PeriodStart) + "\n至 " + statementTime(snapshot.PeriodEnd)
	status := "阶段性快照"
	if snapshot.IsFinal {
		status = "系统月结（最终版）"
	}
	pdf.MultiCell(57, 5, "账单号："+snapshot.StatementNo+"\n账期："+periodLabel+"\n生成时间："+statementTime(snapshot.GeneratedAt)+"\n版本："+status, "", "L", false)
	pdf.SetY(blockY + 46)
	if snapshot.Recipient.UserSupplied {
		pdf.SetFont("NotoSC", "", 7.5)
		pdf.SetTextColor(120, 80, 0)
		pdf.MultiCell(182, 5, "提示：对账抬头和联系地址由用户自行提供，仅用于对账识别，不构成发票抬头或税务信息。", "1", "L", false)
		pdf.Ln(3)
	}

	sectionTitle := func(title string) {
		if pdf.GetY() > 252 {
			pdf.AddPage()
			pdf.SetY(34)
		}
		pdf.SetTextColor(24, 24, 24)
		pdf.SetFont("NotoSC", "", 12)
		pdf.CellFormat(182, 8, title, "B", 1, "L", false, 0, "")
		pdf.Ln(2)
	}
	tableHeader := func(columns []string, widths []float64) {
		pdf.SetFillColor(238, 241, 245)
		pdf.SetTextColor(45, 45, 45)
		pdf.SetFont("NotoSC", "", 8.5)
		for index, column := range columns {
			pdf.CellFormat(widths[index], 7, column, "1", 0, "C", true, 0, "")
		}
		pdf.Ln(-1)
	}
	row := func(values []string, columns []string, widths []float64, alignments []string) {
		if pdf.GetY() > 262 {
			pdf.AddPage()
			pdf.SetY(34)
			tableHeader(columns, widths)
		}
		pdf.SetFont("NotoSC", "", 8.2)
		pdf.SetTextColor(45, 45, 45)
		for index, value := range values {
			pdf.CellFormat(widths[index], 7, value, "1", 0, alignments[index], false, 0, "")
		}
		pdf.Ln(-1)
	}

	sectionTitle("模型 Token 汇总")
	tokenWidths := []float64{80, 34, 34, 34}
	tokenColumns := []string{"模型", "输入 Token", "输出 Token", "计费记录数"}
	tableHeader(tokenColumns, tokenWidths)
	if len(snapshot.Tokens) == 0 {
		row([]string{"本账期无 Token 消费记录", "0", "0", "0"}, tokenColumns, tokenWidths, []string{"L", "R", "R", "R"})
	} else {
		for _, item := range snapshot.Tokens {
			row([]string{statementFitText(pdf, item.ModelName, tokenWidths[0]-2), fmt.Sprintf("%d", item.InputTokens), fmt.Sprintf("%d", item.OutputTokens), fmt.Sprintf("%d", item.RecordCount)}, tokenColumns, tokenWidths, []string{"L", "R", "R", "R"})
		}
	}
	pdf.Ln(4)

	sectionTitle("兑换码类别记账明细")
	redemptionWidths := []float64{26, 72, 48, 36}
	redemptionColumns := []string{"记录ID", "类别", "兑换时间", "记账金额"}
	tableHeader(redemptionColumns, redemptionWidths)
	if len(snapshot.Redemptions) == 0 {
		row([]string{"-", "本账期无已计价兑换记录", "-", "¥0.00"}, redemptionColumns, redemptionWidths, []string{"C", "L", "C", "R"})
	} else {
		for _, item := range snapshot.Redemptions {
			row([]string{fmt.Sprintf("%d", item.RecordID), statementFitText(pdf, item.Category, redemptionWidths[1]-2), statementTime(item.RedeemedAt), statementMoney(item.AmountCents)}, redemptionColumns, redemptionWidths, []string{"C", "L", "C", "R"})
		}
	}
	pdf.Ln(4)

	sectionTitle("钱包人民币充值明细")
	topUpWidths := []float64{42, 92, 48}
	topUpColumns := []string{"记录ID", "站内确认完成时间", "人民币金额"}
	tableHeader(topUpColumns, topUpWidths)
	if len(snapshot.TopUps) == 0 {
		row([]string{"-", "本账期无符合条件的人民币充值", "¥0.00"}, topUpColumns, topUpWidths, []string{"C", "L", "R"})
	} else {
	for _, item := range snapshot.TopUps {
			row([]string{fmt.Sprintf("%d", item.RecordID), statementTime(item.CompletedAt), statementMoney(item.AmountCents)}, topUpColumns, topUpWidths, []string{"C", "L", "R"})
		}
	}
	pdf.Ln(5)

	sectionTitle("订阅购买明细")
	subscriptionWidths := []float64{26, 92, 48}
	subscriptionColumns := []string{"记录ID", "套餐", "人民币金额"}
	tableHeader(subscriptionColumns, subscriptionWidths)
	if len(snapshot.Subscriptions) == 0 {
		row([]string{"-", "本账期无订阅购买记录", "¥0.00"}, subscriptionColumns, subscriptionWidths, []string{"C", "L", "R"})
	} else {
		for _, item := range snapshot.Subscriptions {
			row([]string{fmt.Sprintf("%d", item.RecordID), statementFitText(pdf, item.PlanTitle, subscriptionWidths[1]-2), statementMoney(item.AmountCents)}, subscriptionColumns, subscriptionWidths, []string{"C", "L", "R"})
		}
	}
	pdf.Ln(5)

	if pdf.GetY() > 235 {
		pdf.AddPage()
		pdf.SetY(34)
	}
	pdf.SetX(98)
	pdf.SetFont("NotoSC", "", 10)
	pdf.CellFormat(62, 8, "兑换码类别记账金额", "B", 0, "L", false, 0, "")
	pdf.CellFormat(36, 8, statementMoney(snapshot.RedemptionTotalCents), "B", 1, "R", false, 0, "")
	pdf.SetX(98)
	pdf.CellFormat(62, 8, "钱包人民币充值金额", "B", 0, "L", false, 0, "")
	pdf.CellFormat(36, 8, statementMoney(snapshot.TopUpTotalCents), "B", 1, "R", false, 0, "")
	pdf.SetX(98)
	pdf.SetFont("NotoSC", "", 10)
	pdf.CellFormat(62, 8, "订阅购买金额", "B", 0, "L", false, 0, "")
	pdf.CellFormat(36, 8, statementMoney(snapshot.SubscriptionTotalCents), "B", 1, "R", false, 0, "")
	pdf.SetX(98)
	pdf.SetFont("NotoSC", "", 12)
	pdf.CellFormat(62, 9, "人民币记账金额合计", "B", 0, "L", false, 0, "")
	pdf.CellFormat(36, 9, statementMoney(snapshot.TotalCents), "B", 1, "R", false, 0, "")
	pdf.Ln(5)

	if snapshot.Warnings.UnpricedRedemptions > 0 || snapshot.Warnings.UnknownCurrencyTopUps > 0 {
		pdf.SetTextColor(150, 75, 0)
		pdf.SetFont("NotoSC", "", 8.5)
		warning := fmt.Sprintf("未计入警告：待补价兑换码 %d 条；无法确认币种的历史充值订单 %d 条。以上记录未静默计入人民币合计。", snapshot.Warnings.UnpricedRedemptions, snapshot.Warnings.UnknownCurrencyTopUps)
		pdf.MultiCell(182, 6, warning, "1", "L", false)
		pdf.Ln(4)
	}

	sectionTitle("合规说明")
	pdf.SetTextColor(60, 60, 60)
	pdf.SetFont("NotoSC", "", 8.5)
	for index, disclaimer := range snapshot.Disclaimers {
		if pdf.GetY() > 258 {
			pdf.AddPage()
			pdf.SetY(34)
		}
		pdf.MultiCell(182, 5.5, fmt.Sprintf("%d. %s", index+1, disclaimer), "", "L", false)
	}
	pdf.Ln(2)
	pdf.SetTextColor(110, 110, 110)
	pdf.SetFont("NotoSC", "", 7.5)
	pdf.MultiCell(182, 5, "本 PDF 由服务端固定模板生成，不包含税率、税额、发票代码、发票号码、发票章或已付款证明。完整性校验值见每页页脚。", "", "L", false)

	var output bytes.Buffer
	if err := pdf.Output(&output); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}
