package agent

import (
	"encoding/json"
	"log/slog"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"nofx/store"
)

func TestToolManageModelConfigCreateRequiresCredential(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "visibility.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	a := New(nil, st, DefaultConfig(), slog.Default())

	resp := a.toolManageModelConfig("default", `{"action":"create","provider":"deepseek"}`)
	if !strings.Contains(resp, `"error":"api_key is required for create"`) {
		t.Fatalf("expected missing api_key error, got: %s", resp)
	}
}

func TestToolManageModelConfigCreateDefaultsToEnabledLikeManualPage(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "model-create-enabled.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	a := New(nil, st, DefaultConfig(), slog.Default())

	resp := a.toolManageModelConfig("default", `{"action":"create","provider":"qwen","name":"qwen","api_key":"sk-test-qwen-123456","custom_model_name":"qwen3-max"}`)
	if strings.Contains(resp, `"error"`) {
		t.Fatalf("expected create to succeed, got: %s", resp)
	}

	model, err := st.AIModel().Get("default", "default_qwen")
	if err != nil {
		t.Fatalf("load created model: %v", err)
	}
	if !model.Enabled {
		t.Fatalf("expected agent-created model to default to enabled so it matches manual creation")
	}
}

func TestToolManageModelConfigCreateReusesExistingProviderRecord(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "model-create-upsert.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	a := New(nil, st, DefaultConfig(), slog.Default())

	if err := st.AIModel().UpdateWithName("default", "default_qwen", "qwen1", false, "sk-old-qwen-123456", "", "qwen3-max"); err != nil {
		t.Fatalf("seed existing qwen model: %v", err)
	}

	resp := a.toolManageModelConfig("default", `{"action":"create","provider":"qwen","name":"Qwen","api_key":"sk-new-qwen-123456","custom_model_name":"qwen3-max"}`)
	if strings.Contains(resp, `"error"`) {
		t.Fatalf("expected create to reuse existing qwen config instead of failing, got: %s", resp)
	}

	models, err := st.AIModel().List("default")
	if err != nil {
		t.Fatalf("list models: %v", err)
	}
	qwenCount := 0
	for _, model := range models {
		if model != nil && model.Provider == "qwen" {
			qwenCount++
			if model.ID != "default_qwen" {
				t.Fatalf("expected existing qwen record to be reused, got model id %q", model.ID)
			}
			if model.Name != "Qwen" {
				t.Fatalf("expected reused qwen record to be renamed, got %q", model.Name)
			}
			if !model.Enabled {
				t.Fatalf("expected reused qwen record to be enabled after agent create")
			}
		}
	}
	if qwenCount != 1 {
		t.Fatalf("expected exactly one qwen record after reuse, got %d", qwenCount)
	}
}

func TestToolManageExchangeConfigCreateDefaultsToEnabledLikeManualPage(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "exchange-create-enabled.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	a := New(nil, st, DefaultConfig(), slog.Default())

	resp := a.toolManageExchangeConfig("default", `{"action":"create","exchange_type":"binance","account_name":"Binance Main","api_key":"api-test-123456","secret_key":"secret-test-123456"}`)
	if strings.Contains(resp, `"error"`) {
		t.Fatalf("expected create to succeed, got: %s", resp)
	}

	exchanges, err := st.Exchange().List("default")
	if err != nil {
		t.Fatalf("list exchanges: %v", err)
	}
	if len(exchanges) != 1 || exchanges[0] == nil {
		t.Fatalf("expected one created exchange, got %#v", exchanges)
	}
	if !exchanges[0].Enabled {
		t.Fatalf("expected agent-created exchange to default to enabled so it matches manual creation")
	}
}

func TestToolManageExchangeConfigCreateRejectsIncompleteDraft(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "exchange-create-incomplete.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	a := New(nil, st, DefaultConfig(), slog.Default())

	resp := a.toolManageExchangeConfig("default", `{"action":"create","exchange_type":"okx","account_name":"OKX Main","api_key":"api-test-123456","secret_key":"secret-test-123456"}`)
	if !strings.Contains(resp, `"error"`) || !strings.Contains(resp, "passphrase") {
		t.Fatalf("expected incomplete create to be rejected with missing passphrase, got: %s", resp)
	}

	exchanges, err := st.Exchange().List("default")
	if err != nil {
		t.Fatalf("list exchanges: %v", err)
	}
	if len(exchanges) != 0 {
		t.Fatalf("expected incomplete exchange not to be persisted, got %#v", exchanges)
	}
}

