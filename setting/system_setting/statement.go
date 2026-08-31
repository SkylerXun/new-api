package system_setting

import (
	"strings"

	"github.com/QuantumNous/new-api/setting/config"
)

// StatementSettings contains public issuer information printed on consumption
// statements. Compliance copy is intentionally not configurable.
type StatementSettings struct {
	ContactEmail  string `json:"contact_email"`
	IssuerAddress string `json:"issuer_address"`
}

var defaultStatementSettings = StatementSettings{}

func init() {
	config.GlobalConfig.Register("statement_setting", &defaultStatementSettings)
}

func GetStatementSettings() StatementSettings {
	settings := defaultStatementSettings
	settings.ContactEmail = strings.TrimSpace(settings.ContactEmail)
	settings.IssuerAddress = strings.TrimSpace(settings.IssuerAddress)
	return settings
}
