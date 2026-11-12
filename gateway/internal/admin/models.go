package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/provider"
)

const (
	modelSyncTimeout = 15 * time.Second
	modelTestTimeout = 20 * time.Second
)

// ConfiguredModel is a persisted provider model with its routing state.
type ConfiguredModel struct {
	ModelID      string  `json:"model_id"`
	DisplayName  string  `json:"display_name"`
	Active       bool    `json:"active"`
	Capabilities any     `json:"capabilities"`
	InputPrice   float64 `json:"input_price_per_million"`
	OutputPrice  float64 `json:"output_price_per_million"`
}

// ModelUpdateResult reports one item of a bulk activation change.
type ModelUpdateResult struct {
	ModelID string `json:"model_id"`
	OK      bool   `json:"ok"`
	Error   string `json:"error,omitempty"`
}

// SyncModelsResult reports the outcome of a remote model import.
type SyncModelsResult struct {
	Imported int    `json:"imported"`
	Total    int    `json:"total"`
	Error    string `json:"error,omitempty"`
}

// ModelTestResult is a sanitized per-model connectivity check.
type ModelTestResult struct {
	ModelID   string                   `json:"model_id"`
	OK        bool                     `json:"ok"`
	Status    int                      `json:"status"`
	LatencyMS int64                    `json:"latency_ms"`
	Error     string                   `json:"error,omitempty"`
	Accounts  []AccountModelTestResult `json:"accounts"`
}

// AccountModelTestResult is the per-account outcome of a model test.
type AccountModelTestResult struct {
	AccountID   string `json:"account_id"`
	AccountName string `json:"account_name"`
	OK          bool   `json:"ok"`
	Status      int    `json:"status"`
	LatencyMS   int64  `json:"latency_ms"`
	Error       string `json:"error,omitempty"`
}

// ConfiguredModels lists the models persisted for a provider.
func (s *ProviderService) ConfiguredModels(ctx context.Context, tenantID, providerID string) ([]ConfiguredModel, error) {
	if !isUUID(providerID) {
		return nil, ErrNotFound
	}
	rows, err := s.pool.Query(ctx, `
		SELECT model_id, display_name, active, capabilities,
		       input_price_per_million::float8, output_price_per_million::float8
		FROM provider_models
		WHERE tenant_id = $1 AND provider_id = $2::uuid
		ORDER BY model_id
	`, tenantID, providerID)
	if err != nil {
		return nil, fmt.Errorf("query configured models: %w", err)
	}
	defer rows.Close()

	models := make([]ConfiguredModel, 0)
	for rows.Next() {
		var model ConfiguredModel
		if err := rows.Scan(&model.ModelID, &model.DisplayName, &model.Active, &model.Capabilities, &model.InputPrice, &model.OutputPrice); err != nil {
			return nil, fmt.Errorf("scan configured model: %w", err)
		}
		models = append(models, model)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate configured models: %w", err)
	}
	return models, nil
}

