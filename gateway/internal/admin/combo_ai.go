package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/alltomatos/dontstop/gateway/internal/provider"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ComboAIInput describes an assistant turn.
type ComboAIInput struct {
	Description string                 `json:"description"`
	ProviderID  string                 `json:"provider_id"`
	ModelID     string                 `json:"model_id"`
	Messages    []provider.ChatMessage `json:"messages"`
}

// ComboAIDraft is a reviewable recommendation. It is never persisted by the
// AI service; the normal combo create endpoint saves it after review.
type ComboAIDraft struct {
	Message             string             `json:"message"`
	Name                string             `json:"name"`
	Description         string             `json:"description"`
	Strategy            string             `json:"strategy"`
	FailoverEnabled     bool               `json:"failover_enabled"`
	Members             []ComboMemberInput `json:"members"`
	Warnings            []string           `json:"warnings"`
	AssistantProviderID string             `json:"assistant_provider_id"`
	AssistantModelID    string             `json:"assistant_model_id"`
}

// ComboAIService drafts combos using a configured model and the real catalog.
type ComboAIService struct {
	pool      *pgxpool.Pool
	providers *ProviderService
	combos    *ComboService
}

// NewComboAIService creates the AI draft service.
func NewComboAIService(pool *pgxpool.Pool, providers *ProviderService, combos *ComboService) *ComboAIService {
	return &ComboAIService{pool: pool, providers: providers, combos: combos}
}

// Draft asks a configured model to recommend combo members from the real
// catalog. The model response is parsed and every returned member is validated
// before it reaches the console.
func (s *ComboAIService) Draft(ctx context.Context, tenantID string, input ComboAIInput) (ComboAIDraft, error) {
	description := strings.TrimSpace(input.Description)
	if description == "" && len(input.Messages) == 0 {
		return ComboAIDraft{}, fmt.Errorf("%w: description is required", ErrInvalidInput)
	}
	catalog, err := s.combos.Catalog(ctx, tenantID)
	if err != nil {
		return ComboAIDraft{}, err
	}
	if len(catalog) == 0 {
		return ComboAIDraft{}, fmt.Errorf("%w: combo catalog is empty", ErrInvalidInput)
	}
	providerID, modelID, target, err := s.assistantTarget(ctx, tenantID, input.ProviderID, input.ModelID)
	if err != nil {
		return ComboAIDraft{}, err
	}
	credential, err := s.providers.box.Decrypt(target.CredentialCipher)
	if err != nil {
		return ComboAIDraft{}, fmt.Errorf("decrypt assistant credential: %w", err)
	}
	adapter, err := provider.NewAdapter(target.Kind)
	if err != nil {
		return ComboAIDraft{}, fmt.Errorf("%w: assistant provider has no adapter", ErrInvalidInput)
	}
	system := `You are the dontstop combo architect. Return only a JSON object with keys:
message, name, description, strategy, failover_enabled, members, warnings.
strategy must be priority, round_robin or priority_round_robin.
members must be an array of {provider_id, model_id, priority, weight}. Use only provider_id and
model_id values present in the catalog below. Prefer 2-5 members. Prefer a different provider for
fallback. Do not invent models. Do not include credentials. Priority 0 is the first tier. Weight is
1-100 inside a tier. Warnings must explain unavailable choices. The user must review the draft;
you cannot save or activate it.`
	var catalogText strings.Builder
	for index, model := range catalog {
		if index >= 80 {
			break
		}
		status := "usable"
		if !model.Usable {
			status = "no active account"
		}
		caps, _ := json.Marshal(model.Capabilities)
		fmt.Fprintf(&catalogText, "- provider_id=%s provider=%s model_id=%s display=%s status=%s capabilities=%s\n",
			model.ProviderID, model.ProviderName, model.ModelID, model.DisplayName, status, string(caps))
	}
	user := "User objective:\n" + description + "\n\nCatalog:\n" + catalogText.String()
	if len(input.Messages) > 0 {
		var history strings.Builder
		start := len(input.Messages) - 6
		if start < 0 {
			start = 0
		}
		for _, message := range input.Messages[start:] {
			if message.Role == "user" || message.Role == "assistant" {
				fmt.Fprintf(&history, "\n%s: %s", message.Role, message.Content)
			}
		}
		if history.Len() > 0 {
			user += "\n\nRecent conversation:" + history.String()
		}
	}
	chatCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	response, err := adapter.Chat(chatCtx, provider.Account{BaseURL: target.BaseURL, APIKey: string(credential)}, provider.ChatRequest{
		Model: modelID,
		Messages: []provider.ChatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		MaxTokens: 1600,
	})
	if err != nil {
		return ComboAIDraft{}, fmt.Errorf("assistant request failed: %w", err)
	}
	payload, err := parseComboAIDraft(response.Content)
	if err != nil {
		return ComboAIDraft{}, fmt.Errorf("%w: assistant returned invalid JSON", ErrInvalidInput)
	}
	if payload.Strategy != "priority" && payload.Strategy != "round_robin" && payload.Strategy != "priority_round_robin" {
		payload.Strategy = "priority_round_robin"
	}
	validated, warnings, err := s.validateDraftMembers(ctx, tenantID, catalog, payload.Members)
	if err != nil {
		return ComboAIDraft{}, err
	}
	payload.Warnings = append(payload.Warnings, warnings...)
	if len(validated) == 0 {
		return ComboAIDraft{}, fmt.Errorf("%w: assistant proposed no usable members", ErrInvalidInput)
	}
	return ComboAIDraft{
		Message:             strings.TrimSpace(payload.Message),
		Name:                fallbackName(payload.Name, description),
		Description:         strings.TrimSpace(payload.Description),
		Strategy:            payload.Strategy,
		FailoverEnabled:     payload.FailoverEnabled,
		Members:             validated,
		Warnings:            payload.Warnings,
		AssistantProviderID: providerID,
		AssistantModelID:    modelID,
	}, nil
}