func TestToolGetModelConfigsHidesIncompleteRows(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "visibility-list.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	a := New(nil, st, DefaultConfig(), slog.Default())

	if err := st.AIModel().UpdateWithName("default", "default_openai", "OpenAI", false, "", "", ""); err != nil {
		t.Fatalf("seed incomplete model: %v", err)
	}
	if err := st.AIModel().UpdateWithName("default", "default_deepseek", "DeepSeek", false, "sk-test-12345", "", "deepseek-chat"); err != nil {
		t.Fatalf("seed configured model: %v", err)
	}

	resp := a.toolGetModelConfigs("default")
	if strings.Contains(resp, `"id":"default_openai"`) {
		t.Fatalf("incomplete model should be hidden from tool query: %s", resp)
	}
	if !strings.Contains(resp, `"id":"default_deepseek"`) {
		t.Fatalf("configured model should remain visible: %s", resp)
	}
}

func TestToolManageStrategyUpdateRejectsOutOfRangeLeverageBeforeSave(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "strategy-risk-guard.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	a := New(nil, st, DefaultConfig(), slog.Default())

	cfg := store.GetDefaultStrategyConfig("zh")
	rawCfg, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal strategy config: %v", err)
	}
	strategy := &store.Strategy{
		ID:            "strategy-risk-guard",
		UserID:        "default",
		Name:          "AI500稳重策略",
		Description:   "test",
		IsPublic:      false,
		ConfigVisible: true,
		Config:        string(rawCfg),
	}
	if err := st.Strategy().Create(strategy); err != nil {
		t.Fatalf("create strategy: %v", err)
	}

	resp := a.toolManageStrategy("default", `{"action":"update","strategy_id":"strategy-risk-guard","config":{"risk_control":{"btc_eth_max_leverage":100,"altcoin_max_leverage":100}}}`)
	if !strings.Contains(resp, `不会按你给的原值直接保存`) {
		t.Fatalf("expected out-of-range leverage update to be rejected before save, got: %s", resp)
	}

	updated, err := st.Strategy().Get("default", strategy.ID)
	if err != nil {
		t.Fatalf("reload strategy: %v", err)
	}
	parsed, err := updated.ParseConfig()
	if err != nil {
		t.Fatalf("parse updated strategy config: %v", err)
	}
	if parsed.RiskControl.BTCETHMaxLeverage != 5 || parsed.RiskControl.AltcoinMaxLeverage != 5 {
		t.Fatalf("expected stored leverage to remain unchanged at safe defaults, got btc_eth=%d alt=%d", parsed.RiskControl.BTCETHMaxLeverage, parsed.RiskControl.AltcoinMaxLeverage)
	}
}

func TestToolManageStrategyRejectsFixedMinPositionSizeUpdates(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "strategy-fixed-min-position.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	a := New(nil, st, DefaultConfig(), slog.Default())

	cfg := store.GetDefaultStrategyConfig("zh")
	rawCfg, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal strategy config: %v", err)
	}
	strategy := &store.Strategy{
		ID:            "strategy-fixed-min-position",
		UserID:        "default",
		Name:          "固定最小开仓策略",
		Description:   "test",
		IsPublic:      false,
		ConfigVisible: true,
		Config:        string(rawCfg),
	}
	if err := st.Strategy().Create(strategy); err != nil {
		t.Fatalf("create strategy: %v", err)
	}

	resp := a.toolManageStrategy("default", `{"action":"update","strategy_id":"strategy-fixed-min-position","config":{"risk_control":{"min_position_size":20}}}`)
	if !strings.Contains(resp, "固定值 12 USDT") {
		t.Fatalf("expected fixed min position size rejection, got: %s", resp)
	}

	updated, err := st.Strategy().Get("default", strategy.ID)
	if err != nil {
		t.Fatalf("reload strategy: %v", err)
	}
	parsed, err := updated.ParseConfig()
	if err != nil {
		t.Fatalf("parse updated strategy config: %v", err)
	}
	if parsed.RiskControl.MinPositionSize != 12 {
		t.Fatalf("expected stored min position size to remain fixed at 12, got %v", parsed.RiskControl.MinPositionSize)
	}
}

func TestToolManageStrategyExportRedactsSensitiveConfig(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "strategy-export-redacts.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	a := New(nil, st, DefaultConfig(), slog.Default())

	cfg := store.GetDefaultStrategyConfig("zh")
	cfg.Indicators.NofxOSAPIKey = "cm_secret_export_key"
	rawCfg, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal strategy config: %v", err)
	}
	if err := st.Strategy().Create(&store.Strategy{
		ID:            "strategy-export-redacts",
		UserID:        "default",
		Name:          "导出脱敏策略",
		ConfigVisible: true,
		Config:        string(rawCfg),
	}); err != nil {
		t.Fatalf("create strategy: %v", err)
	}

	resp := a.toolManageStrategy("default", `{"action":"export","strategy_id":"strategy-export-redacts"}`)
	if strings.Contains(resp, "cm_secret_export_key") {
		t.Fatalf("expected export to redact nofxos api key, got: %s", resp)
	}
	if !strings.Contains(resp, `"action":"export"`) {
		t.Fatalf("expected export response, got: %s", resp)
	}
}