// SyncModels imports every model advertised by the provider, keeping the
// activation state of models that already exist.
func (s *ProviderService) SyncModels(ctx context.Context, tenantID, providerID string) (SyncModelsResult, error) {
	if !isUUID(providerID) {
		return SyncModelsResult{}, ErrNotFound
	}
	syncCtx, cancel := context.WithTimeout(ctx, modelSyncTimeout)
	defer cancel()

	remote, status, err := s.remoteModels(syncCtx, tenantID, providerID)
	if err != nil {
		result := SyncModelsResult{Total: len(remote)}
		if errors.Is(err, ErrModelsUnsupported) {
			result.Error = "provider does not expose a models endpoint"
		} else {
			result.Error = fmt.Sprintf("provider models request failed (status %d)", status)
		}
		return result, nil
	}

	imported := 0
	for _, model := range remote {
		id := strings.TrimSpace(model.ID)
		if id == "" {
			continue
		}
		name := strings.TrimSpace(model.Name)
		if name == "" {
			name = id
		}
		capabilities, inputPrice, outputPrice := remoteModelMetadata(model)
		encoded, err := json.Marshal(capabilities)
		if err != nil {
			return SyncModelsResult{Imported: imported, Total: len(remote)}, fmt.Errorf("encode model capabilities: %w", err)
		}
		hasMetadata := capabilities["metadata_source"] != ""
		tag, err := s.pool.Exec(ctx, `
			INSERT INTO provider_models (tenant_id, provider_id, model_id, display_name, capabilities,
			                             input_price_per_million, output_price_per_million, active)
			SELECT $1, p.id, $3, $4, $5::jsonb, $6, $7, true
			FROM providers p
			WHERE p.id = $2::uuid AND p.tenant_id = $1
			ON CONFLICT (tenant_id, provider_id, model_id) DO UPDATE
			SET display_name = CASE
					WHEN provider_models.display_name = '' THEN EXCLUDED.display_name
					ELSE provider_models.display_name
				END,
				capabilities = CASE
					WHEN $8::boolean THEN EXCLUDED.capabilities
					ELSE provider_models.capabilities
				END,
				input_price_per_million = CASE
					WHEN $8::boolean THEN EXCLUDED.input_price_per_million
					ELSE provider_models.input_price_per_million
				END,
				output_price_per_million = CASE
					WHEN $8::boolean THEN EXCLUDED.output_price_per_million
					ELSE provider_models.output_price_per_million
				END
		`, tenantID, providerID, id, name, encoded, inputPrice, outputPrice, hasMetadata)
		if err != nil {
			return SyncModelsResult{Imported: imported, Total: len(remote)}, fmt.Errorf("import provider model: %w", err)
		}
		if tag.RowsAffected() > 0 {
			imported++
		}
	}
	return SyncModelsResult{Imported: imported, Total: len(remote)}, nil
}

// SetModelsActive applies one activation state to many models, reporting each item.
func (s *ProviderService) SetModelsActive(ctx context.Context, tenantID, providerID string, models []string, active bool) ([]ModelUpdateResult, error) {
	if !isUUID(providerID) {
		return nil, ErrNotFound
	}
	ids := sanitizeModelIDs(models)
	if len(ids) == 0 {
		return nil, fmt.Errorf("%w: models are required", ErrInvalidInput)
	}
	results := make([]ModelUpdateResult, 0, len(ids))
	for _, id := range ids {
		result := ModelUpdateResult{ModelID: id}
		if err := s.SetModelActive(ctx, tenantID, providerID, id, active); err != nil {
			result.Error = "update failed"
			if errors.Is(err, ErrNotFound) {
				result.Error = "model not found"
			}
		} else {
			result.OK = true
		}
		results = append(results, result)
	}
	return results, nil
}

