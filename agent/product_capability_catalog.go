package agent

import "strings"

type productFieldSpec struct {
	Key           string
	LabelZH       string
	LabelEN       string
	DescriptionZH string
	DescriptionEN string
	Sensitive     bool
}

type exchangeProductSpec struct {
	Type                string
	DisplayName         string
	CredentialFields    []string
	CredentialSummaryZH string
	CredentialSummaryEN string
	RegistrationGuideZH string
	RegistrationGuideEN string
}

var productFieldCatalog = map[string]productFieldSpec{
	"name":                        {Key: "name", LabelZH: "名称", LabelEN: "name", DescriptionZH: "页面上显示的名字，方便以后识别。", DescriptionEN: "Display name used to identify this item later."},
	"provider":                    {Key: "provider", LabelZH: "模型提供商", LabelEN: "model provider", DescriptionZH: "选择调用哪个模型供应商。", DescriptionEN: "Which model provider to call."},
	"api_key":                     {Key: "api_key", LabelZH: "API Key", LabelEN: "API key", DescriptionZH: "供应商或交易所生成的访问密钥。", DescriptionEN: "Access key from the provider or exchange.", Sensitive: true},
	"custom_model_name":           {Key: "custom_model_name", LabelZH: "模型名称", LabelEN: "model name", DescriptionZH: "实际发送给供应商的模型名；不填时使用默认模型。", DescriptionEN: "Upstream model name; defaults when omitted."},
	"custom_api_url":              {Key: "custom_api_url", LabelZH: "接口地址", LabelEN: "API URL", DescriptionZH: "自定义模型 API 地址；大多数新手可以留空。", DescriptionEN: "Custom API endpoint; most users can leave it empty."},
	"enabled":                     {Key: "enabled", LabelZH: "启用状态", LabelEN: "enabled", DescriptionZH: "是否启用这条配置。", DescriptionEN: "Whether this config is enabled."},
	"exchange_type":               {Key: "exchange_type", LabelZH: "交易所类型", LabelEN: "exchange type", DescriptionZH: "选择 Binance、OKX、Aster 等交易所。", DescriptionEN: "Choose Binance, OKX, Aster, and so on."},
	"account_name":                {Key: "account_name", LabelZH: "账户名", LabelEN: "account display name", DescriptionZH: "只是给你自己看的显示名，可以随便取。", DescriptionEN: "A local display name; you can choose it freely."},
	"secret_key":                  {Key: "secret_key", LabelZH: "Secret", LabelEN: "secret", DescriptionZH: "交易所 API Secret。", DescriptionEN: "Exchange API secret.", Sensitive: true},
	"passphrase":                  {Key: "passphrase", LabelZH: "Passphrase", LabelEN: "passphrase", DescriptionZH: "OKX、Bitget、KuCoin 等交易所需要。", DescriptionEN: "Required by exchanges such as OKX, Bitget, and KuCoin.", Sensitive: true},
	"testnet":                     {Key: "testnet", LabelZH: "测试网", LabelEN: "testnet", DescriptionZH: "是否连接测试网。", DescriptionEN: "Whether to connect to testnet."},
	"hyperliquid_wallet_addr":     {Key: "hyperliquid_wallet_addr", LabelZH: "Hyperliquid 钱包地址", LabelEN: "Hyperliquid wallet address", DescriptionZH: "Hyperliquid 主钱包地址。", DescriptionEN: "Hyperliquid main wallet address.", Sensitive: true},
	"hyperliquid_unified_account": {Key: "hyperliquid_unified_account", LabelZH: "Hyperliquid Unified Account", LabelEN: "Hyperliquid unified account", DescriptionZH: "Hyperliquid 统一账户模式开关。", DescriptionEN: "Hyperliquid unified account mode."},
	"aster_user":                  {Key: "aster_user", LabelZH: "Aster 主钱包地址", LabelEN: "Aster main wallet address", DescriptionZH: "Aster 资金主钱包地址，和账户显示名是两个字段。", DescriptionEN: "Aster main wallet address; separate from the local display name.", Sensitive: true},
	"aster_signer":                {Key: "aster_signer", LabelZH: "Aster API Pro 代理钱包地址", LabelEN: "Aster API Pro wallet address", DescriptionZH: "Aster API Pro 代理钱包地址。", DescriptionEN: "Aster API Pro wallet address.", Sensitive: true},
	"aster_private_key":           {Key: "aster_private_key", LabelZH: "Aster API Pro 代理钱包私钥", LabelEN: "Aster API Pro wallet private key", DescriptionZH: "Aster API Pro 代理钱包私钥。", DescriptionEN: "Aster API Pro wallet private key.", Sensitive: true},
	"lighter_wallet_addr":         {Key: "lighter_wallet_addr", LabelZH: "Lighter 钱包地址", LabelEN: "Lighter wallet address", DescriptionZH: "Lighter 钱包地址。", DescriptionEN: "Lighter wallet address.", Sensitive: true},
	"lighter_private_key":         {Key: "lighter_private_key", LabelZH: "Lighter 私钥", LabelEN: "Lighter private key", DescriptionZH: "Lighter 私钥。", DescriptionEN: "Lighter private key.", Sensitive: true},
	"lighter_api_key_private_key": {Key: "lighter_api_key_private_key", LabelZH: "Lighter API Key 私钥", LabelEN: "Lighter API key private key", DescriptionZH: "Lighter API Key 对应的私钥。", DescriptionEN: "Private key for the Lighter API key.", Sensitive: true},
	"lighter_api_key_index":       {Key: "lighter_api_key_index", LabelZH: "Lighter API Key Index", LabelEN: "Lighter API key index", DescriptionZH: "Lighter API Key 的索引。", DescriptionEN: "Index of the Lighter API key."},
	"scan_interval_minutes":       {Key: "scan_interval_minutes", LabelZH: "扫描间隔", LabelEN: "scan interval", DescriptionZH: "交易员每隔多久检查一次行情和策略。", DescriptionEN: "How often the trader checks market and strategy state."},
	"is_cross_margin":             {Key: "is_cross_margin", LabelZH: "全仓模式", LabelEN: "cross margin", DescriptionZH: "是否使用全仓模式。", DescriptionEN: "Whether cross margin is enabled."},
	"show_in_competition":         {Key: "show_in_competition", LabelZH: "竞技场显示", LabelEN: "show in competition", DescriptionZH: "是否显示在比赛/竞技场页面。", DescriptionEN: "Whether to show this trader in competition views."},
	"exchange_name":               {Key: "exchange_name", LabelZH: "交易所配置", LabelEN: "exchange config", DescriptionZH: "选择一个已经保存的交易所账户。", DescriptionEN: "Select an existing exchange account."},
	"model_name":                  {Key: "model_name", LabelZH: "模型配置", LabelEN: "model config", DescriptionZH: "选择一个已经保存并可用的模型。", DescriptionEN: "Select an existing enabled model config."},
	"strategy_name":               {Key: "strategy_name", LabelZH: "策略模板", LabelEN: "strategy template", DescriptionZH: "选择一个已经保存的策略。", DescriptionEN: "Select an existing strategy template."},
}

