package agent

import (
	"encoding/json"
	"errors"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"nofx/store"
)

func TestUserJourneyAIServicePaymentRequiredGuidance(t *testing.T) {
	a := New(nil, nil, DefaultConfig(), slog.Default())

	msg, err := a.aiServiceFailure("zh", errors.New(`API returned error (status 402): {"error":"payment required"}`))
	if err != nil {
		t.Fatalf("aiServiceFailure returned error: %v", err)
	}

	for _, want := range []string{"当前 AI 服务调用失败", "支付", "余额不足", "充值"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("expected payment-required guidance to contain %q, got: %s", want, msg)
		}
	}
	if strings.Contains(msg, "未配置模型") && !strings.Contains(msg, "不是") {
		t.Fatalf("payment-required guidance must not say the model is missing: %s", msg)
	}
}

func TestUserJourneyAsterExchangeCreateAsksForWalletFieldsNotCEXKeys(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "user-journey-aster.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	a := New(nil, st, DefaultConfig(), slog.Default())

	session := skillSession{
		Name:   "exchange_management",
		Action: "create",
		Fields: map[string]string{
			"exchange_type": "aster",
			"account_name":  "Aster 主账户",
		},
	}
	reply := a.handleExchangeCreateSkill("default", 1, "zh", "我要接入 Aster", session)

	for _, want := range []string{"主钱包地址", "API Pro 代理钱包地址", "API Pro 代理钱包私钥"} {
		if !strings.Contains(reply, want) {
			t.Fatalf("expected Aster prompt to contain %q, got: %s", want, reply)
		}
	}
	for _, unexpected := range []string{"API Key 和 Secret", "Passphrase"} {
		if strings.Contains(reply, unexpected) {
			t.Fatalf("Aster prompt should not ask for CEX credential %q, got: %s", unexpected, reply)
		}
	}
}

func TestUserJourneyClaw402ModelCreateAsksWalletPrivateKey(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "user-journey-claw402-model.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	a := New(nil, st, DefaultConfig(), slog.Default())

	session := skillSession{
		Name:   "model_management",
		Action: "create",
		Fields: map[string]string{
			"provider": "claw402",
		},
	}
	reply := a.handleModelCreateSkill("default", 1, "zh", "我要用 claw402", session)

	for _, want := range []string{"Base", "钱包私钥", "充值"} {
		if !strings.Contains(reply, want) {
			t.Fatalf("expected claw402 prompt to contain %q, got: %s", want, reply)
		}
	}
	if strings.Contains(reply, "API Key") {
		t.Fatalf("claw402 prompt should ask for wallet private key, not generic API Key: %s", reply)
	}
}

func TestUserJourneyTraderCreateAsksNameBeforeBindings(t *testing.T) {
	a := New(nil, nil, DefaultConfig(), slog.Default())

	session := skillSession{
		Name:   "trader_management",
		Action: "create",
		Fields: map[string]string{},
	}
	reply := a.buildTraderCreateMissingPrompt("default", "zh", session, nil)

	for _, want := range []string{"取个名字", "模型 + 交易所 + 策略"} {
		if !strings.Contains(reply, want) {
			t.Fatalf("expected trader-create prompt to explain naming first with %q, got: %s", want, reply)
		}
	}
	if strings.Contains(reply, "已创建") || strings.Contains(reply, "马上为你创建") {
		t.Fatalf("missing-name prompt must not claim execution: %s", reply)
	}
}

func TestUserJourneyStrategyImportDoesNotPersistSensitivePendingConfig(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "user-journey-strategy-import-sensitive.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	a := New(nil, st, DefaultConfig(), slog.Default())

	const userID int64 = 77
	const secret = "cm_secret_should_not_be_persisted"
	resp := a.toolManageStrategyForUser("default", userID, `{
		"action":"import",
		"name":"带密钥的导入策略",
		"config":{
			"strategy_type":"ai_trading",
			"indicators":{"nofxos_api_key":"`+secret+`"},
			"coin_source":{"source_type":"ai500","ai500_limit":3}
		}
	}`)
	if strings.Contains(resp, secret) {
		t.Fatalf("pending import response leaked sensitive key: %s", resp)
	}

	raw, err := st.GetSystemConfig(pendingStrategyImportConfirmationKey(userID))
	if err != nil {
		t.Fatalf("load pending import: %v", err)
	}
	if strings.Contains(raw, secret) {
		t.Fatalf("pending import stored sensitive key in system_config: %s", raw)
	}

	var pending pendingStrategyImportConfirmation
	if err := json.Unmarshal([]byte(raw), &pending); err != nil {
		t.Fatalf("parse pending import: %v\n%s", err, raw)
	}
	configText, _ := json.Marshal(pending.Config)
	if strings.Contains(string(configText), "nofxos_api_key") {
		t.Fatalf("pending import should strip sensitive nofxos_api_key, got: %s", configText)
	}
}