// remoteModelMetadata derives capabilities and prices from an optional /models
// payload. Values are never inferred from the model name; OpenRouter and
// OpenCode/OpenAI-compatible payload shapes are both supported.
func remoteModelMetadata(model RemoteModel) (map[string]any, float64, float64) {
	capabilities := map[string]any{
		"pricing_known":   false,
		"metadata_source": "",
	}
	hasMetadata := false

	input := model.Modalities.Input
	output := model.Modalities.Output
	if len(input) == 0 && len(output) == 0 {
		input = model.Architecture.InputModalities
		output = model.Architecture.OutputModalities
	}
	inputModalities := normalizeModalities(input)
	outputModalities := normalizeModalities(output)
	if len(inputModalities) > 0 || len(outputModalities) > 0 {
		capabilities["input_modalities"] = inputModalities
		capabilities["output_modalities"] = outputModalities
		capabilities["text_input"] = hasModality(inputModalities, "text")
		capabilities["text_output"] = hasModality(outputModalities, "text")
		capabilities["image_input"] = hasModality(inputModalities, "image")
		capabilities["image_output"] = hasModality(outputModalities, "image")
		capabilities["audio_input"] = hasModality(inputModalities, "audio")
		capabilities["audio_output"] = hasModality(outputModalities, "audio")
		capabilities["video_input"] = hasModality(inputModalities, "video")
		capabilities["video_output"] = hasModality(outputModalities, "video")
		capabilities["vision"] = capabilities["image_input"]
		capabilities["audio"] = capabilities["audio_input"]
		hasMetadata = true
	}

	contextWindow := model.ContextLength
	if model.Limit.Context > 0 {
		contextWindow = model.Limit.Context
	}
	if contextWindow > 0 {
		capabilities["context_window"] = contextWindow
		hasMetadata = true
	}
	maxOutput := model.TopProvider.MaxCompletionTokens
	if model.Limit.Output > 0 {
		maxOutput = model.Limit.Output
	}
	if maxOutput > 0 {
		capabilities["max_output_tokens"] = maxOutput
		hasMetadata = true
	}

	switch strings.ToLower(strings.TrimSpace(model.Type)) {
	case "chat":
		capabilities["tools"] = model.Capabilities.Tools.Supported
		capabilities["reasoning"] = model.Capabilities.Reasoning.Supported
		capabilities["stream"] = true
		capabilities["embeddings"] = false
		hasMetadata = true
	case "embedding":
		capabilities["tools"] = false
		capabilities["reasoning"] = false
		capabilities["stream"] = false
		capabilities["embeddings"] = true
		hasMetadata = true
	}
	if len(model.SupportedParameters) > 0 {
		capabilities["tools"] = containsAny(model.SupportedParameters, "tools", "tool_choice")
		capabilities["reasoning"] = containsAny(model.SupportedParameters, "reasoning", "include_reasoning", "reasoning_effort")
		hasMetadata = true
	}
	if len(model.Endpoints) > 0 {
		capabilities["stream"] = containsSuffix(model.Endpoints, "/chat/completions")
		hasMetadata = true
	}

	var inputPrice, outputPrice float64
	inputPrice, inputKnown := tokenPricePerMillion(model.Pricing.Prompt)
	outputPrice, outputKnown := tokenPricePerMillion(model.Pricing.Completion)
	if inputKnown && outputKnown {
		capabilities["pricing_known"] = true
		hasMetadata = true
	} else if model.Type != "" {
		inputPrice = model.Cost.Text.Input.Price
		outputPrice = model.Cost.Text.Output.Price
		capabilities["pricing_known"] = true
		hasMetadata = true
	}
	if hasMetadata {
		capabilities["metadata_source"] = "provider:/models"
	}
	return capabilities, inputPrice, outputPrice
}

func normalizeModalities(values []string) []string {
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		normalized := strings.ToLower(strings.TrimSpace(value))
		switch {
		case normalized == "text":
			normalized = "text"
		case strings.HasPrefix(normalized, "image"):
			normalized = "image"
		case strings.HasPrefix(normalized, "audio"):
			normalized = "audio"
		case strings.HasPrefix(normalized, "video"):
			normalized = "video"
		case normalized == "pdf":
			normalized = "pdf"
		case normalized == "file":
			normalized = "pdf"
		default:
			continue
		}
		if seen[normalized] {
			continue
		}
		seen[normalized] = true
		result = append(result, normalized)
	}
	return result
}