var exchangeProductCatalog = []exchangeProductSpec{
	{Type: "binance", DisplayName: "Binance", CredentialFields: []string{"api_key", "secret_key"}, CredentialSummaryZH: "需要 API Key 和 Secret。", CredentialSummaryEN: "Requires API key and secret."},
	{Type: "bybit", DisplayName: "Bybit", CredentialFields: []string{"api_key", "secret_key"}, CredentialSummaryZH: "需要 API Key 和 Secret。", CredentialSummaryEN: "Requires API key and secret."},
	{Type: "gate", DisplayName: "Gate", CredentialFields: []string{"api_key", "secret_key"}, CredentialSummaryZH: "需要 API Key 和 Secret。", CredentialSummaryEN: "Requires API key and secret."},
	{Type: "indodax", DisplayName: "Indodax", CredentialFields: []string{"api_key", "secret_key"}, CredentialSummaryZH: "需要 API Key 和 Secret。", CredentialSummaryEN: "Requires API key and secret."},
	{Type: "okx", DisplayName: "OKX", CredentialFields: []string{"api_key", "secret_key", "passphrase"}, CredentialSummaryZH: "需要 API Key、Secret 和 Passphrase。", CredentialSummaryEN: "Requires API key, secret, and passphrase."},
	{Type: "bitget", DisplayName: "Bitget", CredentialFields: []string{"api_key", "secret_key", "passphrase"}, CredentialSummaryZH: "需要 API Key、Secret 和 Passphrase。", CredentialSummaryEN: "Requires API key, secret, and passphrase."},
	{Type: "kucoin", DisplayName: "KuCoin", CredentialFields: []string{"api_key", "secret_key", "passphrase"}, CredentialSummaryZH: "需要 API Key、Secret 和 Passphrase。", CredentialSummaryEN: "Requires API key, secret, and passphrase."},
	{Type: "hyperliquid", DisplayName: "Hyperliquid", CredentialFields: []string{"api_key", "hyperliquid_wallet_addr"}, CredentialSummaryZH: "需要 API Key 和 Hyperliquid 钱包地址。", CredentialSummaryEN: "Requires API key and Hyperliquid wallet address."},
	{Type: "aster", DisplayName: "Aster", CredentialFields: []string{"aster_user", "aster_signer", "aster_private_key"}, CredentialSummaryZH: "需要主钱包地址、API Pro 代理钱包地址、API Pro 代理钱包私钥。", CredentialSummaryEN: "Requires main wallet address, API Pro wallet address, and API Pro wallet private key."},
	{Type: "lighter", DisplayName: "Lighter", CredentialFields: []string{"lighter_wallet_addr", "lighter_api_key_private_key"}, CredentialSummaryZH: "需要 Lighter 钱包地址和 Lighter API Key 私钥；API Key Index 可选。", CredentialSummaryEN: "Requires Lighter wallet address and Lighter API key private key; API key index is optional."},
}