func TestStripSensitiveToolFieldsUsesProductFieldCatalog(t *testing.T) {
	input := map[string]any{
		"api_key":           "classic-secret",
		"aster_private_key": "catalog-secret",
		"nested": map[string]any{
			"lighter_private_key": "nested-secret",
			"safe":                "visible",
		},
		"safe": "visible",
	}
	cleaned, ok := stripSensitiveToolFields(input).(map[string]any)
	if !ok {
		t.Fatalf("expected cleaned map")
	}
	if _, ok := cleaned["api_key"]; ok {
		t.Fatalf("expected api_key to be stripped: %#v", cleaned)
	}
	if _, ok := cleaned["aster_private_key"]; ok {
		t.Fatalf("expected catalog-sensitive aster_private_key to be stripped: %#v", cleaned)
	}
	nested, ok := cleaned["nested"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested map: %#v", cleaned["nested"])
	}
	if _, ok := nested["lighter_private_key"]; ok {
		t.Fatalf("expected catalog-sensitive nested private key to be stripped: %#v", nested)
	}
	if cleaned["safe"] != "visible" || nested["safe"] != "visible" {
		t.Fatalf("expected non-sensitive fields to remain, got: %#v", cleaned)
	}
}

func TestToolManageStrategyImportRequiresConfirmation(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "strategy-import-confirmation.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	a := New(nil, st, DefaultConfig(), slog.Default())

	const userID int64 = 42
	importArgs := `{"action":"import","name":"确认导入","config":{"strategy_type":"ai_trading","coin_source":{"source_type":"ai500","ai500_limit":3}}}`
	resp := a.toolManageStrategyForUser("default", userID, importArgs)
	if !strings.Contains(resp, "requires_confirmation") {
		t.Fatalf("expected import to require confirmation, got: %s", resp)
	}
	var pendingResp struct {
		ConfirmationToken string `json:"confirmation_token"`
	}
	if err := json.Unmarshal([]byte(resp), &pendingResp); err != nil {
		t.Fatalf("parse pending import response: %v\n%s", err, resp)
	}
	if pendingResp.ConfirmationToken == "" {
		t.Fatalf("expected confirmation token in pending import response: %s", resp)
	}
	resp = a.toolManageStrategyForUser("default", userID, `{"action":"import","name":"确认导入","confirmed":true,"config":{"strategy_type":"ai_trading","coin_source":{"source_type":"ai500","ai500_limit":3}}}`)
	if !strings.Contains(resp, "confirmation_token") {
		t.Fatalf("expected confirmed import without token to be rejected, got: %s", resp)
	}
	resp = a.toolManageStrategyForUser("default", userID, `{"action":"import","name":"确认导入","confirmed":true,"confirmation_token":"`+pendingResp.ConfirmationToken+`","config":{"strategy_type":"ai_trading","coin_source":{"source_type":"ai500","ai500_limit":3}}}`)
	if strings.Contains(resp, `"error"`) {
		t.Fatalf("expected confirmed import to succeed, got: %s", resp)
	}
	strategies, err := st.Strategy().List("default")
	if err != nil {
		t.Fatalf("list strategies: %v", err)
	}
	var strategy *store.Strategy
	for _, candidate := range strategies {
		if candidate.Name == "确认导入" {
			strategy = candidate
			break
		}
	}
	if strategy == nil {
		t.Fatalf("expected imported strategy to exist")
	}
	parsed, err := strategy.ParseConfig()
	if err != nil {
		t.Fatalf("parse imported config: %v", err)
	}
	if parsed.StrategyType != "ai_trading" || parsed.CoinSource.AI500Limit != 3 {
		t.Fatalf("expected imported ai strategy config, got %+v", parsed)
	}
}

func TestExchangeSkillOptionSummaryMatchesManualPage(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "exchange-options.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	a := New(nil, st, DefaultConfig(), slog.Default())

	summary := a.exchangeSkillOptionSummary("zh")
	for _, expected := range []string{"Binance", "Bybit", "OKX", "Bitget", "Gate", "KuCoin", "Hyperliquid", "Aster", "Lighter", "Indodax"} {
		if !strings.Contains(summary, expected) {
			t.Fatalf("expected option %q in summary, got: %s", expected, summary)
		}
	}
	for _, hidden := range []string{"Alpaca", "Forex", "Metals"} {
		if strings.Contains(summary, hidden) {
			t.Fatalf("did not expect hidden manual-page option %q in summary: %s", hidden, summary)
		}
	}
}