func hasModality(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func tokenPricePerMillion(raw string) (float64, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, false
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil || parsed < 0 {
		return 0, false
	}
	return parsed * 1_000_000, true
}

func containsAny(values []string, targets ...string) bool {
	for _, value := range values {
		for _, target := range targets {
			if strings.EqualFold(strings.TrimSpace(value), target) {
				return true
			}
		}
	}
	return false
}

// SetModelActive enables or disables a model for routing.
func (s *ProviderService) SetModelActive(ctx context.Context, tenantID, providerID, modelID string, active bool) error {
	if !isUUID(providerID) {
		return ErrNotFound
	}
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return fmt.Errorf("%w: model_id is required", ErrInvalidInput)
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE provider_models
		SET active = $4
		WHERE tenant_id = $1 AND provider_id = $2::uuid AND model_id = $3
	`, tenantID, providerID, modelID, active)
	if err != nil {
		return fmt.Errorf("update model state: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// TestModel performs a minimal chat request against every active account.
// A failure in one account does not hide success in another.
// TestModel performs a minimal chat request against one explicit account.
// Empty accountID selects the first active account; a failure never affects
// other accounts or provider metadata.
func (s *ProviderService) TestModel(ctx context.Context, tenantID, providerID, modelID, accountID string) (ModelTestResult, error) {
	if !isUUID(providerID) {
		return ModelTestResult{}, ErrNotFound
	}
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return ModelTestResult{}, fmt.Errorf("%w: model_id is required", ErrInvalidInput)
	}
	var exists bool
	if err := s.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM provider_models
			WHERE tenant_id = $1 AND provider_id = $2::uuid AND model_id = $3
		)
	`, tenantID, providerID, modelID).Scan(&exists); err != nil {
		return ModelTestResult{}, fmt.Errorf("check provider model: %w", err)
	}
	if !exists {
		return ModelTestResult{}, ErrNotFound
	}
	var target accountTarget
	var err error
	if accountID != "" {
		target, err = s.loadAccountTarget(ctx, tenantID, providerID, accountID)
	} else {
		targets, listErr := s.accountTargets(ctx, tenantID, providerID)
		if listErr != nil {
			return ModelTestResult{}, listErr
		}
		target = targets[0]
	}
	if err != nil {
		return ModelTestResult{}, err
	}
	if s.box == nil {
		return ModelTestResult{}, fmt.Errorf("credential box is not configured")
	}
	result := ModelTestResult{ModelID: modelID, Accounts: make([]AccountModelTestResult, 0, 1)}
	accountResult := AccountModelTestResult{AccountID: target.ID, AccountName: target.Name}
	credential, err := s.box.Decrypt(target.CredentialCipher)
	if err != nil {
		accountResult.Error = "credential could not be decrypted"
		result.Error = accountResult.Error
		result.Accounts = append(result.Accounts, accountResult)
		return result, nil
	}
	adapter, err := provider.NewAdapter(target.Kind)
	if err != nil {
		accountResult.Error = "provider kind has no adapter"
		result.Error = accountResult.Error
		result.Accounts = append(result.Accounts, accountResult)
		return result, nil
	}
	testCtx, cancel := context.WithTimeout(ctx, modelTestTimeout)
	defer cancel()
	started := time.Now()
	_, err = adapter.Chat(testCtx, provider.Account{BaseURL: target.BaseURL, APIKey: string(credential)}, provider.ChatRequest{
		Model:     modelID,
		Messages:  []provider.ChatMessage{{Role: "user", Content: "ping"}},
		MaxTokens: 8,
	})
	accountResult.LatencyMS = time.Since(started).Milliseconds()
	accountResult.OK = err == nil
	if err == nil {
		accountResult.Status = http.StatusOK
		result.OK = true
		result.Status = http.StatusOK
		result.LatencyMS = accountResult.LatencyMS
	} else {
		accountResult.Error = providerTestMessage(err)
		result.Error = accountResult.Error
	}
	result.Accounts = append(result.Accounts, accountResult)
	return result, nil
}

// providerTestMessage maps adapter errors to operator-facing messages.
func providerTestMessage(err error) string {
	switch {
	case errors.Is(err, provider.ErrProviderAuth):
		return "provider rejected the credential"
	case errors.Is(err, provider.ErrProviderQuota):
		return "provider account has insufficient quota"
	case errors.Is(err, provider.ErrProviderRateLimited):
		return "provider rate limit reached"
	case errors.Is(err, provider.ErrProviderUnavailable):
		return "provider is unavailable"
	case errors.Is(err, provider.ErrProviderResponse):
		return "model did not accept the request"
	default:
		return "model test failed"
	}
}

func containsSuffix(values []string, suffix string) bool {
	for _, value := range values {
		if strings.HasSuffix(strings.TrimSpace(value), suffix) {
			return true
		}
	}
	return false
}