func productFieldLabel(key, lang string) string {
	if spec, ok := productFieldCatalog[strings.TrimSpace(key)]; ok {
		if lang == "zh" && spec.LabelZH != "" {
			return spec.LabelZH
		}
		if spec.LabelEN != "" {
			return spec.LabelEN
		}
	}
	return strings.TrimSpace(key)
}

func productFieldDescription(key, lang string) string {
	if spec, ok := productFieldCatalog[strings.TrimSpace(key)]; ok {
		if lang == "zh" && spec.DescriptionZH != "" {
			return spec.DescriptionZH
		}
		if spec.DescriptionEN != "" {
			return spec.DescriptionEN
		}
	}
	return productFieldLabel(key, lang)
}

func exchangeProductSpecByType(exchangeType string) (exchangeProductSpec, bool) {
	exchangeType = strings.ToLower(strings.TrimSpace(exchangeType))
	for _, spec := range exchangeProductCatalog {
		if spec.Type == exchangeType {
			return spec, true
		}
	}
	return exchangeProductSpec{}, false
}

func supportedExchangeTypeLabels(lang string) string {
	labels := make([]string, 0, len(exchangeProductCatalog))
	for _, spec := range exchangeProductCatalog {
		labels = append(labels, spec.DisplayName)
	}
	return strings.Join(labels, " / ")
}

func exchangeCredentialFieldsForType(exchangeType string) []string {
	spec, ok := exchangeProductSpecByType(exchangeType)
	if !ok {
		return nil
	}
	return append([]string(nil), spec.CredentialFields...)
}

func fieldLabels(lang string, fields []string) []string {
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field != "" {
			out = append(out, productFieldLabel(field, lang))
		}
	}
	return out
}

func exchangeCreateRequiredFieldKeys(session skillSession) []string {
	exchangeType := strings.ToLower(strings.TrimSpace(fieldValue(session, "exchange_type")))
	keys := []string{"exchange_type", "account_name"}
	if exchangeType == "" {
		return keys
	}
	if credentialFields := exchangeCredentialFieldsForType(exchangeType); len(credentialFields) > 0 {
		return append(keys, credentialFields...)
	}
	return keys
}

func exchangeCreateMissingFieldKeys(session skillSession) []string {
	exchangeType := strings.ToLower(strings.TrimSpace(fieldValue(session, "exchange_type")))
	if exchangeType != "" {
		if _, ok := exchangeProductSpecByType(exchangeType); !ok {
			return []string{"exchange_type"}
		}
	}
	required := exchangeCreateRequiredFieldKeys(session)
	missing := make([]string, 0, len(required))
	for _, key := range required {
		if strings.TrimSpace(fieldValue(session, key)) == "" {
			missing = append(missing, key)
		}
	}
	return missing
}