func TestLoadExchangeOptionsHidesInvisibleExchangeRows(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "exchange-options-visible.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	a := New(nil, st, DefaultConfig(), slog.Default())

	if err := store.DB().Create(&store.Exchange{
		ID:           "hidden-exchange",
		UserID:       "default",
		ExchangeType: "okx",
		AccountName:  "123413",
		Name:         "OKX Futures",
		Type:         "cex",
		Enabled:      false,
	}).Error; err != nil {
		t.Fatalf("seed legacy hidden exchange: %v", err)
	}
	if _, err := st.Exchange().Create("default", "okx", "我的主力OKX账户", true, "api-test", "secret-test", "pass-test", false, "", false, "", "", "", "", "", "", 0); err != nil {
		t.Fatalf("create visible exchange: %v", err)
	}

	options := a.loadExchangeOptions("default")
	if len(options) != 1 {
		t.Fatalf("expected only the visible exchange option, got %+v", options)
	}
	if options[0].Name != "我的主力OKX账户" {
		t.Fatalf("expected visible exchange name, got %+v", options)
	}
}

func TestDescribeExchangeIncludesTypeSpecificVisibleFields(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "exchange-detail.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	a := New(nil, st, DefaultConfig(), slog.Default())

	hyperID, err := st.Exchange().Create("default", "hyperliquid", "Dex Pro", true, "hyper-api-key", "", "", true, "0xabc", true, "", "", "", "", "", "", 0)
	if err != nil {
		t.Fatalf("seed hyperliquid exchange: %v", err)
	}
	detail, ok := a.describeExchange("default", "zh", &EntityReference{ID: hyperID})
	if !ok {
		t.Fatal("expected describeExchange to resolve hyperliquid config")
	}
	for _, expected := range []string{"交易所配置“Dex Pro”详情", "交易所：hyperliquid", "账户名：Dex Pro", "API Key：true", "Hyperliquid 钱包地址：0xabc"} {
		if !strings.Contains(detail, expected) {
			t.Fatalf("expected hyperliquid detail to contain %q, got: %s", expected, detail)
		}
	}

	lighterID, err := st.Exchange().Create("default", "lighter", "Lighter Main", false, "", "", "", false, "", true, "", "", "", "wallet-1", "", "lighter-secret", 7)
	if err != nil {
		t.Fatalf("seed lighter exchange: %v", err)
	}
	detail, ok = a.describeExchange("default", "zh", &EntityReference{ID: lighterID})
	if !ok {
		t.Fatal("expected describeExchange to resolve lighter config")
	}
	for _, expected := range []string{"交易所：lighter", "Lighter 钱包地址：wallet-1", "Lighter API Key 私钥：true", "Lighter API Key Index：7"} {
		if !strings.Contains(detail, expected) {
			t.Fatalf("expected lighter detail to contain %q, got: %s", expected, detail)
		}
	}
}

func TestSkillVisibleFieldSummaryForExchangeUsesReadableNames(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "exchange-field-summary.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	a := New(nil, st, DefaultConfig(), slog.Default())

	summary := a.skillVisibleFieldSummary("default", "zh", "exchange_management", "update")
	for _, expected := range []string{"交易所类型", "账户名", "API Key", "Secret", "Passphrase", "Hyperliquid 钱包地址", "Aster 主钱包地址", "Lighter API Key 私钥", "Lighter API Key Index"} {
		if !strings.Contains(summary, expected) {
			t.Fatalf("expected field label %q in summary, got: %s", expected, summary)
		}
	}
	if strings.Contains(summary, "hyperliquid_wallet_addr") || strings.Contains(summary, "lighter_api_key_private_key") {
		t.Fatalf("field summary should use readable labels instead of raw keys: %s", summary)
	}
}

func TestExchangeCreateAsterMissingPromptUsesFrontendFieldLabels(t *testing.T) {
	a := &Agent{}
	session := skillSession{
		Name:   "exchange_management",
		Action: "create",
		Phase:  "collecting",
		Fields: map[string]string{
			"exchange_type": "aster",
			"account_name":  "我的Aster主账户",
		},
	}

	reply := a.handleExchangeCreateSkill("default", 1, "zh", "", session)
	for _, expected := range []string{"Aster 主钱包地址", "Aster API Pro 代理钱包地址", "Aster API Pro 代理钱包私钥"} {
		if !strings.Contains(reply, expected) {
			t.Fatalf("expected Aster missing prompt to contain %q, got: %s", expected, reply)
		}
	}
	for _, unexpected := range []string{"Aster User", "用户名", "API Key", "Secret"} {
		if strings.Contains(reply, unexpected) {
			t.Fatalf("Aster prompt should not contain %q, got: %s", unexpected, reply)
		}
	}
}

