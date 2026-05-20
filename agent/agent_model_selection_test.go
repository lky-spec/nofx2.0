package agent

import (
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"nofx/crypto"
	"nofx/mcp"
	_ "nofx/mcp/provider"
	"nofx/store"
)

func TestLoadAIClientFromStoreUserPrefersModelWithBalance(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "agent-model-selection.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	if err := st.AIModel().UpdateWithName("default", "default_openai", "OpenAI", true, "sk-test", "", "gpt-5.2"); err != nil {
		t.Fatalf("create openai model: %v", err)
	}
	if err := st.AIModel().UpdateWithName("default", "wallet_claw402", "Claw402", true, "0x205d759b80bae1afa31a36c4afaeec0b10378c1c55e3363bcde5a1db75c747ca", "", "glm-5"); err != nil {
		t.Fatalf("create claw402 model: %v", err)
	}

	restoreWalletAddress := agentWalletAddressFromPrivateKey
	restoreBalanceQuery := agentQueryUSDCBalanceCached
	t.Cleanup(func() {
		agentWalletAddressFromPrivateKey = restoreWalletAddress
		agentQueryUSDCBalanceCached = restoreBalanceQuery
	})

	agentWalletAddressFromPrivateKey = func(privateKey string) (string, error) {
		if privateKey == "0x205d759b80bae1afa31a36c4afaeec0b10378c1c55e3363bcde5a1db75c747ca" {
			return "0xabc", nil
		}
		return "", nil
	}
	agentQueryUSDCBalanceCached = func(address string) (float64, error) {
		if address == "0xabc" {
			return 12.5, nil
		}
		return 0, nil
	}

	a := New(nil, st, DefaultConfig(), slog.Default())
	_, modelName, ok := a.loadAIClientFromStoreUser("default")
	if !ok {
		t.Fatalf("expected model selection to succeed")
	}
	if modelName != "glm-5" {
		t.Fatalf("expected model with wallet balance to be selected, got %q", modelName)
	}
}

func TestRankAgentModelCandidatesDemotesUnfundedWalletModel(t *testing.T) {
	restoreWalletAddress := agentWalletAddressFromPrivateKey
	restoreBalanceQuery := agentQueryUSDCBalanceCached
	t.Cleanup(func() {
		agentWalletAddressFromPrivateKey = restoreWalletAddress
		agentQueryUSDCBalanceCached = restoreBalanceQuery
	})

	agentWalletAddressFromPrivateKey = func(privateKey string) (string, error) {
		return "0xabc", nil
	}
	agentQueryUSDCBalanceCached = func(address string) (float64, error) {
		return 0, nil
	}

	now := time.Now()
	candidates := rankAgentModelCandidates([]*store.AIModel{
		{
			ID:        "wallet_claw402",
			Provider:  "claw402",
			Enabled:   true,
			APIKey:    crypto.EncryptedString("0x205d759b80bae1afa31a36c4afaeec0b10378c1c55e3363bcde5a1db75c747ca"),
			UpdatedAt: now,
		},
		{
			ID:        "default_deepseek",
			Provider:  "deepseek",
			Enabled:   true,
			APIKey:    crypto.EncryptedString("sk-test"),
			UpdatedAt: now.Add(-time.Hour),
		},
	})

	if len(candidates) != 2 {
		t.Fatalf("expected two candidates, got %d", len(candidates))
	}
	if candidates[0].model.ID != "default_deepseek" {
		t.Fatalf("expected API-key model before unfunded wallet model, got %q", candidates[0].model.ID)
	}
}

func TestLoadAIClientFromStoreUserUsesProviderDefaultModelWhenCustomNameEmpty(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "agent-provider-default-model.db")
	st, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	const modelID = "default_deepseek"
	if err := st.AIModel().UpdateWithName("default", modelID, "DeepSeek", true, "sk-test", "", ""); err != nil {
		t.Fatalf("create deepseek model: %v", err)
	}

	a := New(nil, st, DefaultConfig(), slog.Default())
	client, modelName, ok := a.loadAIClientFromStoreUser("default")
	if !ok {
		t.Fatalf("expected model selection to succeed")
	}
	if modelName != mcp.DefaultDeepSeekModel {
		t.Fatalf("expected provider default model %q, got %q", mcp.DefaultDeepSeekModel, modelName)
	}
	if modelName == modelID {
		t.Fatalf("model config id %q must not be used as the upstream model name", modelID)
	}
	embedder, ok := client.(mcp.ClientEmbedder)
	if !ok || embedder.BaseClient() == nil {
		t.Fatalf("expected provider client to expose base client")
	}
	if got := embedder.BaseClient().Model; got != mcp.DefaultDeepSeekModel {
		t.Fatalf("expected base client model %q, got %q", mcp.DefaultDeepSeekModel, got)
	}
}