func TestUserJourneyCommonProductQuestions(t *testing.T) {
	exchangePrompt := func(exchangeType string) string {
		session := skillSession{
			Name:   "exchange_management",
			Action: "create",
			Fields: map[string]string{
				"exchange_type": exchangeType,
				"account_name":  exchangeType + " 主账户",
			},
		}
		return formatExchangeCreateMissingPrompt("zh", session, exchangeCreateMissingFieldKeys(session))
	}

	a := New(nil, nil, DefaultConfig(), slog.Default())
	cases := []struct {
		question   string
		answer     string
		want       []string
		unexpected []string
	}{
		{
			question: "我想接 OKX，需要填什么？",
			answer:   exchangePrompt("okx"),
			want:     []string{"OKX", "API Key", "Secret", "Passphrase"},
		},
		{
			question:   "我想接 Hyperliquid，是不是填 secret 就行？",
			answer:     exchangePrompt("hyperliquid"),
			want:       []string{"Hyperliquid", "API Key", "钱包地址"},
			unexpected: []string{"Passphrase"},
		},
		{
			question:   "Lighter 要哪些凭证？",
			answer:     exchangePrompt("lighter"),
			want:       []string{"Lighter 钱包地址", "Lighter API Key 私钥"},
			unexpected: []string{"Secret", "Passphrase"},
		},
		{
			question: "模型 provider 都有哪些？",
			answer:   availableModelProvidersMessage("zh"),
			want:     []string{"claw402", "deepseek", "openai", "qwen", "blockrun-base", "[推荐]"},
		},
		{
			question:   "claw402 到底要填 API Key 还是钱包？",
			answer:     modelProviderCredentialGuidance("zh", "claw402"),
			want:       []string{"Base 链 EVM 钱包私钥", "Base USDC", "充值入口"},
			unexpected: []string{"custom_api_url"},
		},
		{
			question:   "交易员初始余额能不能我手动填 1000U？",
			answer:     buildSkillDomainPrimer("zh", "trader_management"),
			want:       []string{"初始余额由系统", "自动读取", "不接受手动设置", "充值"},
			unexpected: []string{"用户手动设置"},
		},
		{
			question: "创建策略第一步要选什么类型？",
			answer:   formatStrategyCreateConfigNeeded("zh", "strategy_type"),
			want:     []string{"AI 策略", "网格策略", "你帮我推荐"},
		},
		{
			question: "AI 策略怎么选币？",
			answer:   formatStrategyCreateConfigNeeded("zh", "source_type,static_coins"),
			want:     []string{"AI500", "OI Top", "OI Low", "静态币种", "稳健推荐"},
		},
		{
			question: "网格策略核心要确认哪些东西？",
			answer:   formatStrategyCreateConfigNeeded("zh", "symbol,grid_count,total_investment,leverage,use_atr_bounds 或 upper_price/lower_price"),
			want:     []string{"交易对", "投入", "杠杆", "网格密度", "价格范围", "ATR"},
		},
		{
			question: "策略试跑会不会真的下单？",
			answer:   a.toolTestStrategyRun("default", `{"config":{"strategy_type":"ai_trading"}}`),
			want:     []string{"dry_run", "No trade was executed", "no real AI call"},
		},
	}

	if len(cases) != 10 {
		t.Fatalf("expected exactly 10 user product questions, got %d", len(cases))
	}
	for _, tc := range cases {
		t.Run(tc.question, func(t *testing.T) {
			for _, want := range tc.want {
				if !strings.Contains(tc.answer, want) {
					t.Fatalf("expected answer to contain %q\nquestion: %s\nanswer: %s", want, tc.question, tc.answer)
				}
			}
			for _, unexpected := range tc.unexpected {
				if strings.Contains(tc.answer, unexpected) {
					t.Fatalf("answer should not contain %q\nquestion: %s\nanswer: %s", unexpected, tc.question, tc.answer)
				}
			}
		})
	}
}