func TestExchangeCreateMissingFieldsAreDynamicForDexVenues(t *testing.T) {
	session := skillSession{
		Name:   "exchange_management",
		Action: "create",
		Fields: map[string]string{
			"exchange_type": "aster",
			"account_name":  "我的Aster主账户",
		},
	}

	missing := missingFieldKeysForSkillSession(session)
	for _, want := range []string{"aster_user", "aster_signer", "aster_private_key"} {
		if !containsString(missing, want) {
			t.Fatalf("expected missing Aster field %q, got %v", want, missing)
		}
	}
	for _, unexpected := range []string{"api_key", "secret_key", "passphrase"} {
		if containsString(missing, unexpected) {
			t.Fatalf("Aster create should not require CEX field %q, got %v", unexpected, missing)
		}
	}
}

func TestExchangeProductCatalogMatchesStoreCredentialRequirements(t *testing.T) {
	for _, spec := range exchangeProductCatalog {
		got := store.MissingRequiredExchangeCredentialFields(
			spec.Type,
			"",
			"",
			"",
			"",
			"",
			"",
			"",
			"",
			"",
		)
		if !reflect.DeepEqual(got, spec.CredentialFields) {
			t.Fatalf("credential fields for %s drifted from store validation: catalog=%v store=%v", spec.Type, spec.CredentialFields, got)
		}

		session := skillSession{
			Name:   "exchange_management",
			Action: "create",
			Fields: map[string]string{
				"exchange_type": spec.Type,
			},
		}
		wantSessionMissing := append([]string{"account_name"}, spec.CredentialFields...)
		if got := exchangeCreateMissingFieldKeys(session); !reflect.DeepEqual(got, wantSessionMissing) {
			t.Fatalf("agent missing fields for %s drifted from catalog: want=%v got=%v", spec.Type, wantSessionMissing, got)
		}
	}
}

func TestLLMExtractionAcceptsDexExchangeFields(t *testing.T) {
	a := &Agent{}
	session := skillSession{
		Name:   "exchange_management",
		Action: "create",
		Fields: map[string]string{
			"exchange_type": "aster",
			"account_name":  "我的Aster主账户",
		},
	}

	a.applyLLMExtractionToSkillSession("default", &session, llmFlowExtractionResult{
		Intent: "continue",
		Tasks: []llmFlowExtractionTask{{
			Skill:  "exchange_management",
			Action: "create",
			Fields: map[string]string{
				"aster_user":        "0xmain",
				"aster_signer":      "0xsigner",
				"aster_private_key": "0xprivate",
			},
		}},
	}, "zh", "主钱包 0xmain，代理钱包 0xsigner，私钥 0xprivate")

	for key, want := range map[string]string{
		"aster_user":        "0xmain",
		"aster_signer":      "0xsigner",
		"aster_private_key": "0xprivate",
	} {
		if got := fieldValue(session, key); got != want {
			t.Fatalf("expected %s=%q, got %q", key, want, got)
		}
	}
}

func TestSkillVisibleFieldSummaryForStrategyCoversManualPageFields(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "strategy-field-summary.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	a := New(nil, st, DefaultConfig(), slog.Default())

	summary := a.skillVisibleFieldSummary("default", "zh", "strategy_management", "update_config")
	for _, expected := range []string{"发布到市场", "配置可见", "交易对", "杠杆", "主周期", "多周期时间框架", "NofxOS API key", "角色定义", "自定义 Prompt"} {
		if !strings.Contains(summary, expected) {
			t.Fatalf("expected field label %q in summary, got: %s", expected, summary)
		}
	}
	if strings.Contains(summary, "最小开仓金额") {
		t.Fatalf("strategy field summary should not expose fixed min position size editing: %s", summary)
	}
}