type comboAIPayload struct {
	Message         string `json:"message"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	Strategy        string `json:"strategy"`
	FailoverEnabled bool   `json:"failover_enabled"`
	Warnings        []string
	Members         []struct {
		ProviderID string `json:"provider_id"`
		ModelID    string `json:"model_id"`
		Priority   int    `json:"priority"`
		Weight     int    `json:"weight"`
	} `json:"members"`
}

func parseComboAIDraft(content string) (comboAIPayload, error) {
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)
	var payload comboAIPayload
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return comboAIPayload{}, err
	}
	return payload, nil
}

func (s *ComboAIService) validateDraftMembers(ctx context.Context, tenantID string, catalog []ComboCatalogModel, proposed []struct {
	ProviderID string `json:"provider_id"`
	ModelID    string `json:"model_id"`
	Priority   int    `json:"priority"`
	Weight     int    `json:"weight"`
}) ([]ComboMemberInput, []string, error) {
	available := make(map[string]ComboCatalogModel, len(catalog))
	for _, model := range catalog {
		available[model.ProviderID+"/"+model.ModelID] = model
	}
	seen := map[string]bool{}
	result := make([]ComboMemberInput, 0, len(proposed))
	warnings := make([]string, 0)
	for index, member := range proposed {
		key := member.ProviderID + "/" + member.ModelID
		model, ok := available[key]
		if !ok {
			warnings = append(warnings, "removed unknown model "+key)
			continue
		}
		if seen[key] {
			warnings = append(warnings, "removed duplicate model "+key)
			continue
		}
		if !model.Usable {
			warnings = append(warnings, "model "+key+" has no active account")
			continue
		}
		seen[key] = true
		priority := member.Priority
		if priority < 0 {
			priority = 0
		}
		weight := member.Weight
		if weight < 1 {
			weight = 1
		}
		if weight > 100 {
			weight = 100
		}
		result = append(result, ComboMemberInput{
			ProviderID:  member.ProviderID,
			ModelID:     member.ModelID,
			Priority:    priority,
			Position:    index,
			Weight:      weight,
			Enabled:     true,
			Constraints: map[string]any{},
		})
	}
	if len(result) == 0 {
		return nil, warnings, nil
	}
	validated, err := s.combos.validateMembers(ctx, tenantID, result)
	if err != nil {
		return nil, warnings, err
	}
	return validated, warnings, nil
}

func (s *ComboAIService) assistantTarget(ctx context.Context, tenantID, providerID, modelID string) (string, string, accountTarget, error) {
	if providerID != "" || modelID != "" {
		if providerID == "" || modelID == "" {
			return "", "", accountTarget{}, fmt.Errorf("%w: provider_id and model_id are required together", ErrInvalidInput)
		}
		var exists bool
		if err := s.pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM provider_models pm
				JOIN providers p ON p.id = pm.provider_id AND p.tenant_id = pm.tenant_id
				WHERE pm.tenant_id = $1 AND pm.provider_id = $2::uuid AND pm.model_id = $3
				  AND pm.active = true AND p.enabled = true
			)
		`, tenantID, providerID, modelID).Scan(&exists); err != nil {
			return "", "", accountTarget{}, fmt.Errorf("validate assistant model: %w", err)
		}
		if !exists {
			return "", "", accountTarget{}, ErrNotFound
		}
	} else {
		err := s.pool.QueryRow(ctx, `
			SELECT p.id::text, pm.model_id
			FROM provider_models pm
			JOIN providers p ON p.id = pm.provider_id AND p.tenant_id = pm.tenant_id
			WHERE pm.tenant_id = $1 AND pm.active = true AND p.enabled = true
			  AND EXISTS (SELECT 1 FROM provider_accounts pa
			              WHERE pa.tenant_id = pm.tenant_id AND pa.provider_id = pm.provider_id
			                AND pa.status = 'active' AND (pa.cooldown_until IS NULL OR pa.cooldown_until <= now()))
			ORDER BY p.name, pm.model_id
			LIMIT 1
		`, tenantID).Scan(&providerID, &modelID)
		if errors.Is(err, pgx.ErrNoRows) {
			return "", "", accountTarget{}, ErrNotFound
		}
		if err != nil {
			return "", "", accountTarget{}, fmt.Errorf("select assistant model: %w", err)
		}
	}
	targets, err := s.providers.accountTargets(ctx, tenantID, providerID)
	if err != nil || len(targets) == 0 {
		return "", "", accountTarget{}, ErrNotFound
	}
	return providerID, modelID, targets[0], nil
}

func fallbackName(name, description string) string {
	name = strings.TrimSpace(name)
	if name != "" {
		return name
	}
	description = strings.TrimSpace(description)
	if description == "" {
		return "AI combo"
	}
	runes := []rune(description)
	if len(runes) > 48 {
		runes = runes[:48]
	}
	return "AI · " + string(runes)
}