func TestStrategyVisibleFieldSummaryUsesTargetStrategyType(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "strategy-type-field-summary.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	cfg := store.GetDefaultStrategyConfig("zh")
	cfg.StrategyType = "grid_trading"
	cfg.GridConfig = &store.GridStrategyConfig{
		Symbol:          "ETHUSDT",
		GridCount:       12,
		TotalInvestment: 1000,
		Leverage:        3,
		UseATRBounds:    true,
		ATRMultiplier:   2,
		Distribution:    "gaussian",
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal strategy config: %v", err)
	}
	strategy := &store.Strategy{
		ID:            "strategy-grid-fields",
		UserID:        "default",
		Name:          "我的第一个网格策略",
		Description:   "",
		IsPublic:      false,
		ConfigVisible: true,
		Config:        string(raw),
	}
	if err := st.Strategy().Create(strategy); err != nil {
		t.Fatalf("create strategy: %v", err)
	}
	a := New(nil, st, DefaultConfig(), slog.Default())

	session := skillSession{
		Name:   "strategy_management",
		Action: "update_config",
		TargetRef: &EntityReference{
			ID:   strategy.ID,
			Name: strategy.Name,
		},
	}
	resources := a.buildActiveSessionResources("default", session)
	if got := resources["target_strategy_type"]; got != "grid_trading" {
		t.Fatalf("expected grid strategy type in resources, got: %#v", got)
	}
	fields, ok := resources["target_editable_fields"].([]string)
	if !ok {
		t.Fatalf("expected editable field list in resources, got: %#v", resources["target_editable_fields"])
	}
	joined := strings.Join(fields, ",")
	if !strings.Contains(joined, "symbol") || strings.Contains(joined, "source_type") {
		t.Fatalf("expected grid-only editable fields in resources, got: %s", joined)
	}
}

func TestSkillVisibleFieldSummaryForTraderMatchesManualPanelFields(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "trader-field-summary.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	a := New(nil, st, DefaultConfig(), slog.Default())

	summary := a.skillVisibleFieldSummary("default", "zh", "trader_management", "update")
	for _, expected := range []string{"交易所", "模型", "策略", "扫描间隔", "全仓模式", "竞技场显示"} {
		if !strings.Contains(summary, expected) {
			t.Fatalf("expected trader field label %q in summary, got: %s", expected, summary)
		}
	}
	for _, unexpected := range []string{"名称", "初始资金", "初始余额", "杠杆", "交易对", "Prompt", "AI500", "OI Top"} {
		if strings.Contains(summary, unexpected) {
			t.Fatalf("trader field summary should stay within manual panel fields, got: %s", summary)
		}
	}
}

func TestToolUpdateTraderRejectsRenameOutsideManualPanel(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "trader-update-reject-rename.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	a := New(nil, st, DefaultConfig(), slog.Default())

	if err := st.AIModel().UpdateWithName("default", "default_deepseek", "DeepSeek", true, "sk-test-12345", "", "deepseek-chat"); err != nil {
		t.Fatalf("seed model: %v", err)
	}
	exchangeID, err := st.Exchange().Create("default", "binance", "Main", true, "api-test", "secret-test", "", false, "", false, "", "", "", "", "", "", 0)
	if err != nil {
		t.Fatalf("seed exchange: %v", err)
	}
	cfg := store.GetDefaultStrategyConfig("zh")
	rawCfg, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal strategy config: %v", err)
	}
	if err := st.Strategy().Create(&store.Strategy{
		ID:            "strategy-trader-rename",
		UserID:        "default",
		Name:          "Rename Strategy",
		Description:   "test",
		IsPublic:      false,
		ConfigVisible: true,
		Config:        string(rawCfg),
	}); err != nil {
		t.Fatalf("seed strategy: %v", err)
	}
	if err := st.Trader().Create(&store.Trader{
		ID:                  "trader-rename",
		UserID:              "default",
		Name:                "原交易员",
		AIModelID:           "default_deepseek",
		ExchangeID:          exchangeID,
		StrategyID:          "strategy-trader-rename",
		InitialBalance:      1000,
		ScanIntervalMinutes: 5,
		IsCrossMargin:       true,
		ShowInCompetition:   true,
	}); err != nil {
		t.Fatalf("seed trader: %v", err)
	}

	resp := a.toolManageTrader("default", `{"action":"update","trader_id":"trader-rename","name":"新名字"}`)
	if !strings.Contains(resp, "trader rename is not supported here") {
		t.Fatalf("expected rename rejection, got: %s", resp)
	}
}

func TestToolCreateTraderResponseHidesLegacyTraderTuningFields(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "trader-create-response-shape.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	a := New(nil, st, DefaultConfig(), slog.Default())

	if err := st.AIModel().UpdateWithName("default", "default_deepseek", "DeepSeek", true, "sk-test-12345", "", "deepseek-chat"); err != nil {
		t.Fatalf("seed model: %v", err)
	}
	exchangeID, err := st.Exchange().Create("default", "binance", "Main", true, "api-test", "secret-test", "", false, "", false, "", "", "", "", "", "", 0)
	if err != nil {
		t.Fatalf("seed exchange: %v", err)
	}
	cfg := store.GetDefaultStrategyConfig("zh")
	rawCfg, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal strategy config: %v", err)
	}
	if err := st.Strategy().Create(&store.Strategy{
		ID:            "strategy-trader-shape",
		UserID:        "default",
		Name:          "Shape Strategy",
		Description:   "test",
		IsPublic:      false,
		ConfigVisible: true,
		Config:        string(rawCfg),
	}); err != nil {
		t.Fatalf("seed strategy: %v", err)
	}

	originalFetcher := traderInitialBalanceFetcher
	traderInitialBalanceFetcher = func(exchangeCfg *store.Exchange, userID string) (float64, bool, error) {
		return 88.5, true, nil
	}
	defer func() {
		traderInitialBalanceFetcher = originalFetcher
	}()

	resp := a.toolManageTrader("default", `{"action":"create","name":"形状测试","ai_model_id":"default_deepseek","exchange_id":"`+exchangeID+`","strategy_id":"strategy-trader-shape"}`)
	if strings.Contains(resp, `"error"`) {
		t.Fatalf("expected trader create to succeed, got: %s", resp)
	}
	for _, blocked := range []string{"btc_eth_leverage", "altcoin_leverage", "trading_symbols", "custom_prompt", "system_prompt_template"} {
		if strings.Contains(resp, blocked) {
			t.Fatalf("expected trader create response to hide legacy tuning field %q, got: %s", blocked, resp)
		}
	}
}

func TestToolCreateTraderAutoReadsInitialBalanceFromExchange(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "trader-auto-balance.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	a := New(nil, st, DefaultConfig(), slog.Default())

	if err := st.AIModel().UpdateWithName("default", "default_deepseek", "DeepSeek", true, "sk-test-12345", "", "deepseek-chat"); err != nil {
		t.Fatalf("seed model: %v", err)
	}
	exchangeID, err := st.Exchange().Create("default", "binance", "Main", true, "api-test", "secret-test", "", false, "", false, "", "", "", "", "", "", 0)
	if err != nil {
		t.Fatalf("seed exchange: %v", err)
	}
	cfg := store.GetDefaultStrategyConfig("zh")
	rawCfg, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal strategy config: %v", err)
	}
	if err := st.Strategy().Create(&store.Strategy{
		ID:            "strategy-auto-balance",
		UserID:        "default",
		Name:          "Auto Balance Strategy",
		Description:   "test",
		IsPublic:      false,
		ConfigVisible: true,
		Config:        string(rawCfg),
	}); err != nil {
		t.Fatalf("seed strategy: %v", err)
	}

	originalFetcher := traderInitialBalanceFetcher
	traderInitialBalanceFetcher = func(exchangeCfg *store.Exchange, userID string) (float64, bool, error) {
		if exchangeCfg == nil || exchangeCfg.ID != exchangeID {
			t.Fatalf("unexpected exchange config passed to balance fetcher: %#v", exchangeCfg)
		}
		if userID != "default" {
			t.Fatalf("unexpected user id %q", userID)
		}
		return 4321.25, true, nil
	}
	defer func() {
		traderInitialBalanceFetcher = originalFetcher
	}()

	resp := a.toolManageTrader("default", `{"action":"create","name":"奶茶","ai_model_id":"default_deepseek","exchange_id":"`+exchangeID+`","strategy_id":"strategy-auto-balance","initial_balance":999}`)
	if strings.Contains(resp, `"error"`) {
		t.Fatalf("expected trader create to succeed, got: %s", resp)
	}

	traders, err := st.Trader().List("default")
	if err != nil {
		t.Fatalf("list traders: %v", err)
	}
	if len(traders) != 1 {
		t.Fatalf("expected one trader, got %d", len(traders))
	}
	if traders[0].InitialBalance != 4321.25 {
		t.Fatalf("expected initial balance to be auto-read from exchange, got %.2f", traders[0].InitialBalance)
	}
}

func TestDescribeStrategyIncludesManualPageSections(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "strategy-detail.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	a := New(nil, st, DefaultConfig(), slog.Default())

	cfg := store.GetDefaultStrategyConfig("zh")
	cfg.StrategyType = "grid_trading"
	cfg.GridConfig = &store.GridStrategyConfig{
		Symbol:                "BTCUSDT",
		GridCount:             12,
		TotalInvestment:       1500,
		Leverage:              4,
		UpperPrice:            120000,
		LowerPrice:            90000,
		UseATRBounds:          false,
		ATRMultiplier:         2,
		Distribution:          "gaussian",
		MaxDrawdownPct:        15,
		StopLossPct:           5,
		DailyLossLimitPct:     10,
		UseMakerOnly:          true,
		EnableDirectionAdjust: true,
		DirectionBiasRatio:    0.7,
	}
	rawCfg, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal strategy config: %v", err)
	}

	strategy := &store.Strategy{
		ID:            "strategy-detail-1",
		UserID:        "default",
		Name:          "Grid Alpha",
		Description:   "grid strategy for regression",
		IsPublic:      true,
		ConfigVisible: true,
		Config:        string(rawCfg),
	}
	if err := st.Strategy().Create(strategy); err != nil {
		t.Fatalf("create strategy: %v", err)
	}
	strategy.ConfigVisible = false
	if err := st.Strategy().Update(strategy); err != nil {
		t.Fatalf("update strategy visibility: %v", err)
	}

	detail, ok := a.describeStrategy("default", "zh", &EntityReference{ID: strategy.ID})
	if !ok {
		t.Fatal("expected describeStrategy to resolve seeded strategy")
	}
	for _, expected := range []string{
		"策略“Grid Alpha”概览",
		"发布设置：已发布到市场；配置隐藏",
		"网格参数：交易对 BTCUSDT；网格 12；总投资 1500.00；杠杆 4；分布 gaussian",
		"网格边界：上沿 120000.0000，下沿 90000.0000",
	} {
		if !strings.Contains(detail, expected) {
			t.Fatalf("expected strategy detail to contain %q, got: %s", expected, detail)
		}
	}
	for _, unexpected := range []string{
		"标的来源：",
		"NofxOS 数据：",
	} {
		if strings.Contains(detail, unexpected) {
			t.Fatalf("expected grid strategy detail not to contain AI field %q, got: %s", unexpected, detail)
		}
	}
}

func TestStrategyConfigFromToolArgsMergesPatchOntoExistingStrategy(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "strategy-preview-existing-base.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	a := New(nil, st, DefaultConfig(), slog.Default())

	cfg := store.GetDefaultStrategyConfig("zh")
	cfg.CoinSource.SourceType = "static"
	cfg.CoinSource.UseAI500 = false
	cfg.CoinSource.StaticCoins = []string{"ETHUSDT"}
	cfg.Indicators.Klines.PrimaryTimeframe = "1h"
	cfg.RiskControl.BTCETHMaxLeverage = 4
	rawCfg, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("marshal strategy config: %v", err)
	}
	if err := st.Strategy().Create(&store.Strategy{
		ID:            "strategy-preview-existing-base",
		UserID:        "default",
		Name:          "Existing Base",
		ConfigVisible: true,
		Config:        string(rawCfg),
	}); err != nil {
		t.Fatalf("create strategy: %v", err)
	}

	merged, strategyID, err := a.strategyConfigFromToolArgs("default", `{"strategy_id":"strategy-preview-existing-base","config":{"risk_control":{"btc_eth_max_leverage":7}}}`)
	if err != nil {
		t.Fatalf("merge strategy args: %v", err)
	}
	if strategyID != "strategy-preview-existing-base" {
		t.Fatalf("expected strategy id to round trip, got %q", strategyID)
	}
	if merged.CoinSource.SourceType != "static" || len(merged.CoinSource.StaticCoins) != 1 || merged.CoinSource.StaticCoins[0] != "ETHUSDT" {
		t.Fatalf("expected existing coin source to be preserved, got %+v", merged.CoinSource)
	}
	if merged.Indicators.Klines.PrimaryTimeframe != "1h" {
		t.Fatalf("expected existing primary timeframe to be preserved, got %q", merged.Indicators.Klines.PrimaryTimeframe)
	}
	if merged.RiskControl.BTCETHMaxLeverage != 7 {
		t.Fatalf("expected patch to apply leverage 7, got %v", merged.RiskControl.BTCETHMaxLeverage)
	}
}

func TestToolTestStrategyRunSkipsCandidateLookupByDefault(t *testing.T) {
	a := New(nil, nil, DefaultConfig(), slog.Default())

	resp := a.toolTestStrategyRun("default", `{"config":{"strategy_type":"ai_trading"}}`)
	if strings.Contains(resp, `"error"`) {
		t.Fatalf("expected dry run to succeed, got: %s", resp)
	}
	var payload struct {
		CandidateLookup string   `json:"candidate_lookup"`
		CandidateCount  int      `json:"candidate_count"`
		CandidateError  string   `json:"candidate_error"`
		Candidates      []string `json:"candidates"`
	}
	if err := json.Unmarshal([]byte(resp), &payload); err != nil {
		t.Fatalf("parse dry run response: %v\n%s", err, resp)
	}
	if payload.CandidateLookup != "skipped" {
		t.Fatalf("expected candidate lookup to be skipped by default, got %q in %s", payload.CandidateLookup, resp)
	}
	if payload.CandidateCount != 0 || payload.CandidateError != "" || len(payload.Candidates) != 0 {
		t.Fatalf("expected no candidates or external lookup error by default, got %+v", payload)
	}
}
